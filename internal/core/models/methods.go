package models

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/avatar31/halmidi/internal/core/namespace"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

var (
	tagsKVRegex = regexp.MustCompile(s3common.TagKeyValueRegex)
)

func (meta *TagMap) Validate(bucket string) error {
	if len(meta.Items) > s3common.MAX_ALLOWED_TAGS {
		return s3common.GetTooManyTagsS3Error(bucket)
	}

	for k, v := range meta.Items {
		if err := ValidateTagKV(bucket, k, v); err != nil {
			return err
		}
	}

	return nil
}

// BucketMeta
func NewBucketMeta(name string, tags *TagMap, ol *BucketObjectLocking) *BucketMeta {
	return &BucketMeta{
		Uuid:          uuid.New().String(),
		Name:          name,
		CreationDate:  utils.ConvertTimeToString(time.Now()),
		Arn:           s3common.GenerateS3BucketARN(name),
		ObjectLocking: ol,
		Tags:          tags,
		Owner: &Owner{
			Id:          namespace.DefaultNameSpaceUUID,
			DisplayName: namespace.DEFAULT_NAMESPACE,
		},
	}
}

func DecodeBucketMeta(data []byte) (*BucketMeta, error) {
	var meta BucketMeta
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *BucketMeta) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

func (meta *BucketMeta) IsObjectLockingEnabled() bool {
	return meta.ObjectLocking != nil &&
		meta.ObjectLocking.ObjectLockEnabled != s3common.ObjectLockingStatusEnabled.String()
}

func (meta *BucketMeta) IsVersioningInitialized() bool {
	return meta.Versioning != nil
}

func (meta *BucketMeta) IsVersioningEnabled() bool {
	return meta.Versioning != nil && meta.Versioning.Status == s3common.VersioningStatusEnabled.String()
}

func (meta *BucketMeta) IsVersioningSuspended() bool {
	return meta.Versioning != nil && meta.Versioning.Status == s3common.VersioningStatusSuspended.String()
}

func (meta *BucketMeta) IsBucketLifecycleConfigured() bool {
	return meta.LifecycleConfig != nil && len(meta.LifecycleConfig.Rules) > 0
}

func (meta *BucketMeta) UpdateTags(bucket string, tags *TagMap) error {
	if err := tags.Validate(bucket); err != nil {
		return err
	}
	meta.Tags = tags
	return nil
}

// BucketVersioning

func NewBucketVersioning(status s3common.VersioningStatus) *BucketVersioning {
	return &BucketVersioning{Status: status.String()}
}

// BucketObjectLocking

func NewBucketObjectLocking(enabled s3common.ObjectLockingStatus) *BucketObjectLocking {
	return &BucketObjectLocking{ObjectLockEnabled: enabled.String()}
}

