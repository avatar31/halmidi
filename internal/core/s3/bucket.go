package s3

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/avatar31/omashu"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/storage"
	"github.com/avatar31/halmidi/utils"
)

var (
	bucketNameRegex = regexp.MustCompile(s3common.BucketNameRegex)
)

type ListBucketOptions struct {
	Prefix            string
	ContinuationToken string
	MaxItems          int
}

type CreateBucketRequest struct {
	Name          string
	Versioning    *s3common.VersioningStatus
	ObjectLocking *models.BucketObjectLocking
	Tags          *models.TagMap
}

func (r *CreateBucketRequest) Validate(ctx context.Context) error {
	if len(r.Name) < s3common.MIN_ALLOWED_BUCKET_NAME_LENGTH ||
		len(r.Name) > s3common.MAX_ALLOWED_BUCKET_NAME_LENGTH ||
		!bucketNameRegex.MatchString(r.Name) {
		return s3common.GetInvalidArgumentS3Error(r.Name, "Invalid Bucket name")
	}

	if r.Versioning != nil {
		if err := r.Versioning.Validate(r.Name); err != nil {
			return err
		}
	}

	if r.ObjectLocking != nil {
		if err := r.ObjectLocking.Validate(r.Name); err != nil {
			return err
		}
	}

	if err := r.Tags.Validate(r.Name); err != nil {
		return err
	}

	return nil
}

type BucketService struct {
	log logger.Logger
	bdb *omashu.DistributedBadger
}

func NewBucketServiceWithDB(ctx context.Context, db *omashu.DistributedBadger) *BucketService {
	service := &BucketService{
		log: logger.GetLogger(ctx),
		bdb: db,
	}
	return service
}

func NewBucketService(ctx context.Context) *BucketService {
	return NewBucketServiceWithDB(ctx, dbstore.GetDBStore(ctx))
}

func (s *BucketService) IsExist(ctx context.Context, bucket string) bool {
	return s.bdb.Exists(ctx, dbstore.GetBucketDBKeyWithNS(bucket))
}

func (s *BucketService) ListBuckets(ctx context.Context, opts ListBucketOptions) ([]*models.BucketMeta, string, error) {
	startAfter := ""
	if opts.ContinuationToken != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.ContinuationToken)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid continuation-token")
		}
		startAfter = dbstore.GetBucketDBKeyWithNS(decodedMarker)
	}

	result := []*models.BucketMeta{}
	dbPrefix := dbstore.GetBucketDBKeyWithNS(opts.Prefix)
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxItems, func(k, v []byte) bool {
		meta, err := models.DecodeBucketMeta(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from bucket list")
			return false
		}

		result = append(result, meta)
		return true
	})
	if err != nil {
		s.log.WithError(err).Error("Error while reading bucket list from db")
		return nil, "", s3common.GetInternalErrorS3Error("")
	}

	if nextCursor != "" {
		// Trim the object namespace from the cursor before returning
		nextCursor = strings.TrimPrefix(nextCursor, dbstore.GetBucketDBKeyWithNS(""))
	}

	return result, secure.EncodeSignedMarker(nextCursor), nil
}

func (s *BucketService) GetBucketEntity(ctx context.Context, bucket string) (*BucketEntity, error) {
	entity := &BucketEntity{
		log: s.log.WithField("bucket", bucket),
		bdb: s.bdb,
	}

	err := entity.fetchMetadataFromDB(ctx, bucket)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *BucketService) CreateBucket(ctx context.Context, req CreateBucketRequest) (*models.BucketMeta, error) {
	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	if s.IsExist(ctx, req.Name) {
		// TODO: or BucketAlreadyOwnedByYou
		return nil, s3common.GetBucketAlreadyExistsS3Error(req.Name)
	}

	log := s.log.WithField("bucket", req.Name)
	meta := models.NewBucketMeta(req.Name, req.Tags, req.ObjectLocking)
	meta.ObjectLocking = req.ObjectLocking

	if req.Versioning != nil {
		meta.Versioning = models.NewBucketVersioning(*req.Versioning)
	}

	key := dbstore.GetBucketDBKeyWithNS(req.Name)
	metaBytes, err := meta.Encode()
	if err != nil {
		log.WithError(err).Error("Error while encoding bucket metadata")
		return nil, s3common.GetInternalErrorS3Error(req.Name)
	}
	err = s.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, key, metaBytes)
		if err != nil {
			return err
		}

		err = txn.Set(ctx, dbstore.GetBucketSpaceUsageDBKeyWithNS(meta.Name), utils.Uint64ToBytes(0))
		if err != nil {
			return err
		}

		return txn.Set(ctx, dbstore.GetBucketObjectCountDBKeyWithNS(meta.Name), utils.Uint64ToBytes(0))
	})

	if err != nil {
		log.WithError(err).Error("Error while updating bucket metadata")
		return nil, s3common.GetInternalErrorS3Error(req.Name)
	}

	return meta, nil
}

func (s *BucketService) GetObjectService(ctx context.Context, bucket *BucketEntity) *ObjectService {
	return &ObjectService{
		log:    s.log,
		bdb:    s.bdb,
		bucket: bucket.GetMetadata(ctx),
	}
}

type BucketEntity struct {
	log   logger.Logger
	bdb   *omashu.DistributedBadger
	dbKey string
	meta  *models.BucketMeta
}

func (e *BucketEntity) GetMetadata(ctx context.Context) *models.BucketMeta {
	return e.meta
}

