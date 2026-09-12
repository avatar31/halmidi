package s3

import (
	"context"
	"os"
	"sort"

	"github.com/avatar31/omashu"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	objectio "github.com/avatar31/halmidi/internal/persistent/object_io"
	"github.com/avatar31/halmidi/utils"
)

type ObjectVersionService struct {
	log    logger.Logger
	bdb    *omashu.DistributedBadger
	bucket *models.BucketMeta
	object *models.ObjectMeta
}

func NewObjectVersionService(ctx context.Context, bucket *models.BucketMeta,
	object *models.ObjectMeta) *ObjectVersionService {
	return &ObjectVersionService{
		log: logger.GetLogger(ctx).WithFields(map[string]any{
			"bucket": bucket.Name,
			"object": object.Key,
		}),
		bdb:    dbstore.GetDBStore(ctx),
		bucket: bucket,
		object: object,
	}
}

func (s *ObjectVersionService) ListVersions(ctx context.Context) ([]*models.ObjectVersion, error) {
	dbPrefix := dbstore.GetObjectVersioningDBKeyWithNS(s.object.Uuid, "")
	versionsMap, err := s.bdb.GetByPrefix(ctx, dbPrefix)
	if err != nil {
		s.log.WithError(err).Error("Error while reading object versions list from db")
		return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	versions := []*models.ObjectVersion{}
	for k, v := range versionsMap {
		meta, err := models.DecodeObjectVersion(v)
		if err != nil {
			s.log.WithError(err).Errorf("Error while unmarshalling object versioning metadata with key %s", k)
			continue
		}

		versions = append(versions, meta)
	}

	if len(versions) > 0 {
		sort.Slice(versions, func(i, j int) bool {
			aTime, err := utils.ConvertStringToTime(versions[i].LastModified)
			if err != nil {
				return false
			}
			bTime, err := utils.ConvertStringToTime(versions[j].LastModified)
			if err != nil {
				return false
			}
			return aTime.Before(*bTime)
		})

		versions[len(versions)-1].IsLatest = true
	}
	return versions, nil
}

func (s *ObjectVersionService) GetObjectVersionEntity(ctx context.Context,
	versionId string) (*ObjectVersionEntity, error) {
	entity := &ObjectVersionEntity{
		bdb:    s.bdb,
		bucket: s.bucket,
		object: s.object,
	}

	err := entity.fetchMetadataFromDB(ctx, versionId)
	if err != nil {
		return nil, err
	}

	entity.log = s.log.WithField("version", versionId)
	return entity, nil
}

func (s *ObjectVersionService) GetObjectVersionEntityWithMetadata(ctx context.Context,
	meta *models.ObjectVersion) *ObjectVersionEntity {
	return &ObjectVersionEntity{
		log:    s.log.WithField("version", meta.Id),
		bdb:    s.bdb,
		bucket: s.bucket,
		object: s.object,
		meta:   meta,
	}
}

func (s *ObjectVersionService) CreateNewObjectVersion(ctx context.Context) (*models.ObjectVersion, string, string) {
	if !s.bucket.IsVersioningInitialized() {
		return nil, "", ""
	}

	var version *models.ObjectVersion
	if s.bucket.IsVersioningSuspended() {
		versionEntity, err := s.GetObjectVersionEntity(ctx, s3common.NO_VERSION)
		if err != nil {
			// There is no object uploaded with NO_VERSION after suspending versioning
			s.log.Info("Creating object with NO_VERSION")
			version = s.object.ApplyVersioning(s.bucket, false)
		} else {
			version = versionEntity.GetMetadata(ctx)
		}
	} else {
		version = s.object.ApplyVersioning(s.bucket, false)
	}

	var name, id string
	if version.Id != s3common.NO_VERSION {
		// TODO: Check is it right behavior to return version id as empty string for NO_VERSION
		name = version.Name
		id = version.Id
	}

	return version, name, id
}

type ObjectVersionEntity struct {
	log    logger.Logger
	bdb    *omashu.DistributedBadger
	bucket *models.BucketMeta
	object *models.ObjectMeta
	dbKey  string
	meta   *models.ObjectVersion
}

func (e *ObjectVersionEntity) GetMetadata(ctx context.Context) *models.ObjectVersion {
	return e.meta
}

func (e *ObjectVersionEntity) fetchMetadataFromDB(ctx context.Context, versionId string) error {
	e.dbKey = dbstore.GetObjectVersioningDBKeyWithNS(e.object.Uuid, versionId)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading object versioning metadata from db")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}
	if !ok {
		return s3common.GetNoSuchVersionS3Error(e.bucket.Name)
	}

	data, err := models.DecodeObjectVersion(b)
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling object versioning metadata from db")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	e.meta = data
	return nil
}

func (e *ObjectVersionEntity) UpdateObjectVersionLegaHold(ctx context.Context, status s3common.LegalHoldStatus) error {
	delta := &models.ObjectVersion{}
	delta.UpdateLegalHoldStatus(status)

	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating object version legal hold status")
		return s3common.GetInternalErrorS3Error(e.bucket.Name)
	}

	return nil
}

func (e *ObjectVersionEntity) GetObjectVersionLegaHold(ctx context.Context) (s3common.LegalHoldStatus, error) {
	legalHold := s3common.LegalHoldStatusOff
	if e.meta.ObjectLock != nil && e.meta.ObjectLock.LegalHoldStatus != nil {
		legalHold = s3common.LegalHoldStatus(*e.meta.ObjectLock.LegalHoldStatus)
	}

	return legalHold, nil
}

func (e *ObjectVersionEntity) GetObjectVersion(ctx context.Context, opts GetObjectOptions) (*models.ObjectMeta, *os.File, error) {
	if e.meta.IsDeleteMarker != nil && *e.meta.IsDeleteMarker {
		return nil, nil, s3common.GetNoSuchKeyS3Error(e.bucket.Name)
	}

	input := objectio.ObjectInput{
		Bucket:  e.bucket.Name,
		Id:      e.object.Uuid,
		Key:     e.object.Key,
		Version: e.meta.Name,
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

	return e.object, file, nil
}

func (e *ObjectVersionEntity) MergeDelta(ctx context.Context, delta *models.ObjectVersion) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading object version details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Name)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.ObjectVersion{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging policy delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return b, nil
}
