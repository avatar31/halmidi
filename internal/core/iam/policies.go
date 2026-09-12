package iam

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/avatar31/omashu"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
)

var (
	versionIdRegex = regexp.MustCompile(s3common.VersionIdRegex)
)

type CreatePolicyRequest struct {
	PolicyName     string         `json:"policyName"`
	PolicyDocument string         `json:"policyDocument"`
	Description    string         `json:"description"`
	Path           string         `json:"path"`
	Tags           *models.TagMap `json:"tags"`
}

func (r *CreatePolicyRequest) Validate() error {
	if len(r.Description) > s3common.MAX_ALLOWED_POLICY_DESC_LENGTH {
		return s3common.GetInvalidArgumentS3Error("", "Policy description length exceeds the maximum allowed limit.")
	}

	if len(r.Path) == 0 {
		r.Path = s3common.DEFAULT_IAM_RESOURCE_PATH
	}

	if err := validatePath(r.Path); err != nil {
		return err
	}

	if len(r.PolicyName) == 0 ||
		len(r.PolicyName) > s3common.MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH ||
		!resourceNameRegex.MatchString(r.PolicyName) {
		return s3common.GetInvalidArgumentS3Error(r.PolicyName, "Invalid Policy name.")
	}

	if r.Tags != nil {
		if err := r.Tags.Validate(""); err != nil {
			return err
		}
	}

	if err := ValidatePolicyDocument(r.PolicyDocument); err != nil {
		return err
	}

	return nil
}

type ListPoliciesOptions struct {
	ListIAMResourceOptions
	OnlyAttached      *bool
	PolicyUsageFilter string
}

type IAMPolicyService struct {
	log logger.Logger
	bdb *omashu.DistributedBadger
}

func NewIAMPolicyService(ctx context.Context) *IAMPolicyService {
	return &IAMPolicyService{
		log: logger.GetLogger(ctx),
		bdb: dbstore.GetDBStore(ctx),
	}
}

func (s *IAMPolicyService) IsExist(ctx context.Context, policyPath string) bool {
	return s.bdb.Exists(ctx, dbstore.GetPolicyDBKeyWithNS(policyPath))
}

func (s *IAMPolicyService) CreateIAMPolicy(ctx context.Context, req CreatePolicyRequest) (*models.IAMPolicy, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	policyPath := models.ConcatPathForArn(req.Path, req.PolicyName)
	if s.IsExist(ctx, policyPath) {
		return nil, s3common.GetEntityAlreadyExistsS3Error(req.PolicyName)
	}

	// TODO: Add tags
	policy := models.NewIAMPolicy(req.PolicyName, req.Description, req.PolicyDocument, req.Path, req.Tags)
	versionId := fmt.Sprintf("v%d", policy.NextVersionNumber)

	policy.DefaultVersionId = versionId
	policy.NextVersionNumber++
	policyBytes, err := policy.Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while marshalling policy %s", req.PolicyName)
		return nil, s3common.GetInternalErrorS3Error(req.PolicyName)
	}

	policyVersionBytes, err := models.NewIAMPolicyVersion(versionId, req.PolicyDocument).Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while marshalling policy version for policy %s", req.PolicyName)
		return nil, s3common.GetInternalErrorS3Error(req.PolicyName)
	}

	err = s.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, dbstore.GetPolicyDBKeyWithNS(policyPath), policyBytes)
		if err != nil {
			return err
		}

		policyVersionKey := dbstore.GetPolicyVersionDBKeyWithNS(policy.Uuid, versionId)
		return txn.Set(ctx, policyVersionKey, policyVersionBytes)
	})

	if err != nil {
		s.log.WithError(err).Errorf("Error while creating policy %s", req.PolicyName)
		return nil, s3common.GetInternalErrorS3Error(req.PolicyName)
	}

	s.log.Infof("Policy %s created.", req.PolicyName)
	return policy, nil
}

