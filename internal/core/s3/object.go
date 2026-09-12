package s3

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/avatar31/omashu"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	objectio "github.com/avatar31/halmidi/internal/persistent/object_io"
	"github.com/avatar31/halmidi/internal/persistent/storage/erasure"
	"github.com/avatar31/halmidi/utils"
)

type ListObjectOptions struct {
	Prefix     string
	Delimiter  string
	StartAfter string
	MaxKeys    int
}

type GetObjectOptions struct {
	VersionId  string
	RangeStart *int64
	RangeEnd   *int64
}

type PutObjectOptions struct {
	LockMode      string
	RetainUntil   string
	ContentType   string
	ContentLength int64

	// Copy Object Options
	CopySourceBucket string
	CopySourceKey    string
}

type ObjectService struct {
	log    logger.Logger
	bdb    *omashu.DistributedBadger
	bucket *models.BucketMeta
}

func NewObjectServiceWithDB(ctx context.Context, db *omashu.DistributedBadger, bucketName string) (*ObjectService, error) {
	bucket, err := NewBucketServiceWithDB(ctx, db).GetBucketEntity(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return &ObjectService{
		log:    logger.GetLogger(ctx).WithField("bucket", bucket.meta.Name),
		bdb:    db,
		bucket: bucket.GetMetadata(ctx),
	}, nil
}

func newObjectServiceInt(log logger.Logger, db *omashu.DistributedBadger, bucket *models.BucketMeta) *ObjectService {
	return &ObjectService{
		log:    log,
		bdb:    db,
		bucket: bucket,
	}
}

func NewObjectService(ctx context.Context, bucket string) (*ObjectService, error) {
	return NewObjectServiceWithDB(ctx, dbstore.GetDBStore(ctx), bucket)
}

func (s *ObjectService) IsExist(ctx context.Context, key string) bool {
	return s.bdb.Exists(ctx, dbstore.GetObjectDBKeyWithNS(s.bucket.Name, key))
}

// TODO: Test EncodingType
func (s *ObjectService) ListObjects(ctx context.Context, opts *ListObjectOptions) ([]*models.ObjectMeta, []string,
	string, error) {
	commonPrefixes := []string{}
	commonPrefixMap := make(map[string]bool)
	result := []*models.ObjectMeta{}

	startAfter := ""
	if opts.StartAfter != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.StartAfter)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, nil, "", s3common.GetInvalidArgumentS3Error(s.bucket.Name, "Invalid continuation-token")
		}
		startAfter = dbstore.GetObjectDBKeyWithNS(s.bucket.Name, decodedMarker)
	}

	dbPrefix := dbstore.GetObjectDBKeyWithNS(s.bucket.Name, opts.Prefix)
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxKeys, func(k, v []byte) bool {
		meta, err := models.DecodeObjectMeta(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from object list")
			return false
		}

		// Handle delimiter logic for common prefixes
		if opts.Delimiter != "" && strings.HasPrefix(meta.Key, opts.Prefix) {
			remaining := strings.TrimPrefix(meta.Key, opts.Prefix)
			delimiterIndex := strings.Index(remaining, opts.Delimiter)
			if delimiterIndex != -1 {
				// Found delimiter, extract common prefix
				commonPrefix := opts.Prefix + remaining[:delimiterIndex+len(opts.Delimiter)]
				if !commonPrefixMap[commonPrefix] {
					commonPrefixes = append(commonPrefixes, commonPrefix)
					commonPrefixMap[commonPrefix] = true
				}
				return true
			}
		}

		result = append(result, meta)
		return true
	})
	if err != nil {
		s.log.WithError(err).Error("Error while reading object details from db")
		return nil, nil, "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	if nextCursor != "" {
		// Trim the object namespace from the cursor before returning
		nextCursor = strings.TrimPrefix(nextCursor, dbstore.GetObjectDBKeyWithNS(s.bucket.Name, ""))
	}

	return result, commonPrefixes, secure.EncodeSignedMarker(nextCursor), nil
}

