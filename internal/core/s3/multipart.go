package s3

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/avatar31/omashu"
	"google.golang.org/protobuf/proto"

	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	objectio "github.com/avatar31/halmidi/internal/persistent/object_io"
	"github.com/avatar31/halmidi/internal/persistent/storage/erasure"
)

type UploadPartOptions struct {
	UploadID   string
	PartNumber int32

	// For Copy Part
	CopySourceBucket     string
	CopySourceKey        string
	CopySourceRangeStart int64
	CopySourceRangeEnd   int64
}

type InitMultipartUploadOptions struct {
	LockMode    string
	RetainUntil string
	LegalHold   string
}

type MultiPartUploadService struct {
	log    logger.Logger
	bdb    *omashu.DistributedBadger
	bucket *models.BucketMeta
}

func NewMultiPartUploadService(ctx context.Context, bucketName string) (*MultiPartUploadService, error) {
	db := dbstore.GetDBStore(ctx)
	bucket, err := NewBucketServiceWithDB(ctx, db).GetBucketEntity(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	service := &MultiPartUploadService{
		log:    logger.GetLogger(ctx).WithField("bucket", bucket.meta.Name),
		bdb:    db,
		bucket: bucket.GetMetadata(ctx),
	}

	return service, nil
}

func (s *MultiPartUploadService) InitMultipartUpload(ctx context.Context, objectKey string,
	opts InitMultipartUploadOptions) (string, error) {
	if !isValidObjectKey(objectKey) {
		return "", s3common.GetInvalidKeyS3Error(s.bucket.Name)
	}

	log := s.log.WithField("objectKey", objectKey)

	if !s.bucket.IsObjectLockingEnabled() &&
		((opts.LockMode != "" && opts.RetainUntil != "") || (opts.LegalHold != "")) {
		return "", s3common.GetInvalidRequestS3Error(s.bucket.Name, "Bucket is not enabled for Object Lock.")
	}

	if opts.LegalHold != "" {
		if err := s3common.LegalHoldStatus(opts.LegalHold).Validate(s.bucket.Name); err != nil {
			return "", err
		}
	}

	if opts.LockMode != "" {
		if err := s3common.ObjectLockingMode(opts.LockMode).Validate(s.bucket.Name); err != nil {
			return "", err
		}
	}

	mpu := models.NewMultipartUpload(s.bucket, objectKey, opts.LockMode, opts.RetainUntil, opts.LegalHold)
	metaBytes, err := mpu.Encode()
	if err != nil {
		log.WithError(err).Errorf("Error while encoding metadata for multipart upload")
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	dbKey := dbstore.GetMultipartUploadDBKeyWithNS(mpu.Uuid)
	if err := s.bdb.Set(ctx, dbKey, metaBytes); err != nil {
		log.WithError(err).Errorf("Error while updating metadata for multipart upload")
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	return mpu.Uuid, nil
}

func (s *MultiPartUploadService) ListMultipartUploads(ctx context.Context) ([]*models.MultipartUpload, error) {
	log := s.log.WithField("operation", "ListMultipartUploads")
	prefix := dbstore.GetMultipartUploadDBKeyWithNS("")

	result := []*models.MultipartUpload{}
	_, err := s.bdb.IterateByPrefix(ctx, prefix, "", nil, func(k, v []byte) bool {
		var meta models.MultipartUpload
		err := proto.Unmarshal(v, &meta)
		if err != nil {
			log.WithError(err).Errorf("Error while unmarshalling multipart upload metadata with key %s", string(k))
			return false
		}

		if meta.Bucket != s.bucket.Name {
			return false
		}

		result = append(result, &meta)
		return true
	})

	if err != nil {
		log.WithError(err).Error("Error while reading multipart uploads metadata from db")
		return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	return result, nil
}

func (s *MultiPartUploadService) UploadPart(ctx context.Context, body io.Reader,
	opts UploadPartOptions) (string, error) {
	if opts.CopySourceBucket != "" && opts.CopySourceKey != "" {
		fileName, err := s.getCopySourceReader(ctx, opts)
		if err != nil {
			return "", err
		}

		file, err := os.Open(fileName)
		if err != nil {
			s.log.WithError(err).Errorf("Error while opening copy source file %s", fileName)
			return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
		}

		defer func() {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}()

		body = file
	}

	mtuKey := dbstore.GetMultipartUploadDBKeyWithNS(opts.UploadID)
	msgBytes, ok, err := s.bdb.Get(ctx, mtuKey)
	if err != nil {
		s.log.WithError(err).Errorf("Error while reading multipart upload metadata for uploadID %s", opts.UploadID)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}
	if !ok {
		return "", s3common.GetNoSuchUploadS3Error(s.bucket.Name)
	}

	mpu, err := models.DecodeMultipartUpload(msgBytes)
	if err != nil {
		s.log.WithError(err).Errorf("Error while decoding multipart upload metadata for uploadID %s", opts.UploadID)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	log := s.log.WithFields(map[string]any{
		"objectKey": mpu.Key,
		"uploadID":  opts.UploadID,
	})

	tmpDir := filepath.Join(config.GetTmpDir(), "multipart", opts.UploadID)
	_ = os.MkdirAll(tmpDir, 0755)
	partPath := filepath.Join(tmpDir, fmt.Sprintf("part%d", opts.PartNumber))

	file, err := os.Create(partPath)
	if err != nil {
		log.WithError(err).Errorf("Error while creating part file %s", partPath)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := md5.New()
	size, err := io.Copy(io.MultiWriter(file, hash), body)
	if err != nil {
		log.WithError(err).Errorf("Error while copying data to part file %s", partPath)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	etag := hex.EncodeToString(hash.Sum(nil))
	partBytes, err := models.NewMultipartUploadPart(opts.UploadID, partPath, etag, opts.PartNumber, size).Encode()
	if err != nil {
		log.WithError(err).Errorf("Error while encoding metadata for part %s", partPath)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	partDBKey := dbstore.GetMultipartUploadPartDBKeyWithNS(opts.UploadID, fmt.Sprint(opts.PartNumber))
	err = s.bdb.Set(ctx, partDBKey, partBytes)
	if err != nil {
		log.WithError(err).Errorf("Error while updating metadata for part %s", partPath)
		return "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	return etag, nil
}

func (s *MultiPartUploadService) ListParts(ctx context.Context, uploadId string) ([]*models.MultipartUploadPart, error) {
	log := logger.GetLogger(ctx)
	prefix := dbstore.GetMultipartUploadPartDBKeyWithNS(uploadId, "")
	partsMap, err := s.bdb.GetByPrefix(ctx, prefix)
	if err != nil {
		log.WithError(err).Error("Error while reading multipart upload  parts details from db")
		return nil, s3common.GetInternalErrorS3Error("")
	}

	result := []*models.MultipartUploadPart{}
	for k, v := range partsMap {
		var meta models.MultipartUploadPart
		err = proto.Unmarshal(v, &meta)
		if err != nil {
			log.WithError(err).Errorf("Error while unmarshalling multipart upload part metadata with key %s", k)
			continue
		}

		result = append(result, &meta)
	}

	return result, nil
}

func (s *MultiPartUploadService) getCopySourceReader(ctx context.Context, opts UploadPartOptions) (string, error) {
	service := NewBucketService(ctx)
	bucketEntity, err := service.GetBucketEntity(ctx, opts.CopySourceBucket)
	if err != nil {
		return "", err
	}

	objectEntity, err := service.GetObjectService(ctx, bucketEntity).GetObjectEntity(ctx, opts.CopySourceKey)
	if err != nil {
		return "", err
	}

	getObjectOpts := GetObjectOptions{
		RangeStart: &opts.CopySourceRangeStart,
		RangeEnd:   &opts.CopySourceRangeEnd,
	}
	_, file, err := objectEntity.GetObject(ctx, getObjectOpts)
	if err != nil {
		return "", err
	}
	fileName := file.Name()
	_ = file.Close()

	return fileName, nil
}

func (s *MultiPartUploadService) ValidateParts(ctx context.Context, uploadId string,
	parts []int32) (map[int32]*models.MultipartUploadPart, error) {
	mpuParts, err := s.ListParts(ctx, uploadId)
	if err != nil {
		return nil, err
	}

	if len(parts) != len(mpuParts) {
		s.log.Errorf("Parts in request are not matching with parts available in DB. Available parts in DB: %+v",
			mpuParts)
		return nil, s3common.GetInvalidPartS3Error(s.bucket.Name)
	}

	partMap := make(map[int32]*models.MultipartUploadPart)
	for _, storedPart := range mpuParts {
		partMap[storedPart.PartNumber] = storedPart
	}

	for _, partNumber := range parts {
		_, ok := partMap[partNumber]
		if !ok {
			s.log.Errorf("Details for part number %d not available in DB.", partNumber)
			return nil, s3common.GetInvalidPartS3Error(s.bucket.Name)
		}
	}

	return partMap, nil
}

func (s *MultiPartUploadService) CompleteMultipartUpload(ctx context.Context, uploadId string,
	parts []int32) (string, string, error) {
	mpuKey := dbstore.GetMultipartUploadDBKeyWithNS(uploadId)
	mpuBytes, ok, err := s.bdb.Get(ctx, mpuKey)
	if err != nil {
		s.log.WithError(err).Errorf("Error while reading multipart upload metadata for uploadID %s", uploadId)
		return "", "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}
	if !ok {
		return "", "", s3common.GetNoSuchUploadS3Error(s.bucket.Name)
	}

	mpu, err := models.DecodeMultipartUpload(mpuBytes)
	if err != nil {
		s.log.WithError(err).Errorf("Error while decoding multipart upload metadata for uploadID %s", uploadId)
		return "", "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	s.log = s.log.WithFields(map[string]any{
		"objectKey": mpu.Key,
		"uploadID":  uploadId,
	})

	partMap, err := s.ValidateParts(ctx, uploadId, parts)
	if err != nil {
		return "", "", err
	}

	s.log.Info("Saving multi-part upload to storage.")
	objectEntity, err := newObjectServiceInt(s.log, s.bdb, s.bucket).GetObjectEntity(ctx, mpu.Key)
	if err != nil {
		s.log.Info("Object doesn't exist. Creating new object.")
		objectEntity = &ObjectEntity{
			log:    s.log,
			bdb:    s.bdb,
			bucket: s.bucket,
			meta:   models.NewObjectMetadata(s.bucket, mpu.Key),
		}
	}

	versionMeta, versionName, versionId := NewObjectVersionService(ctx, s.bucket, objectEntity.meta).
		CreateNewObjectVersion(ctx)
	if versionMeta == nil {
		err := objectEntity.meta.ApplyObjectLocking(ctx, objectEntity.bucket, string(mpu.LockMode), mpu.RetainUntil)
		if err != nil {
			return "", "", err
		}
	} else {
		objectEntity.log = objectEntity.log.WithField("versionId", versionId)
		err := versionMeta.ApplyObjectLocking(ctx, objectEntity.bucket, string(mpu.LockMode), mpu.RetainUntil)
		if err != nil {
			return "", "", err
		}
	}

	fileInfo, err := s.writeDataToStorage(ctx, partMap, parts, objectEntity, versionName)
	if err != nil {
		return "", "", err
	}

	err = objectEntity.updateObjectMeta(ctx, fileInfo, versionMeta)
	if err != nil {
		s.log.WithError(err).Error("Error while updating metadata after writing file data to disk")
		return "", "", s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	err = s.bdb.Delete(ctx, mpuKey)
	if err != nil {
		s.log.WithError(err).Errorf("Error while deleting multipart upload metadata for uploadID %s", uploadId)
	}

	err = s.bdb.DeleteByPrefix(ctx, dbstore.GetMultipartUploadPartDBKeyWithNS(uploadId, ""))
	if err != nil {
		s.log.WithError(err).Errorf("Error while deleting multipart upload parts metadata for uploadID %s", uploadId)
	}

	_ = os.RemoveAll(filepath.Join(config.GetTmpDir(), "multipart", uploadId))

	return fileInfo.Checksum, versionId, nil
}

func (s *MultiPartUploadService) writeDataToStorage(ctx context.Context, dbPartsMap map[int32]*models.MultipartUploadPart,
	parts []int32, objectEntity *ObjectEntity, version string) (*erasure.FileInfo, error) {
	input := objectio.ObjectInput{
		Bucket:  s.bucket.Name,
		Id:      objectEntity.meta.Uuid,
		Key:     objectEntity.meta.Key,
		Version: version,
	}

	success := false
	write, save, err := objectio.NewObjectIO().CreateStreamForWrite(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Error while writing data to disk")
		return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	defer func() {
		if success {
			s.log.Info("Successfully saved multipart upload as an object.")
		} else {
			_, _ = save(false)
		}
	}()

	for _, partNumber := range parts {
		part := dbPartsMap[partNumber]
		pf, err := os.Open(part.Path)
		if err != nil {
			s.log.WithError(err).Errorf("Error while reading multipart upload part: %+v", part)
			return nil, s3common.GetInvalidPartS3Error(s.bucket.Name)
		}

		err = write(pf)
		if err != nil {
			s.log.WithError(err).Error("Error while writing data to disk")
			return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
		}

		_ = pf.Close()
	}

	success = true
	stat, err := save(success)
	if err != nil {
		s.log.WithError(err).Error("Error while writing data to disk")
		return nil, s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	return stat, nil
}

func (s *MultiPartUploadService) AbortMultipartUpload(ctx context.Context, uploadId string) error {
	log := s.log.WithField("uploadID", uploadId)
	log.Info("Aborting multipart upload")

	mpuDbKey := dbstore.GetMultipartUploadDBKeyWithNS(uploadId)
	if !s.bdb.Exists(ctx, mpuDbKey) {
		return s3common.GetNoSuchUploadS3Error(s.bucket.Name)
	}

	err := s.bdb.Delete(ctx, mpuDbKey)
	if err != nil {
		log.WithError(err).Error("Error while deleting upload id from cache")
		return s3common.GetInternalErrorS3Error(s.bucket.Name)
	}

	go func() {
		err := s.bdb.DeleteByPrefix(ctx, dbstore.GetMultipartUploadPartDBKeyWithNS(uploadId, ""))
		if err != nil {
			log.WithError(err).Errorf("Error while deleting multipart upload parts metadata for uploadID %s", uploadId)
		}
		_ = os.RemoveAll(filepath.Join(config.GetTmpDir(), "multipart", uploadId))
	}()

	return nil
}

func isValidObjectKey(objectKey string) bool {
	return len(objectKey) > 0 && len(objectKey) <= s3common.MAX_ALLOWED_OBJECT_KEY_LENGTH
}