func (s *IAMPolicyService) ListIAMPolicies(ctx context.Context, opts ListPoliciesOptions) ([]*models.IAMPolicy,
	string, error) {
	startAfter := ""
	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}
		startAfter = dbstore.GetPolicyDBKeyWithNS(decodedMarker)
	}

	result := []*models.IAMPolicy{}
	dbPrefix := dbstore.GetPolicyDBKeyWithNS(opts.PathPrefix)
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxItems, func(k, v []byte) bool {
		meta, err := models.DecodeIAMPolicy(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from policy list")
			return false
		}

		if opts.OnlyAttached != nil && *opts.OnlyAttached &&
			(s.bdb.HasChild(ctx, dbstore.GetPolicyAttachedUserDBKeyWithNS(meta.Path, "")) ||
				s.bdb.HasChild(ctx, dbstore.GetPolicyAttachedGroupDBKeyWithNS(meta.Path, ""))) {
			// TODO: Handle PolicyUsageFilter
			result = append(result, meta)
			return true
		}

		return false
	})
	if err != nil {
		s.log.WithError(err).Error("Error while reading object details from db")
		return nil, "", s3common.GetInternalErrorS3Error("")
	}

	if nextCursor != "" {
		// Trim the object namespace from the cursor before returning
		nextCursor = strings.TrimPrefix(nextCursor, dbstore.GetPolicyDBKeyWithNS(""))
	}

	return result, secure.EncodeSignedMarker(nextCursor), nil
}

func (s *IAMPolicyService) GetIAMPolicyEntity(ctx context.Context, arn string) (*PolicyEntity, error) {
	entity := &PolicyEntity{
		log: s.log,
		bdb: s.bdb,
	}

	if !s3common.IsValidARNLength(arn) {
		return nil, s3common.GetInvalidArgumentS3Error(arn, "Invalid PolicyArn.")
	}

	err := entity.fetchMetadataFromDB(ctx, arn)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type PolicyEntity struct {
	log   logger.Logger
	bdb   *omashu.DistributedBadger
	dbKey string
	meta  *models.IAMPolicy
}

func (e *PolicyEntity) fetchMetadataFromDB(ctx context.Context, arn string) error {
	policyPath, err := getPolicyPathFromARN(arn)
	if err != nil {
		return err
	}

	e.dbKey = dbstore.GetPolicyDBKeyWithNS(policyPath)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while fetching policy from DB")
		return s3common.GetInternalErrorS3Error(arn)
	}
	if !ok {
		return s3common.GetNoSuchEntityS3Error(arn)
	}

	data, err := models.DecodeIAMPolicy(b)
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling policy data from DB")
		return s3common.GetInternalErrorS3Error(arn)
	}

	e.meta = data
	return nil
}

func (e *PolicyEntity) GetMetadata(ctx context.Context) *models.IAMPolicy {
	return e.meta
}

func (e *PolicyEntity) GetPolicyTags(ctx context.Context) *models.TagMap {
	return e.meta.Tags
}

func (e *PolicyEntity) AddPolicyTags(ctx context.Context, newTags *models.TagMap) error {
	if newTags == nil {
		return nil
	}

	mergedTags, err := mergeTags(e.meta.Tags, newTags)
	if err != nil {
		return err
	}

	return e.updatePolicy(ctx, &models.IAMPolicy{Tags: mergedTags})
}

func (e *PolicyEntity) RemovePolicyTags(ctx context.Context, keys []string) error {
	if len(keys) == 0 || e.meta.Tags == nil || len(e.meta.Tags.Items) == 0 {
		return nil
	}

	return e.updatePolicy(ctx, &models.IAMPolicy{Tags: nil})
}