func (e *BucketEntity) fetchMetadataFromDB(ctx context.Context, bucket string) error {
	e.dbKey = dbstore.GetBucketDBKeyWithNS(bucket)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while fetching bucket metadata from db")
		return s3common.GetInternalErrorS3Error(bucket)
	}
	if !ok {
		return s3common.GetNoSuchBucketS3Error(bucket)
	}

	data, err := models.DecodeBucketMeta(b)
	if err != nil {
		e.log.WithError(err).Error("Error while decoding bucket metadata from db")
		return s3common.GetInternalErrorS3Error(bucket)
	}

	e.meta = data
	return nil
}

func (e *BucketEntity) GetObjectLocking(ctx context.Context) *models.BucketObjectLocking {
	return e.meta.ObjectLocking
}

func (e *BucketEntity) GetVersioningConfig(ctx context.Context) *models.BucketVersioning {
	return e.meta.Versioning
}

func (e *BucketEntity) UpdateVersioning(ctx context.Context, status s3common.VersioningStatus) error {
	if err := status.Validate(e.meta.Name); err != nil {
		return err
	}

	delta := &models.BucketMeta{Versioning: models.NewBucketVersioning(status)}
	return e.updateBucket(ctx, delta)
}

func (e *BucketEntity) updateBucket(ctx context.Context, delta *models.BucketMeta) error {
	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating bucket")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return nil
}

func (e *BucketEntity) ListObjectVersions(ctx context.Context,
	prefix string) ([]*models.ObjectVersion, []*models.ObjectVersion, error) {
	versions := []*models.ObjectVersion{}
	deleteMarkers := []*models.ObjectVersion{}
	if !e.meta.IsVersioningEnabled() {
		return versions, deleteMarkers, nil
	}

	objService := NewBucketService(ctx).GetObjectService(ctx, e)
	objects, err := objService.ListObjectsWithPrefix(ctx, prefix)
	if err != nil {
		return nil, nil, err
	}
	if len(objects) == 0 {
		return versions, deleteMarkers, nil
	}

	for _, objectMeta := range objects {
		versionService := NewObjectVersionService(ctx, e.meta, objectMeta)
		mixedVersions, err := versionService.ListVersions(ctx)
		if err != nil {
			return nil, nil, err
		}

		for _, meta := range mixedVersions {
			if meta.IsDeleteMarker != nil {
				deleteMarkers = append(deleteMarkers, meta)
			} else {
				versions = append(versions, meta)
			}
		}
	}

	return versions, deleteMarkers, nil
}

func (e *BucketEntity) UpdateObjectLocking(ctx context.Context, req *models.BucketObjectLocking) error {
	if e.meta.ObjectLocking == nil {
		return s3common.GetInvalidRequestS3Error(e.meta.Name, "Object Locking cannot enabled after creating bucket.")
	}

	if err := req.Validate(e.meta.Name); err != nil {
		return err
	}

	delta := &models.BucketMeta{ObjectLocking: req}
	return e.updateBucket(ctx, delta)
}

func (e *BucketEntity) GetTags(ctx context.Context) (*models.TagMap, error) {
	if len(e.meta.Tags.Items) <= 0 {
		return nil, s3common.GetNoSuchTagSetS3Error(e.meta.Name)
	}
	return e.meta.Tags, nil
}

func (e *BucketEntity) UpdateTags(ctx context.Context, tags *models.TagMap) error {
	delta := &models.BucketMeta{}
	if err := delta.UpdateTags(e.meta.Name, tags); err != nil {
		return err
	}

	return e.updateBucket(ctx, delta)
}

func (e *BucketEntity) GetLifeCycleConfig(ctx context.Context) (*models.LifecycleConfig, error) {
	if e.meta.LifecycleConfig == nil {
		return nil, s3common.GetNoSuchLifecycleConfigurationS3Error(e.meta.Name)
	}

	return e.meta.LifecycleConfig, nil
}

func (e *BucketEntity) updateLifeCycleConfig(ctx context.Context, config *models.LifecycleConfig) error {
	delta := &models.BucketMeta{LifecycleConfig: config}
	return e.updateBucket(ctx, delta)
}

func (e *BucketEntity) UpdateLifeCycleConfig(ctx context.Context, config *models.LifecycleConfig) error {
	if err := config.Validate(e.meta.Name); err != nil {
		return err
	}

	return e.updateLifeCycleConfig(ctx, config)
}

func (e *BucketEntity) DeleteLifeCycleConfig(ctx context.Context) error {
	return e.updateLifeCycleConfig(ctx, nil)
}

func (e *BucketEntity) DeleteBucket(ctx context.Context) error {
	prefix := dbstore.GetObjectDBKeyWithNS(e.meta.Name, "")
	if e.bdb.Count(ctx, prefix) > 0 {
		return s3common.GetBucketNotEmptyS3Error(e.meta.Name)
	}

	err := e.bdb.Delete(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while deleting bucket metadata")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	go func(bucket string) {
		disks := storage.GetDisks()
		for _, disk := range disks {
			bucketDiskPath := filepath.Join(disk.Path, bucket)
			if err := os.RemoveAll(bucketDiskPath); err != nil {
				e.log.WithError(err).Errorf("Error while deleting bucket data on disk %s", disk.Path)
				// TODO: Handle error
			}
		}
	}(e.meta.Name)

	return nil
}

func (e *BucketEntity) MergeDelta(ctx context.Context, delta *models.BucketMeta) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading bucket details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Name)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.BucketMeta{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging policy delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return b, nil
}
