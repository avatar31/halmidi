package iam

import (
	"context"
	"time"

	"github.com/avatar31/omashu"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

type UserAccessKeyService struct {
	log  logger.Logger
	bdb  *omashu.DistributedBadger
	user *models.User
}

type AccessKeyService struct {
	log logger.Logger
	bdb *omashu.DistributedBadger
}

func NewUserAccessKeyService(ctx context.Context, username string) (*UserAccessKeyService, error) {
	userEntity, err := NewUserService(ctx).GetUserEntity(ctx, username)
	if err != nil {
		return nil, err
	}

	return &UserAccessKeyService{
		log:  logger.GetLogger(ctx).WithField("username", username),
		bdb:  dbstore.GetDBStore(ctx),
		user: userEntity.GetMetadata(ctx),
	}, nil
}

func (s *UserAccessKeyService) GetUser() *models.User {
	return s.user
}

func (s *UserAccessKeyService) ListUsersAccessKeys(ctx context.Context) ([]*models.UserAccessKey, error) {
	refMap, err := s.bdb.GetByPrefix(ctx, dbstore.GetUserAccessKeyRefDBKeyWithNS(s.user.Name, ""))
	if err != nil {
		s.log.WithError(err).Error("Error while reading user accessKey references from db")
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	if len(refMap) == 0 {
		return []*models.UserAccessKey{}, nil
	}

	dbKeys := []string{}
	for _, v := range refMap {
		ref, err := models.DecodeResourceRef(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling user accesskey reference from db")
			return nil, s3common.GetInternalErrorS3Error(s.user.Name)
		}

		dbKeys = append(dbKeys, dbstore.GetAccessKeyDBKeyWithNS(ref.Reference))
	}

	result := []*models.UserAccessKey{}
	accessKeyMap, err := s.bdb.BulkGet(ctx, dbKeys)
	if err != nil {
		s.log.WithError(err).Error("Error while reading user accessKey details from db")
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	for _, b := range accessKeyMap {
		accessKey, err := models.DecodeUserAccessKey(b)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling user accesskey metadata from db")
			continue
		}
		result = append(result, accessKey)
	}

	return result, nil
}

func (s *UserAccessKeyService) CreateAccessKey(ctx context.Context) (*models.UserAccessKey, error) {
	meta, err := models.NewUserAccessKey(s.user.Name)
	if err != nil {
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	metaBytes, err := meta.Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while marshalling access key metadata for user %s", s.user.Name)
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	refBytes, err := models.NewResourceRef(meta.AccessKeyId).Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while marshalling access key ref for user %s", s.user.Name)
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	accessKeyDBKey := dbstore.GetAccessKeyDBKeyWithNS(meta.AccessKeyId)
	userAccessKeyRefDBKey := dbstore.GetUserAccessKeyRefDBKeyWithNS(s.user.Name, meta.AccessKeyId)
	err = s.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		result, err := s.bdb.GetByPrefixWithTxn(ctx, txn, dbstore.GetUserAccessKeyRefDBKeyWithNS(s.user.Name, ""))
		if err != nil {
			return err
		}

		// TODO: Access key can be inactive, need to count only active keys
		if len(result) >= s3common.MAX_ALLOWED_USER_ACCESS_KEYS {
			s.log.Errorf("User %s has reached max access keys limit", s.user.Name)
			return s3common.GetLimitExceededS3Error(s.user.Name)
		}

		err = txn.Set(ctx, accessKeyDBKey, metaBytes)
		if err != nil {
			return err
		}

		return txn.Set(ctx, userAccessKeyRefDBKey, refBytes)
	})
	if err != nil {
		if s3common.IsS3Error(err) {
			return nil, err
		}
		s.log.WithError(err).Errorf("Error while creating access key for user %s", s.user.Name)
		return nil, s3common.GetInternalErrorS3Error(s.user.Name)
	}

	s.log.Infof("Created new access key for user %s", s.user.Name)
	return meta, nil
}

func (s *UserAccessKeyService) GetUserAccessKeyEntity(ctx context.Context,
	accessKey string) (*UserAccessKeyEntity, error) {
	entity := &UserAccessKeyEntity{
		log:  s.log,
		bdb:  s.bdb,
		user: s.user,
	}

	err := entity.fetchMetadataFromDB(ctx, accessKey)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func NewAccessKeyService(ctx context.Context) *AccessKeyService {
	return &AccessKeyService{
		log: logger.GetLogger(ctx),
		bdb: dbstore.GetDBStore(ctx),
	}
}

func (s *AccessKeyService) GetUserAccessKeyEntity(ctx context.Context,
	accessKey string) (*UserAccessKeyEntity, error) {
	entity := &UserAccessKeyEntity{
		log: s.log,
		bdb: s.bdb,
	}

	err := entity.fetchMetadataFromDB(ctx, accessKey)
	if err != nil {
		return nil, err
	}

	userEntity, err := NewUserService(ctx).GetUserEntity(ctx, entity.meta.UserName)
	if err != nil {
		return nil, err
	}
	entity.user = userEntity.GetMetadata(ctx)
	return entity, nil
}

type UserAccessKeyEntity struct {
	log   logger.Logger
	bdb   *omashu.DistributedBadger
	user  *models.User
	dbKey string
	meta  *models.UserAccessKey
}

func (e *UserAccessKeyEntity) fetchMetadataFromDB(ctx context.Context, accessKey string) error {
	e.dbKey = dbstore.GetAccessKeyDBKeyWithNS(accessKey)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Errorf("Error while reading access key %s from db", accessKey)
		return s3common.GetInternalErrorS3Error(accessKey)
	}
	if !ok {
		return s3common.GetNoSuchEntityS3Error(accessKey)
	}

	data, err := models.DecodeUserAccessKey(b)
	if err != nil {
		e.log.WithError(err).Errorf("Error while unmarshalling access key %s metadata from db", accessKey)
		return s3common.GetInternalErrorS3Error(accessKey)
	}

	e.meta = data
	return nil
}

func (e *UserAccessKeyEntity) GetMetadata(ctx context.Context) *models.UserAccessKey {
	return e.meta
}

func (e *UserAccessKeyEntity) DeleteAccessKey(ctx context.Context) error {
	userAccessKeyRefDbKey := dbstore.GetUserAccessKeyRefDBKeyWithNS(e.meta.UserName, e.meta.AccessKeyId)

	err := e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, userAccessKeyRefDbKey)
		if err != nil {
			return err
		}
		return txn.Delete(ctx, e.dbKey)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while deleting access key %s", e.meta.AccessKeyId)
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}

	e.log.Infof("Deleted access key %s", e.meta.AccessKeyId)
	return nil
}