func (e *PolicyEntity) ListEntities(ctx context.Context, pathPrefix string) ([]*models.User,
	[]*models.Group, error) {
	var usernames, groupnames []string

	refG, ctx := errgroup.WithContext(ctx)
	refG.Go(func() error {
		refMap, err := e.bdb.GetByPrefix(ctx, dbstore.GetPolicyAttachedUserDBKeyWithNS(e.meta.Path, ""))
		if err != nil {
			return err
		}

		for _, b := range refMap {
			ref, err := models.DecodeResourceRef(b)
			if err != nil {
				return err
			}

			usernames = append(usernames, dbstore.GetUserDBKeyWithNS(ref.Reference))
		}
		return nil
	})

	refG.Go(func() error {
		refMap, err := e.bdb.GetByPrefix(ctx, dbstore.GetPolicyAttachedGroupDBKeyWithNS(e.meta.Path, ""))
		if err != nil {
			return err
		}

		for _, b := range refMap {
			ref, err := models.DecodeResourceRef(b)
			if err != nil {
				return err
			}

			groupnames = append(groupnames, dbstore.GetUserDBKeyWithNS(ref.Reference))
		}
		return nil
	})

	if err := refG.Wait(); err != nil {
		e.log.WithError(err).Error("Error while listing user and groups")
		return nil, nil, err
	}

	users := []*models.User{}
	groups := []*models.Group{}

	metaG, ctx := errgroup.WithContext(ctx)
	metaG.Go(func() error {
		userMap, err := e.bdb.BulkGet(ctx, usernames)
		if err != nil {
			return err
		}

		for _, b := range userMap {
			user, err := models.DecodeUser(b)
			if err != nil {
				return err
			}

			if strings.HasPrefix(user.Path, pathPrefix) {
				users = append(users, user)
			}
		}

		return nil
	})

	metaG.Go(func() error {
		groupMap, err := e.bdb.BulkGet(ctx, groupnames)
		if err != nil {
			return err
		}

		for _, b := range groupMap {
			group, err := models.DecodeGroup(b)
			if err != nil {
				return err
			}

			if strings.HasPrefix(group.Path, pathPrefix) {
				groups = append(groups, group)
			}
		}
		return nil
	})

	if err := metaG.Wait(); err != nil {
		e.log.WithError(err).Error("Error while listing user and groups")
		return nil, nil, err
	}

	return users, groups, nil
}

func (e *PolicyEntity) updatePolicy(ctx context.Context, delta *models.IAMPolicy) error {
	b, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, b)
	if err != nil {
		e.log.WithError(err).Error("Error while updating policy")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Infof("Policy updated.")
	return nil
}

func (e *PolicyEntity) DeletePolicy(ctx context.Context) error {
	if e.meta.IsManaged {
		errMsg := "IAM managed policies can not be deleted."
		return s3common.GetDeleteConflictS3Error(e.meta.Name, errMsg)
	}

	if e.bdb.HasChild(ctx, dbstore.GetPolicyAttachedUserDBKeyWithNS(e.meta.Path, "")) ||
		e.bdb.HasChild(ctx, dbstore.GetPolicyAttachedGroupDBKeyWithNS(e.meta.Path, "")) {
		errMsg := "Cannot delete policy while it is still attached to some users or groups."
		return s3common.GetDeleteConflictS3Error(e.meta.Name, errMsg)
	}

	if NewIAMPolicyVersionService(ctx, e.meta).CountVersions(ctx) > 1 {
		errMsg := "Cannot delete the default version. Set another version as default first."
		return s3common.GetDeleteConflictS3Error(e.meta.Name, errMsg)
	}

	err := e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		defaultVersionKey := dbstore.GetPolicyVersionDBKeyWithNS(e.meta.Uuid, e.meta.DefaultVersionId)
		if err := txn.Delete(ctx, defaultVersionKey); err != nil {
			return err
		}

		return txn.Delete(ctx, e.dbKey)
	})

	if err != nil {
		e.log.WithError(err).Errorf("Error while deleting policy %s", e.meta.Name)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Infof("Policy %s deleted.", e.meta.Name)
	return nil
}

func (e *PolicyEntity) MergeDelta(ctx context.Context, delta *models.IAMPolicy) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user policy details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Name)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.IAMPolicy{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging policy delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return b, nil
}

func (e *PolicyEntity) CreatePolicyVersion(ctx context.Context, policyDocument string,
	setAsDefault bool) (*models.IAMPolicyVersion, error) {
	service := NewIAMPolicyVersionService(ctx, e.meta)
	iamPolicyVersion, err := service.CreatePolicyVersion(ctx, policyDocument, setAsDefault)
	if err != nil {
		return nil, err
	}

	if setAsDefault {
		go func() {
			_ = e.resetAuthzEngineForPolicyAttachedEntites(context.WithoutCancel(ctx))
		}()
	}

	return iamPolicyVersion, nil
}