func (meta *BucketObjectLocking) UpdateModeAndRetention(mode s3common.ObjectLockingMode, days, years int32) {
	meta.Rule = &LockRule{
		DefaultRetention: &DefaultRetention{
			Mode:  mode.String(),
			Days:  days,
			Years: years,
		},
	}
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectLockConfiguration.html
func (meta *BucketObjectLocking) Validate(bucket string) error {
	if meta.ObjectLockEnabled != s3common.ObjectLockingStatusEnabled.String() {
		return s3common.GetInvalidArgumentS3Error(bucket, "Object Locking can not be disabled")
	}

	if err := s3common.ObjectLockingMode(meta.Rule.DefaultRetention.Mode).Validate(bucket); err != nil {
		return err
	}

	if meta.Rule.DefaultRetention.Days == 0 && meta.Rule.DefaultRetention.Years == 0 {
		return s3common.GetInvalidArgumentS3Error(bucket, "Please specify Retention Period Days or Years.")
	}

	if meta.Rule.DefaultRetention.Days != 0 && meta.Rule.DefaultRetention.Years != 0 {
		return s3common.GetInvalidArgumentS3Error(bucket, "Please specify either one of Days or Years.")
	}

	return nil
}

// ObjectMeta

func NewObjectMetadata(bucket *BucketMeta, objectKey string) *ObjectMeta {
	now := utils.ConvertTimeToString(time.Now())
	return &ObjectMeta{
		Uuid:              utils.GenerateDeterministicUUID(objectKey, s3common.ObjectNameSpaceUUID),
		Key:               objectKey,
		LastModified:      now,
		NextVersionNumber: 1,
		Arn:               s3common.GenerateS3ObjectARN(bucket.Name, objectKey),
		Owner: &Owner{
			Id:          namespace.DefaultNameSpaceUUID,
			DisplayName: namespace.DEFAULT_NAMESPACE,
		},
	}
}

func DecodeObjectMeta(data []byte) (*ObjectMeta, error) {
	var meta ObjectMeta
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *ObjectMeta) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

func (meta *ObjectMeta) CanBeDeleted(bucket string, bypassGovRetention bool) error {
	if meta.ObjectLock != nil {
		return meta.ObjectLock.CanBeDeleted(bucket, bypassGovRetention)
	} else {
		if bypassGovRetention {
			return s3common.GetInvalidRequestS3Error(bucket, "Object does not have Locking enabled to bypass.")
		}
	}

	return nil
}

func (meta *ObjectMeta) ApplyObjectLocking(ctx context.Context, bucketMeta *BucketMeta, newLockMode,
	newRetain string) error {
	objectLock, err := getObjectLocking(bucketMeta, newLockMode, newRetain)
	if err != nil {
		return err
	}

	if objectLock != nil {
		logger.GetLogger(ctx).WithFields(map[string]any{
			"bucket":    bucketMeta.Name,
			"objectKey": meta.Key,
		}).Info("Enabling object locking.")
		meta.ObjectLock = objectLock
	}

	return nil
}

func (meta *ObjectMeta) UpdateTags(bucket string, tags *TagMap) error {
	if err := tags.Validate(bucket); err != nil {
		return err
	}
	meta.Tags = tags
	return nil
}

func (meta *ObjectMeta) ApplyVersioning(bucket *BucketMeta, isDeleteMarker bool) *ObjectVersion {
	version := NewObjectVersion(bucket, meta, isDeleteMarker)
	meta.NextVersionNumber++

	return version
}

func (meta *ObjectMeta) UpdateLegalHoldStatus(status s3common.LegalHoldStatus) {
	if meta.ObjectLock == nil {
		meta.ObjectLock = NewObjectLock()
	}

	meta.ObjectLock.UpdateLegalHoldStatus(status)
}

// ObjectLock

func NewObjectLock() *ObjectLock {
	return &ObjectLock{}
}

func (meta *ObjectLock) UpdateModeAndRetainUntil(bucket, newLockMode, newRetain string) error {
	if err := s3common.ObjectLockingMode(newLockMode).Validate(bucket); err != nil {
		return err
	}

	retainUntil, err := utils.FormatTimeString(newRetain)
	if err != nil {
		return s3common.GetInvalidRequestS3Error(bucket, "Invalid object-lock retain until date format")
	}

	meta.Mode = &newLockMode
	meta.RetainUntil = &retainUntil
	return nil
}

func (meta *ObjectLock) RetainFromBucketConfig(bucket string, bucketOLConfig *BucketObjectLocking) {

	// This check is required because user may enabled object locking while creating a bucket.
	// But might not have updated mode and retention days by calling PutObjectLockConfiguration()
	err := s3common.ObjectLockingMode(bucketOLConfig.Rule.DefaultRetention.Mode).Validate(bucket)
	if err == nil {
		until := time.Now()
		if bucketOLConfig.Rule.DefaultRetention.Days != 0 {
			until = until.AddDate(0, 0, int(bucketOLConfig.Rule.DefaultRetention.Days))
		} else {
			until = until.AddDate(int(bucketOLConfig.Rule.DefaultRetention.Years), 0, 0)
		}

		retainUntilStr := utils.ConvertTimeToString(until)
		meta.Mode = &bucketOLConfig.Rule.DefaultRetention.Mode
		meta.RetainUntil = &retainUntilStr
	}
}

func (meta *ObjectLock) UpdateLegalHoldStatus(status s3common.LegalHoldStatus) {
	t := status.String()
	meta.LegalHoldStatus = &t
}

func (meta *ObjectLock) CanBeDeleted(bucket string, bypassGovRetention bool) error {
	now := time.Now().UTC()

	// Legal Hold Check - Priority 1
	if meta.LegalHoldStatus != nil && *meta.LegalHoldStatus == s3common.LegalHoldStatusOn.String() {
		return s3common.GetAccessDeniedS3Error(bucket, "Object is under legal hold.")
	}

	// Object Locking Check - Priority 2
	if meta.Mode != nil && meta.RetainUntil != nil {
		retailUntil, _ := utils.ConvertStringToTime(*meta.RetainUntil)
		retainPeroidExpired := retailUntil.After(now)

		if *meta.Mode == s3common.ObjectLockingModeCompliance.String() && retainPeroidExpired {
			return s3common.GetAccessDeniedS3Error(bucket, "Object is under Compliance retention.")
		}

		if *meta.Mode == s3common.ObjectLockingModeGovernance.String() &&
			retainPeroidExpired &&
			!bypassGovRetention {
			return s3common.GetAccessDeniedS3Error(bucket, "Object is under Governance retention.")
		}
	}

	return nil
}

// ObjectVersion

func NewObjectVersion(bucekt *BucketMeta, objectMeta *ObjectMeta, isDeleteMarker bool) *ObjectVersion {
	now := utils.ConvertTimeToString(time.Now())
	id := s3common.NO_VERSION
	if bucekt.IsVersioningEnabled() {
		id = uuid.New().String()
	}

	version := ObjectVersion{
		Key:          objectMeta.Key,
		Name:         fmt.Sprintf("v%d", objectMeta.NextVersionNumber),
		Id:           id,
		LastModified: now,
	}

	if isDeleteMarker {
		version.IsDeleteMarker = &isDeleteMarker
		version.Size = 0
	}

	return &version
}

func DecodeObjectVersion(data []byte) (*ObjectVersion, error) {
	var meta ObjectVersion
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *ObjectVersion) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

func (meta *ObjectVersion) ApplyObjectLocking(ctx context.Context, bucketMeta *BucketMeta, newLockMode,
	newRetain string) error {
	objectLock, err := getObjectLocking(bucketMeta, newLockMode, newRetain)
	if err != nil {
		return err
	}

	if objectLock != nil {
		logger.GetLogger(ctx).WithFields(map[string]any{
			"bucket":    bucketMeta.Name,
			"objectKey": meta.Key,
			"version":   meta.Name,
		}).Info("Enabling object locking.")
		meta.ObjectLock = objectLock
	}

	return nil
}

func (meta *ObjectVersion) CanBeDeleted(bucket string, bypassGovRetention bool) error {
	if meta.ObjectLock != nil {
		return meta.ObjectLock.CanBeDeleted(bucket, bypassGovRetention)
	} else {
		if bypassGovRetention {
			return s3common.GetInvalidRequestS3Error(bucket, "Object does not have Locking enabled to bypass.")
		}
	}

	return nil
}

func (meta *ObjectVersion) UpdateLegalHoldStatus(status s3common.LegalHoldStatus) {
	if meta.ObjectLock == nil {
		meta.ObjectLock = NewObjectLock()
	}

	meta.ObjectLock.UpdateLegalHoldStatus(status)
}

// MultipartUploadPart

func NewMultipartUploadPart(uuid, path, etag string, partNumber int32, size int64) *MultipartUploadPart {
	return &MultipartUploadPart{
		UploadId:     uuid,
		PartNumber:   partNumber,
		Path:         path,
		Etag:         etag,
		Size:         size,
		LastModified: utils.ConvertTimeToString(time.Now()),
	}
}

func DecodeMultipartUploadPart(data []byte) (*MultipartUploadPart, error) {
	var meta MultipartUploadPart
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *MultipartUploadPart) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// MultipartUpload

func NewMultipartUpload(bucket *BucketMeta, objectKey, lockMode, retainUntil, legalHold string) *MultipartUpload {
	return &MultipartUpload{
		Uuid:            uuid.New().String(),
		Bucket:          bucket.Name,
		Key:             objectKey,
		LockMode:        lockMode,
		RetainUntil:     retainUntil,
		LegalHoldStatus: legalHold,
		Owner:           bucket.Owner,
		Initiator:       bucket.Owner,
		Initiated:       utils.ConvertTimeToString(time.Now()),
	}
}

func DecodeMultipartUpload(data []byte) (*MultipartUpload, error) {
	var meta MultipartUpload
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *MultipartUpload) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// LifecycleConfig

func (meta *LifecycleConfig) Validate(bucket string) error {
	if len(meta.Rules) == 0 {
		return s3common.GetInvalidRequestS3Error(bucket, "At least one lifecycle rule must be specified.")
	}

	if len(meta.Rules) > s3common.MAX_ALLOWED_LIFECYCLE_RULES {
		msg := "The number of lifecycle rules must not exceed the allowed limit of 1000 rules."
		return s3common.GetInvalidRequestS3Error(bucket, msg)
	}

	for _, rule := range meta.Rules {
		if err := rule.Validate(bucket); err != nil {
			return err
		}
	}

	return nil
}

// LifeCycleRule

func (meta *LifeCycleRule) Validate(bucket string) error {
	if err := s3common.LifiCycleStatus(meta.Status).Validate(bucket); err != nil {
		return err
	}

	if meta.Expiration == nil &&
		meta.AbortIncompleteMultipartUpload == nil &&
		meta.NoncurrentVersionExpiration == nil {
		return s3common.GetInvalidRequestS3Error(bucket, "At least one action must be specified in a lifecycle rule.")
	}

	// TODO: Implement Tag validation

	return nil
}

func (meta *LifeCycleRule) IsEnabled() bool {
	return meta.Status == s3common.LifiCycleStatusEnabled.String()
}

// User

func NewUser(name, path, permBoundary string, tags *TagMap) *User {
	return &User{
		Name:                name,
		Uuid:                uuid.New().String(),
		Path:                path,
		PermissionsBoundary: permBoundary,
		CreateDate:          utils.ConvertTimeToString(time.Now()),
		Arn:                 s3common.GenerateUserARN(ConcatPathForArn(path, name)),
		Tags:                tags,
	}
}

func DecodeUser(data []byte) (*User, error) {
	var meta User
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *User) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// UserAccessKey

func NewUserAccessKey(username string) (*UserAccessKey, error) {
	accessKeyId, secretAccessKey, err := secure.CreateNewAccessAndSecretKey()
	if err != nil {
		return nil, err
	}

	return &UserAccessKey{
		UserName:        username,
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		Status:          s3common.AccessKeyStatusActive.String(),
		CreateDate:      utils.ConvertTimeToString(time.Now()),
	}, nil
}

func DecodeUserAccessKey(data []byte) (*UserAccessKey, error) {
	var meta UserAccessKey
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *UserAccessKey) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// IAMPolicy

func NewIAMPolicy(name, desc, document, path string, tags *TagMap) *IAMPolicy {
	now := utils.ConvertTimeToString(time.Now())
	return &IAMPolicy{
		Uuid:                  uuid.NewString(),
		Name:                  name,
		Arn:                   s3common.GeneratePolicyARN(ConcatPathForArn(path, name)),
		IsAttachable:          true,
		Description:           desc,
		Path:                  path,
		DefaultPolicyDocument: document,
		CreateDate:            now,
		UpdateDate:            now,
		NextVersionNumber:     1,
		IsManaged:             false,
		Tags:                  tags,
	}
}

func DecodeIAMPolicy(data []byte) (*IAMPolicy, error) {
	var meta IAMPolicy
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *IAMPolicy) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// IAMPolicyVersion

func NewIAMPolicyVersion(versionId string, document string) *IAMPolicyVersion {
	return &IAMPolicyVersion{
		VersionId:  versionId,
		CreateDate: utils.ConvertTimeToString(time.Now()),
		Document:   document,
	}
}

func DecodeIAMPolicyVersion(data []byte) (*IAMPolicyVersion, error) {
	var meta IAMPolicyVersion
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *IAMPolicyVersion) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// Group

func NewGroup(name, path string) *Group {
	return &Group{
		Uuid:       uuid.New().String(),
		Name:       name,
		Path:       path,
		CreateDate: utils.ConvertTimeToString(time.Now()),
		Arn:        s3common.GenerateGroupARN(ConcatPathForArn(path, name)),
	}
}

func DecodeGroup(data []byte) (*Group, error) {
	var meta Group
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *Group) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// AccessKeyLastUsed
func NewAccessKeyLastUsed(lastUsed string) *AccessKeyLastUsed {
	return &AccessKeyLastUsed{
		LastUsed: lastUsed,
	}
}

func DecodeAccessKeyLastUsed(data []byte) (*AccessKeyLastUsed, error) {
	var meta AccessKeyLastUsed
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *AccessKeyLastUsed) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// ResourceRef

func NewResourceRef(ref string) *ResourceRef {
	return &ResourceRef{
		Reference: ref,
	}
}

func DecodeResourceRef(data []byte) (*ResourceRef, error) {
	var meta ResourceRef
	if err := proto.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *ResourceRef) Encode() ([]byte, error) {
	return proto.Marshal(meta)
}

// Reused functions

func getObjectLocking(bucketMeta *BucketMeta, newLockMode, newRetain string) (*ObjectLock, error) {
	if bucketMeta.IsObjectLockingEnabled() {
		objectLock := NewObjectLock()

		if newLockMode != "" && newRetain != "" {
			// Precedence will be given to mode and retain value specified during object upload
			if err := objectLock.UpdateModeAndRetainUntil(bucketMeta.Name, newLockMode, newRetain); err != nil {
				return nil, err
			}
		} else {
			// If not specified during upload then fallback to bucket level config
			objectLock.RetainFromBucketConfig(bucketMeta.Name, bucketMeta.ObjectLocking)
			if objectLock.Mode == nil && *objectLock.Mode == "" {
				// This case is possible if user enabled objectLocking while creating a bucket
				// but has not updated configuration by calling PutObjectLockConfiguration()
				return nil, nil
			}
		}

		return objectLock, nil
	}

	if newLockMode != "" && newRetain != "" {
		msg := "Bucket does not have Object Locking enabled."
		return nil, s3common.GetInvalidRequestS3Error(bucketMeta.Name, msg)
	}

	return nil, nil
}

func ValidateTagKV(bucket string, k, v string) error {
	if len(k) > s3common.MAX_ALLOWED_TAG_KEY_LENGTH ||
		len(v) > s3common.MAX_ALLOWED_TAG_VALUE_LENGTH ||
		strings.HasPrefix(strings.ToLower(k), "aws:") ||
		!tagsKVRegex.MatchString(k) ||
		(v != "" && !tagsKVRegex.MatchString(v)) {
		return s3common.GetInvalidTagS3Error(bucket)
	}

	return nil
}

func ConcatPathForArn(path, resource string) string {
	if path == s3common.DEFAULT_IAM_RESOURCE_PATH {
		return resource
	}
	return strings.Trim(path, "/") + "/" + resource
}

func MergeProtoMessages(existingVal []byte, base, delta proto.Message) ([]byte, error) {
	if reflect.TypeOf(base) != reflect.TypeOf(delta) {
		return nil, fmt.Errorf("base and delta must be same type")
	}

	if len(existingVal) == 0 {
		return proto.Marshal(delta)
	}

	if err := proto.Unmarshal(existingVal, base); err != nil {
		return nil, err
	}

	proto.Merge(base, delta)
	return proto.Marshal(base)
}

func LoadModelsDescriptorSet() (*descriptorpb.FileDescriptorSet, error) {
	data, err := os.ReadFile("internal/core/models/models.pb")
	if err != nil {
		return nil, err
	}

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &fds); err != nil {
		return nil, err
	}

	return &fds, nil
}