func (e *UserAccessKeyEntity) UpdateUsersAccessKey(ctx context.Context, status s3common.AccessKeyStatus) error {
	if e.meta.Status == status.String() {
		return nil
	}

	delta := &models.UserAccessKey{Status: status.String()}
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user accessKey details from db")
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}
	if !ok {
		e.log.Errorf("Access key %s not found in db while updating status", e.meta.AccessKeyId)
		return s3common.GetNoSuchEntityS3Error(e.meta.AccessKeyId)
	}

	updated, err := models.MergeProtoMessages(b, &models.UserAccessKey{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging user accessKey details")
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}
	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating user accessKey details from db")
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}

	e.log.Infof("Updated status of access key %s to %s", e.meta.AccessKeyId, status)
	return nil
}

func (e *UserAccessKeyEntity) GetAccessKeyLastUsed(ctx context.Context) (string, error) {
	dbKey := dbstore.GetUserAccessKeyLastUsedDBKeyWithNS(e.meta.AccessKeyId)
	b, ok, err := e.bdb.Get(ctx, dbKey)
	if err != nil || !ok {
		e.log.WithError(err).Errorf("Error while reading last used timestamp for access key %s", e.meta.AccessKeyId)
		return "", s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}

	msg, err := models.DecodeAccessKeyLastUsed(b)
	if err != nil {
		e.log.WithError(err).Errorf("Error while unmarshalling last used timestamp for access key %s", e.meta.AccessKeyId)
		return "", s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}

	return msg.LastUsed, nil
}

func (e *UserAccessKeyEntity) UpdateAccessKeyLastUsed(ctx context.Context) error {
	now := utils.ConvertTimeToString(time.Now())
	b, err := models.NewAccessKeyLastUsed(now).Encode()
	if err != nil {
		e.log.WithError(err).Errorf("Error while marshalling last used timestamp for access key %s", e.meta.AccessKeyId)
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}

	dbKey := dbstore.GetUserAccessKeyLastUsedDBKeyWithNS(e.meta.AccessKeyId)
	err = e.bdb.Set(ctx, dbKey, b)
	if err != nil {
		e.log.WithError(err).Errorf("Error while updating last used timestamp for access key %s", e.meta.AccessKeyId)
		return s3common.GetInternalErrorS3Error(e.meta.AccessKeyId)
	}
	return nil
}