func (e *PolicyEntity) resetAuthzEngineForPolicyAttachedEntites(ctx context.Context) error {
	users, groups, err := e.ListEntities(ctx, "")
	if err != nil {
		e.log.WithError(err).Warning("Error while listing entities attached to policy for resetting authz " +
			"engine cache")
		return err
	}

	usersMap := make(map[string]struct{}, 0)
	for _, group := range groups {
		groupEntity, err := NewGroupService(ctx).GetGroupEntity(ctx, group.Arn)
		if err != nil {
			e.log.WithError(err).Warningf("Error while fetching group entity for group %s for resetting "+
				"authz engine cache", group.Name)
			continue
		}

		uMap, err := groupEntity.resetAuthzEngineForGroupUsers(ctx)
		if err != nil {
			e.log.WithError(err).Warningf("Error while resetting authz engine cache for users of group %s",
				group.Name)
		}

		for u := range uMap {
			usersMap[u] = struct{}{}
		}
	}

	authzEngine := GetAuthzEngine()
	for _, user := range users {
		if _, ok := usersMap[user.Name]; !ok {
			authzEngine.Reset(ctx, user.Name)
		}
	}

	return nil
}

type IAMPolicyVersionService struct {
	log       logger.Logger
	bdb       *omashu.DistributedBadger
	IAMPolicy *models.IAMPolicy
}

func NewIAMPolicyVersionService(ctx context.Context, iamPolicy *models.IAMPolicy) *IAMPolicyVersionService {
	return &IAMPolicyVersionService{
		log:       logger.GetLogger(ctx).WithField("policyName", iamPolicy.Name),
		bdb:       dbstore.GetDBStore(ctx),
		IAMPolicy: iamPolicy,
	}
}

func (s *IAMPolicyVersionService) CountVersions(ctx context.Context) int {
	return s.bdb.Count(ctx, dbstore.GetPolicyVersionDBKeyWithNS(s.IAMPolicy.Uuid, ""))
}

func (s *IAMPolicyVersionService) CreatePolicyVersion(ctx context.Context, policyDocument string,
	setAsDefault bool) (*models.IAMPolicyVersion, error) {
	if err := ValidatePolicyDocument(policyDocument); err != nil {
		return nil, err
	}

	// Check if document is same as existing version
	if setAsDefault && s.IAMPolicy.DefaultPolicyDocument == policyDocument {
		return nil, s3common.GetInvalidArgumentS3Error("",
			"New version document is identical to default version.")
	}

	// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_managed-versioning.html#version-limits
	if s.CountVersions(ctx) >= s3common.MAX_ALLOWED_POLICY_VERSIONS {
		return nil, s3common.GetLimitExceededS3Error(s.IAMPolicy.Name, "Maximum number of policy versions exceeded.")
	}

	versionId := fmt.Sprintf("v%d", s.IAMPolicy.NextVersionNumber)
	delta := &models.IAMPolicy{NextVersionNumber: s.IAMPolicy.NextVersionNumber + 1}
	if setAsDefault {
		delta.DefaultVersionId = versionId
		delta.DefaultPolicyDocument = policyDocument
	}

	policyVersion := models.NewIAMPolicyVersion(versionId, policyDocument)
	b, err := policyVersion.Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while marshalling policy version %s", versionId)
		return nil, s3common.GetInternalErrorS3Error(s.IAMPolicy.Name)
	}

	policyEntity, err := NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, s.IAMPolicy.Arn)
	if err != nil {
		return nil, err
	}

	latestPolicyMeta := policyEntity.GetMetadata(ctx)
	proto.Merge(latestPolicyMeta, delta)

	updatedPolicy, err := latestPolicyMeta.Encode()
	if err != nil {
		s.log.WithError(err).Error("Error while marshalling updated policy data")
		return nil, s3common.GetInternalErrorS3Error(s.IAMPolicy.Name)
	}

	err = s.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, policyEntity.dbKey, updatedPolicy)
		if err != nil {
			return err
		}

		policyVersionDbKey := dbstore.GetPolicyVersionDBKeyWithNS(s.IAMPolicy.Uuid, versionId)
		return txn.Set(ctx, policyVersionDbKey, b)
	})

	if err != nil {
		s.log.WithError(err).Errorf("Error while creating policy version")
		return nil, s3common.GetInternalErrorS3Error(s.IAMPolicy.Name)
	}

	s.log.Infof("Policy version %s created.", versionId)
	return policyVersion, nil
}