// TODO: P0: Why do we need 2 handlers for listing objects and listing objects with prefix?
// Can we merge them into one handler?
func (s *ObjectService) ListObjectsWithPrefix(ctx context.Context, prefix string) ([]*models.ObjectMeta, error) {
	result := []*models.ObjectMeta{}
	dbPrefix := dbstore.GetObjectDBKeyWithNS(s.bucket.Name, "")
	_, err := s.bdb.IterateByPrefix(ctx, dbPrefix, "", nil, func(k, v []byte) bool {
		meta, err := models.DecodeObjectMeta(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from object list")
			return false
		}

		if strings.HasPrefix(meta.Key, prefix) {
			result = append(result, meta)
		}
		return true
	})

	if err != nil {
		s.log.WithError(err).Error("Error while iterating objects list in db.")
		return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	return result, nil
}

func (s *ObjectService) GetObjectEntity(ctx context.Context, objectKey string) (*ObjectEntity, error) {
	entity := &ObjectEntity{
		log:    s.log.WithField("objectKey", objectKey),
		bdb:    s.bdb,
		bucket: s.bucket,
	}

	err := entity.fetchMetadataFromDB(ctx, objectKey)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *ObjectService) PutObject(ctx context.Context, objectKey string, body io.Reader,
	opts PutObjectOptions) (string, string, error) {
	if !isValidObjectKey(objectKey) {
		return "", "", s3common.GetInvalidKeyS3Error(s.bucket.Name)
	}

	s.log = s.log.WithField("objectKey", objectKey)
	s.log.Info("Uploading object")

	e, err := s.GetObjectEntity(ctx, objectKey)
	if err != nil {
		s.log.Info("Object doesn't exist. Creating new object.")
		e = &ObjectEntity{
			log:    s.log,
			bdb:    s.bdb,
			bucket: s.bucket,
			meta:   models.NewObjectMetadata(s.bucket, objectKey),
		}
	}

	versionMeta, versionName, versionId := NewObjectVersionService(ctx, s.bucket, e.meta).
		CreateNewObjectVersion(ctx)

	e.meta.ContentType = opts.ContentType
	if versionMeta == nil {
		if err := e.meta.ApplyObjectLocking(ctx, e.bucket, opts.LockMode, opts.RetainUntil); err != nil {
			return "", "", err
		}
	} else {
		e.log = e.log.WithField("versionId", versionId)
		if err := versionMeta.ApplyObjectLocking(ctx, e.bucket, opts.LockMode, opts.RetainUntil); err != nil {
			return "", "", err
		}
	}

	input := objectio.ObjectInput{
		Data:       body,
		ContentLen: opts.ContentLength,
		Bucket:     e.bucket.Name,
		Id:         e.meta.Uuid,
		Key:        e.meta.Key,
		Version:    versionName,
	}

	fileInfo, err := objectio.NewObjectIO().Write(ctx, input)
	if err != nil {
		e.log.WithError(err).Error("Error while writing data to disk")
		return "", "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	err = e.updateObjectMeta(ctx, fileInfo, versionMeta)
	if err != nil {
		e.log.WithError(err).Error("Error while updating metadata after writing file data to disk")
		return "", "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return fileInfo.Checksum, versionId, nil
}

func (s *ObjectService) CopyObject(ctx context.Context, objectKey string, opts PutObjectOptions) (string,
	string, error) {
	log := s.log.WithField("objectKey", objectKey)
	log.Infof("Copying object from %s/%s -> %s/%s", opts.CopySourceBucket, opts.CopySourceKey, s.bucket.Name,
		objectKey)

	var srcObjectEntity *ObjectEntity
	if s.bucket.Name != opts.CopySourceBucket {
		service := NewBucketService(ctx)
		srcBucketEntity, err := service.GetBucketEntity(ctx, opts.CopySourceBucket)
		if err != nil {
			return "", "", err
		}

		srcObjectEntity, err = service.GetObjectService(ctx, srcBucketEntity).GetObjectEntity(ctx, opts.CopySourceKey)
		if err != nil {
			log.WithError(err).Errorf("Error while fetching source object with key %s", opts.CopySourceKey)
			return "", "", err
		}
	} else {
		var err error
		srcObjectEntity, err = s.GetObjectEntity(ctx, opts.CopySourceKey)
		if err != nil {
			log.WithError(err).Errorf("Error while fetching source object with key %s", opts.CopySourceKey)
			return "", "", err
		}
	}

	destObjectEntity, err := s.GetObjectEntity(ctx, objectKey)
	if err != nil {
		destObjectEntity = &ObjectEntity{
			log:    s.log.WithField("objectKey", objectKey),
			bdb:    s.bdb,
			bucket: s.bucket,
			meta:   models.NewObjectMetadata(s.bucket, objectKey),
		}
	} else {
		log.Infof("Object with key already found in destination")
	}

	versionMeta, versionName, versionId := NewObjectVersionService(ctx, s.bucket, destObjectEntity.meta).
		CreateNewObjectVersion(ctx)

	destObjectEntity.meta.ContentType = srcObjectEntity.meta.ContentType
	if versionMeta == nil {
		err := destObjectEntity.meta.ApplyObjectLocking(ctx, destObjectEntity.bucket, opts.LockMode, opts.RetainUntil)
		if err != nil {
			return "", "", err
		}
	} else {
		destObjectEntity.log = destObjectEntity.log.WithField("versionId", versionId)
		err := versionMeta.ApplyObjectLocking(ctx, destObjectEntity.bucket, opts.LockMode, opts.RetainUntil)
		if err != nil {
			return "", "", err
		}
	}

	destInput := objectio.ObjectInput{
		Bucket:  destObjectEntity.bucket.Name,
		Id:      destObjectEntity.meta.Uuid,
		Key:     destObjectEntity.meta.Key,
		Version: versionName,
	}
	srcInput := objectio.ObjectInput{
		Bucket:       opts.CopySourceBucket,
		Id:           srcObjectEntity.meta.Uuid,
		Key:          srcObjectEntity.meta.Key,
		Shards:       convertToErasureShards(srcObjectEntity.meta.Shards),
		DataShards:   srcObjectEntity.meta.DataShards,
		ParityShards: srcObjectEntity.meta.ParityShards,
		TotalSize:    int64(srcObjectEntity.meta.Size),
		Checksum:     srcObjectEntity.meta.Etag,
	}
	fileInfo, err := objectio.NewObjectIO().Copy(ctx, srcInput, destInput)
	if err != nil {
		log.WithError(err).Error("Error while copying data into disk")
		return "", "", s3common.GetInternalErrorS3Error(destObjectEntity.bucket.Name)
	}

	err = destObjectEntity.updateObjectMeta(ctx, fileInfo, versionMeta)
	if err != nil {
		log.WithError(err).Error("Error while updating metadata after writing file data to disk")
		return "", "", s3common.GetInternalErrorS3Error(destObjectEntity.bucket.Name)
	}

	return fileInfo.Checksum, versionId, nil
}

// TODO: P0: Check versioning and object locking and all
func (s *ObjectService) DeleteObjects(ctx context.Context, keys []string) ([]string, error) {
	deletedKeys := []string{}
	objIO := objectio.NewObjectIO()

	processedKeysCount := 0
	totalKeysCount := len(keys)

	// TODO: P0: Make this parallel
	for processedKeysCount <= totalKeysCount {
		key := keys[processedKeysCount]
		err := s.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
			log := s.log.WithField("key", key)
			dbKey := dbstore.GetObjectDBKeyWithNS(s.bucket.Name, key)
			metaBytes, found, err := s.bdb.GetWithTxn(ctx, txn, dbKey)
			if err != nil {
				log.WithError(err).Error("Error while getting object metadata, so skipping delete")
				return err
			}
			if !found {
				s.log.WithField("key", key).Info("Object not found, so skipping delete")
				return nil
			}

			meta, err := models.DecodeObjectMeta(metaBytes)
			if err != nil {
				log.WithError(err).Errorf("Error while decoding object metadata for key %s, so skipping delete", key)
				return err
			}

			err = s.bdb.DeleteWithTxn(ctx, txn, dbKey)
			if err != nil {
				log.WithError(err).Error("Error while deleting object metadata, so skipping delete")
				return err
			}

			err = objIO.Remove(ctx, objectio.ObjectInput{
				Bucket:       s.bucket.Name,
				Key:          meta.Key,
				Id:           meta.Uuid,
				DoCleanup:    true,
				Shards:       convertToErasureShards(meta.Shards),
				DataShards:   meta.DataShards,
				ParityShards: meta.ParityShards,
				TotalSize:    int64(meta.Size),
				Checksum:     meta.Etag,
			})
			if err != nil {
				log.WithError(err).Error("Error while deleting object from disk during bulk delete")
			}

			return nil
		})
		processedKeysCount++
		if err != nil {
			continue
		}
		deletedKeys = append(deletedKeys, key)
	}

	return deletedKeys, nil
}

type ObjectEntity struct {
	log    logger.Logger
	bdb    *omashu.DistributedBadger
	bucket *models.BucketMeta
	dbKey  string
	meta   *models.ObjectMeta
}

func (e *ObjectEntity) GetMetadata(ctx context.Context) *models.ObjectMeta {
	return e.meta
}

func (e *ObjectEntity) fetchMetadataFromDB(ctx context.Context, objectKey string) error {
	e.dbKey = dbstore.GetObjectDBKeyWithNS(e.bucket.Name, objectKey)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while fetching bucket metadata from db")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}
	if !ok {
		return s3common.GetNoSuchKeyS3Error(e.bucket.Name)
	}

	data, err := models.DecodeObjectMeta(b)
	if err != nil {
		e.log.WithError(err).Error("Error while decoding object metadata from db")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	e.meta = data
	return nil
}

func (e *ObjectEntity) GetTags(ctx context.Context) (*models.TagMap, error) {
	return e.meta.Tags, nil
}

func (e *ObjectEntity) UpdateTags(ctx context.Context, tags *models.TagMap) error {
	delta := &models.ObjectMeta{}
	if err := delta.UpdateTags(e.bucket.Name, tags); err != nil {
		return err
	}

	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating object metadata")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return nil
}

func (e *ObjectEntity) GetObjectLegaHold(ctx context.Context, versionId string) (s3common.LegalHoldStatus, error) {
	if versionId != "" {
		if !e.bucket.IsVersioningEnabled() {
			return "", s3common.GetInvalidRequestS3Error(e.bucket.Name, "Versioning is not enabled in the bucket")
		}

		versionEntity, err := NewObjectVersionService(ctx, e.bucket, e.meta).GetObjectVersionEntity(ctx, versionId)
		if err != nil {
			return "", err
		}

		return versionEntity.GetObjectVersionLegaHold(ctx)
	}

	legalHold := s3common.LegalHoldStatusOff
	if e.meta.ObjectLock != nil && e.meta.ObjectLock.LegalHoldStatus != nil {
		legalHold = s3common.LegalHoldStatus(*e.meta.ObjectLock.LegalHoldStatus)
	}

	return legalHold, nil
}

func (e *ObjectEntity) UpdateObjectLegaHold(ctx context.Context, versionId string,
	status s3common.LegalHoldStatus) error {
	if !e.bucket.IsObjectLockingEnabled() {
		return s3common.GetInvalidRequestS3Error(e.bucket.Name, "Bucket is not enabled for Object Lock.")
	}

	if versionId != "" {
		if !e.bucket.IsVersioningEnabled() {
			return s3common.GetInvalidArgumentS3Error(e.bucket.Name, "Bucket is not enabled for Versioning.")
		}

		versionEntity, err := NewObjectVersionService(ctx, e.bucket, e.meta).GetObjectVersionEntity(ctx, versionId)
		if err != nil {
			return err
		}

		return versionEntity.UpdateObjectVersionLegaHold(ctx, status)
	}

	delta := &models.ObjectMeta{}
	delta.UpdateLegalHoldStatus(status)

	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating object legal hold status")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return nil
}

// | Versioning | `versionId` Supplied     | Behavior                           |
// | ---------- | ------------------------ | ---------------------------------- |
// | Disabled   | ❌ Not supplied          | ✅ Returns the object (if exists) |
// | Disabled   | ✅ Supplied (any value)  | ❌ Returns `400 InvalidArgument`  |
// | Enabled    | ❌ Not supplied          | ✅ Returns the object (if exists) |
// | Enabled    | ✅ Supplied (valid Id)   | ✅ Returns specified version      |
// | Enabled    | ✅ Supplied (invalid Id) | ❌ Returns `NoSuchVersion`        |
// | Suspended  | ❌ Not supplied          | ✅ Returns the object (if exists) |
// | Suspended  | ✅ Supplied (valid Id)   | ✅ Returns the object (if exists) |
// | Suspended  | ✅ Supplied (invalid Id) | ❌ Returns `404 NoSuchVersion`    |
func (e *ObjectEntity) GetObject(ctx context.Context, opts GetObjectOptions) (*models.ObjectMeta, *os.File, error) {
	if opts.VersionId != "" {
		if e.bucket.Versioning == nil {
			return nil, nil, s3common.GetInvalidArgumentS3Error(e.bucket.Name, "Versioning not enabled in the bucket.")
		}

		versionEntity, err := NewObjectVersionService(ctx, e.bucket, e.meta).
			GetObjectVersionEntity(ctx, opts.VersionId)
		if err != nil {
			return nil, nil, err
		}

		return versionEntity.GetObjectVersion(ctx, opts)
	}

	if e.bucket.IsVersioningEnabled() {
		versionService := NewObjectVersionService(ctx, e.bucket, e.meta)
		allVersions, err := versionService.ListVersions(ctx)
		if err != nil || len(allVersions) == 0 {
			return nil, nil, s3common.GetNoSuchKeyS3Error(e.bucket.Name)
		}

		versionMeta := allVersions[len(allVersions)-1]
		return versionService.GetObjectVersionEntityWithMetadata(ctx, versionMeta).
			GetObjectVersion(ctx, opts)
	}

	input := objectio.ObjectInput{
		Bucket:       e.bucket.Name,
		Id:           e.meta.Uuid,
		Key:          e.meta.Key,
		Shards:       convertToErasureShards(e.meta.Shards),
		DataShards:   e.meta.DataShards,
		ParityShards: e.meta.ParityShards,
		TotalSize:    int64(e.meta.Size),
		Checksum:     e.meta.Etag,
	}

	var file *os.File
	var err error
	if opts.RangeStart != nil && opts.RangeEnd != nil {
		file, err = objectio.NewObjectIO().ReadBytesRange(ctx, input, *opts.RangeStart, *opts.RangeEnd)
	} else {
		file, err = objectio.NewObjectIO().Read(ctx, input)
	}
	if err != nil {
		e.log.WithError(err).Errorf("Error while reading object from disk")
		return nil, nil, s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return e.meta, file, nil
}

func (e *ObjectEntity) DeleteObject(ctx context.Context, versionId string, bypassGovRetention bool) (string, error) {
	// case 1: Versioning was never enabled
	// 		Remove folder .../{bucket}/{objectId}
	// case 2: Versioning is enabled
	// 		case 2.1: Version is not specified
	// 				Add Delete Marker (No file/dir to be deleted)
	// 		case 2.2: Specified Version to delete
	// 			case 2.2.1: Specified version is just a delete marker
	// 				Remove Delete Marker Version (No file/dir to be deleted)
	// 			case 2.2.2: Specified version is actual file
	// 				case 2.2.2.1: This is the only version in object
	// 					Follow case 1
	// 				case 2.2.2.2: There are still some more versions found
	// 					Remove folder .../{bucket}/{objectId}/{version}
	// case 3: Versioning is suspended
	// 		case 3.1: Specified Version to delete
	// 			Follow case 2.2
	// 		case 3.2: Version is not specified
	// 			case 3.2.1: There are still few versions found
	// 				case 3.2.1.1: Non delete markers versions found
	// 					Remove file .../{bucket}/{objectId}/{filename}
	// 				case 3.2.1.2: All are delete marker versions
	// 					Remove folder .../{bucket}/{objectId}
	// 			case 3.2.2: There are no more versions found
	// 				Follow case 1

	// case 1: Versioning was never enabled
	if !e.bucket.IsVersioningInitialized() {
		if versionId != "" {
			return "", s3common.GetInvalidArgumentS3Error(e.bucket.Name, "Versioning is not enabled in the bucket")
		}

		return e.deleteObjectWithNoVersioning(ctx, bypassGovRetention)
	}

	// case 2: Versioning is enabled
	if e.bucket.IsVersioningEnabled() {
		return e.deleteObjectWithVersioningEnabled(ctx, versionId, bypassGovRetention)
	}

	// case 3: Versioning is suspended
	return e.deleteObjectWithVersioningSuspended(ctx, versionId, bypassGovRetention)
}

func (e *ObjectEntity) deleteObjectWithNoVersioning(ctx context.Context, bypassGovRetention bool) (string, error) {
	if err := e.meta.CanBeDeleted(e.bucket.Name, bypassGovRetention); err != nil {
		return "", err
	}

	return e.cleanupObject(ctx)
}

func (e *ObjectEntity) cleanupObject(ctx context.Context) (string, error) {
	e.log.Info("Deleting object.")

	input := objectio.ObjectInput{
		Bucket:       e.bucket.Name,
		Key:          e.meta.Key,
		Id:           e.meta.Uuid,
		DoCleanup:    true,
		Shards:       convertToErasureShards(e.meta.Shards),
		DataShards:   e.meta.DataShards,
		ParityShards: e.meta.ParityShards,
		TotalSize:    int64(e.meta.Size),
		Checksum:     e.meta.Etag,
	}
	err := objectio.NewObjectIO().Remove(ctx, input)
	if err != nil {
		e.log.WithError(err).Error("Error while deleting object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, e.dbKey)
		if err != nil {
			return err
		}

		// update bucket object count
		err = txn.DecrBy(ctx, dbstore.GetBucketObjectCountDBKeyWithNS(e.bucket.Name), 1)
		if err != nil {
			return err
		}

		// update bucket space usage
		err = txn.DecrBy(ctx, dbstore.GetBucketSpaceUsageDBKeyWithNS(e.bucket.Name), e.meta.Size)
		return err
	})

	if err != nil {
		e.log.WithError(err).Error("Error while deleting metadata for object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return "", nil
}

func (e *ObjectEntity) deleteObjectWithVersioningEnabled(ctx context.Context, versionId string,
	bypassGovRetention bool) (string, error) {
	// Keep on adding new delete markers if user haven't specified version
	if versionId == "" {
		// case 2.1: Version is not specified. So, just add delete marker
		return e.addDeleteMarker(ctx, bypassGovRetention)
	}

	// case 2.2: Specified Version to delete
	versionEntity, err := NewObjectVersionService(ctx, e.bucket, e.meta).GetObjectVersionEntity(ctx, versionId)
	if err != nil {
		return "", err
	}

	versionMeta := versionEntity.GetMetadata(ctx)
	if err := versionMeta.CanBeDeleted(e.bucket.Name, bypassGovRetention); err != nil {
		return "", err
	}

	log := e.log.WithField("versionId", versionId)

	// Delete only version metadata if versionId refers to delete marker
	if versionMeta.IsDeleteMarker != nil && *versionMeta.IsDeleteMarker {
		// case 2.2.1: Specified version is just a delete marker. So, remove only delete marker from metadata DB
		log.Info("Deleting delete marker")
		err := dbstore.GetDBStore(ctx).Delete(ctx, dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, versionMeta.Id))
		if err != nil {
			log.WithError(err).Errorf("Error while deleting metadata for object")
			return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
		}

		return versionId, nil
	}

	// case 2.2.2: Specified version is actual file
	availableVersionsCount := e.bdb.Count(ctx, dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, ""))
	if availableVersionsCount == 1 {
		// case 2.2.2.1: This is the only version in object. So, cleanup object details
		return e.cleanupObject(ctx)
	}

	log.Infof("Deleting object version: %s", versionMeta.Name)

	// case 2.2.2.2: There are still some more versions found. So, delete specified version
	input := objectio.ObjectInput{
		Bucket:       e.bucket.Name,
		Key:          e.meta.Key,
		Id:           e.meta.Uuid,
		Version:      versionMeta.Name,
		Shards:       convertToErasureShards(versionMeta.Shards),
		DataShards:   versionMeta.DataShards,
		ParityShards: versionMeta.ParityShards,
		TotalSize:    int64(versionMeta.Size),
		Checksum:     versionMeta.Etag,
	}
	err = objectio.NewObjectIO().Remove(ctx, input)
	if err != nil {
		log.WithError(err).Error("Error while deleting object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, versionMeta.Id))
		if err != nil {
			return err
		}

		// update bucket space usage
		err = txn.DecrBy(ctx, dbstore.GetBucketSpaceUsageDBKeyWithNS(e.bucket.Name), versionMeta.Size)
		return err
	})

	if err != nil {
		log.WithError(err).Error("Error while deleting metadata for object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return versionId, nil
}

func (e *ObjectEntity) deleteObjectWithVersioningSuspended(ctx context.Context, versionId string,
	bypassGovRetention bool) (string, error) {
	if versionId != "" {
		// case 3.1: Specified Version to delete
		return e.deleteObjectWithVersioningEnabled(ctx, versionId, bypassGovRetention)
	}

	// case 3.2: Version is not specified
	versionService := NewObjectVersionService(ctx, e.bucket, e.meta)
	allVersions, err := versionService.ListVersions(ctx)
	if err != nil {
		return "", err
	}

	versionsCount := len(allVersions)
	if versionsCount == 0 {
		// We shouldn't come here
		e.log.Warning("No Versions found for object but object meta still exist in DB")
		return "", s3common.GetNoSuchKeyS3Error(e.bucket.Name)
	}

	foundNoVersionObject := false
	latestVersion := allVersions[versionsCount-1]
	if id := latestVersion.Id; id == s3common.NO_VERSION {
		foundNoVersionObject = true
	}

	if !foundNoVersionObject {
		return "", s3common.GetNoSuchKeyS3Error(e.bucket.Name)
	}

	if versionsCount == 1 {
		// case 3.2.2: There are no more versions found
		return e.cleanupObject(ctx)
	}

	// case 3.2.1: There are still few versions found
	if err := latestVersion.CanBeDeleted(e.bucket.Name, bypassGovRetention); err != nil {
		return "", err
	}

	cleanup := true
	for _, v := range allVersions {
		if v.Id != s3common.NO_VERSION && v.IsDeleteMarker == nil {
			// case 3.2.1.1: Non delete markers versions found
			cleanup = false
			break
		}
	}

	e.log.Info("Deleting object.")
	input := objectio.ObjectInput{
		Bucket:       e.bucket.Name,
		Key:          e.meta.Key,
		Id:           e.meta.Uuid,
		DoCleanup:    cleanup,
		Shards:       convertToErasureShards(e.meta.Shards),
		DataShards:   e.meta.DataShards,
		ParityShards: e.meta.ParityShards,
		TotalSize:    int64(e.meta.Size),
		Checksum:     e.meta.Etag,
	}
	err = objectio.NewObjectIO().Remove(ctx, input)
	if err != nil {
		e.log.WithError(err).Error("Error while deleting object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, latestVersion.Id))
		if err != nil {
			return err
		}

		// update bucket space usage
		err = txn.DecrBy(ctx, dbstore.GetBucketSpaceUsageDBKeyWithNS(e.bucket.Name), latestVersion.Size)
		return err
	})

	if err != nil {
		e.log.WithError(err).Error("Error while deleting metadata for object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return "", nil
}

func (e *ObjectEntity) addDeleteMarker(ctx context.Context, bypassGovRetention bool) (string, error) {
	if err := e.meta.CanBeDeleted(e.bucket.Name, bypassGovRetention); err != nil {
		return "", err
	}

	version := e.meta.ApplyVersioning(e.bucket, true)
	log := e.log.WithField("versionId", version.Name)
	log.Info("Adding delete marker")

	bdb := dbstore.GetDBStore(ctx)
	updatedObject, err := e.MergeDelta(ctx, &models.ObjectMeta{NextVersionNumber: e.meta.NextVersionNumber})
	if err != nil {
		return "", err
	}
	err = bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		// Update object metadata
		err = txn.Set(ctx, e.dbKey, updatedObject)
		if err != nil {
			return err
		}

		// Update object versioning metadata
		dbKey := dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, version.Id)
		versionBytes, err := version.Encode()
		if err != nil {
			return err
		}
		return txn.Set(ctx, dbKey, versionBytes)
	})

	if err != nil {
		log.WithError(err).Errorf("Error while updating metadata for version for object")
		return "", s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return version.Id, nil
}

func (e *ObjectEntity) updateObjectMeta(ctx context.Context, fileInfo *erasure.FileInfo,
	version *models.ObjectVersion) error {
	delta := &models.ObjectMeta{
		Size:         uint64(fileInfo.TotalSize),
		Etag:         fileInfo.Checksum,
		LastModified: utils.ConvertTimeToString(fileInfo.CreatedAt),
		DataShards:   fileInfo.DataShards,
		ParityShards: fileInfo.ParityShards,
		Shards:       convertToDbShards(fileInfo.Shards),
	}

	updatedObject, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	return e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		// Update object metadata
		err := txn.Set(ctx, e.dbKey, updatedObject)
		if err != nil {
			return err
		}

		// update bucket object count
		err = txn.IncrBy(ctx, dbstore.GetBucketObjectCountDBKeyWithNS(e.bucket.Name), 1)
		if err != nil {
			return err
		}

		// update bucket space usage
		err = txn.IncrBy(ctx, dbstore.GetBucketSpaceUsageDBKeyWithNS(e.bucket.Name), e.meta.Size)
		if err != nil {
			return err
		}

		if version == nil {
			return nil
		}

		// Update object versioning metadata
		version.Size = delta.Size
		version.Etag = delta.Etag
		version.LastModified = delta.LastModified
		version.DataShards = fileInfo.DataShards
		version.ParityShards = fileInfo.ParityShards
		version.Shards = convertToDbShards(fileInfo.Shards)

		b, err := version.Encode()
		if err != nil {
			return err
		}

		dbKey := dbstore.GetObjectVersioningDBKeyWithNS(e.meta.Uuid, version.Id)
		return txn.Set(ctx, dbKey, b)
	})
}

func (e *ObjectEntity) MergeDelta(ctx context.Context, delta *models.ObjectMeta) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading object details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Key)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Key)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.ObjectMeta{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging object delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Key)
	}

	return b, nil
}

func convertToErasureShards(dbShards []*models.Shard) []erasure.Shard {
	shards := make([]erasure.Shard, len(dbShards))
	for i, s := range dbShards {
		shards[i] = erasure.Shard{
			Type:     s.Type,
			Size:     s.Size,
			Checksum: s.Checksum,
			Path:     s.Path,
		}
	}
	return shards
}

func convertToDbShards(shards []erasure.Shard) []*models.Shard {
	dbShards := make([]*models.Shard, len(shards))
	for i, s := range shards {
		dbShards[i] = &models.Shard{
			Type:     s.Type,
			Size:     s.Size,
			Checksum: s.Checksum,
			Path:     s.Path,
		}
	}
	return dbShards
}