func (s *IAMPolicyVersionService) ListIAMPolicyVersions(ctx context.Context,
	opts ListIAMResourceOptions) ([]*models.IAMPolicyVersion, string, error) {
	startAfter := ""
	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}
		startAfter = dbstore.GetPolicyVersionDBKeyWithNS(s.IAMPolicy.Uuid, decodedMarker)
	}

	result := []*models.IAMPolicyVersion{}
	dbPrefix := dbstore.GetPolicyVersionDBKeyWithNS(s.IAMPolicy.Uuid, "")
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxItems, func(k, v []byte) bool {
		meta, err := models.DecodeIAMPolicyVersion(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from policy versions list")
			return false
		}

		result = append(result, meta)
		return true
	})
	if err != nil {
		s.log.WithError(err).Error("Error while reading object details from db")
		return nil, "", s3common.GetInternalErrorS3Error("")
	}

	if nextCursor != "" {
		// Trim the object namespace from the cursor before returning
		nextCursor = strings.TrimPrefix(nextCursor, dbPrefix)
	}

	return result, nextCursor, nil
}

func (s *IAMPolicyVersionService) GetIAMPolicyVersionEntity(ctx context.Context,
	versionId string) (*PolicyVersionEntity, error) {
	if versionId == "" || !versionIdRegex.MatchString(versionId) {
		return nil, s3common.GetInvalidArgumentS3Error(s.IAMPolicy.Name, "Invalid VersionId.")
	}

	entity := &PolicyVersionEntity{
		log:       s.log.WithField("versionId", versionId),
		bdb:       s.bdb,
		IAMPolicy: s.IAMPolicy,
	}

	err := entity.fetchMetadataFromDB(ctx, versionId)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type PolicyVersionEntity struct {
	log       logger.Logger
	bdb       *omashu.DistributedBadger
	IAMPolicy *models.IAMPolicy
	dbKey     string
	meta      *models.IAMPolicyVersion
}

func (e *PolicyVersionEntity) GetMetadata(ctx context.Context) *models.IAMPolicyVersion {
	return e.meta
}

func (e *PolicyVersionEntity) fetchMetadataFromDB(ctx context.Context, versionId string) error {
	e.dbKey = dbstore.GetPolicyVersionDBKeyWithNS(e.IAMPolicy.Uuid, versionId)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while fetching policy version from DB")
		return s3common.GetInternalErrorS3Error(e.IAMPolicy.Name)
	}
	if !ok {
		return s3common.GetNoSuchEntityS3Error(e.IAMPolicy.Name)
	}

	data, err := models.DecodeIAMPolicyVersion(b)
	if err != nil {
		e.log.WithError(err).Error("Error while decoding policy version metadata from db")
		return s3common.GetInternalErrorS3Error("")
	}

	e.meta = data
	return nil
}

func (e *PolicyVersionEntity) SetDefaultIAMPolicyVersion(ctx context.Context) error {
	if e.IAMPolicy.DefaultVersionId == e.meta.VersionId {
		return nil
	}

	policyEntity, err := NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, e.IAMPolicy.Arn)
	if err != nil {
		return err
	}

	delta := &models.IAMPolicy{
		DefaultVersionId:      e.meta.VersionId,
		DefaultPolicyDocument: e.meta.Document,
	}

	err = policyEntity.updatePolicy(ctx, delta)
	if err != nil {
		return err
	}

	go func() {
		_ = policyEntity.resetAuthzEngineForPolicyAttachedEntites(context.WithoutCancel(ctx))
	}()
	return nil
}

func (e *PolicyVersionEntity) DeleteIAMPolicyVersion(ctx context.Context) error {
	if e.IAMPolicy.DefaultVersionId == e.meta.VersionId {
		return s3common.GetDeleteConflictS3Error(e.IAMPolicy.Name, "Cannot delete the default policy version.")
	}

	err := e.bdb.Delete(ctx, dbstore.GetPolicyVersionDBKeyWithNS(e.IAMPolicy.Uuid, e.meta.VersionId))
	if err != nil {
		e.log.WithError(err).Error("Error while deleting policy version")
		return s3common.GetInternalErrorS3Error(e.IAMPolicy.Name)
	}

	e.log.Info("Policy version deleted")
	return nil
}

func getPolicyPathFromARN(arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) != s3common.ARN_PARTS_COUNT {
		return "", s3common.GetNoSuchEntityS3Error("")
	}

	resourcePart := parts[5]
	resourceParts := strings.SplitN(resourcePart, "/", 2)
	if len(resourceParts) != 2 || resourceParts[0] != "policy" || resourceParts[1] == "" {
		return "", s3common.GetNoSuchEntityS3Error("")
	}

	return resourceParts[1], nil
}
