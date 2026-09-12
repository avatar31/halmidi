package iam

import (
	"context"
	"maps"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/avatar31/omashu"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

var (
	resourceNameRegex = regexp.MustCompile(s3common.IAMResourceNameRegex)
	resourcePathRegex = regexp.MustCompile(s3common.IAMResourcePathRegex)
)

type ListIAMResourceOptions struct {
	Marker     string
	PathPrefix string
	MaxItems   int
}

type CreateUserRequest struct {
	UserName            string
	Path                string
	PermissionsBoundary string
	Tags                *models.TagMap
}

func (r *CreateUserRequest) Validate(ctx context.Context) error {
	if len(r.UserName) == 0 ||
		len(r.UserName) > s3common.MAX_ALLOWED_USER_NAME_LENGTH ||
		!resourceNameRegex.MatchString(r.UserName) {
		return s3common.GetInvalidArgumentS3Error(r.UserName, "Invalid User name.")
	}

	if len(r.Path) == 0 {
		r.Path = s3common.DEFAULT_IAM_RESOURCE_PATH
	}

	if err := validatePath(r.Path); err != nil {
		return err
	}

	if len(r.PermissionsBoundary) != 0 {
		if len(r.PermissionsBoundary) < s3common.MIN_ALLOWED_PERM_BOUNDARY_LENGTH ||
			len(r.PermissionsBoundary) > s3common.MAX_ALLOWED_PERM_BOUNDARY_LENGTH {
			return s3common.GetInvalidArgumentS3Error(r.PermissionsBoundary, "Invalid Permissions Boundary length.")
		}
	}

	if r.Tags != nil {
		if err := r.Tags.Validate(""); err != nil {
			return err
		}
	}
	return nil
}

type UserService struct {
	log logger.Logger
	bdb *omashu.DistributedBadger
}

func NewUserService(ctx context.Context) *UserService {
	return &UserService{
		log: logger.GetLogger(ctx),
		bdb: dbstore.GetDBStore(ctx),
	}
}

func (s *UserService) IsExist(ctx context.Context, username string) bool {
	return s.bdb.Exists(ctx, dbstore.GetUserDBKeyWithNS(username))
}

func (s *UserService) ListUsers(ctx context.Context, opts ListIAMResourceOptions) ([]*models.User, string, error) {
	startAfter := ""
	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}
		startAfter = dbstore.GetUserDBKeyWithNS(decodedMarker)
	}

	result := []*models.User{}
	dbPrefix := dbstore.GetUserDBKeyWithNS("")
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxItems, func(k, v []byte) bool {
		var meta models.User
		err := proto.Unmarshal(v, &meta)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from users list")
			return false
		}

		if opts.PathPrefix != "" && !strings.HasPrefix(meta.Path, opts.PathPrefix) {
			return false
		}

		result = append(result, &meta)
		return true
	})
	if err != nil {
		s.log.WithError(err).Error("Error while reading user details from db")
		return nil, "", s3common.GetInternalErrorS3Error("")
	}

	if nextCursor != "" {
		// Trim the object namespace from the cursor before returning
		nextCursor = strings.TrimPrefix(nextCursor, dbPrefix)
	}

	return result, secure.EncodeSignedMarker(nextCursor), nil
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	if s.IsExist(ctx, req.UserName) {
		return nil, s3common.GetEntityAlreadyExistsS3Error(req.UserName)
	}

	user := models.NewUser(req.UserName, req.Path, req.PermissionsBoundary, req.Tags)
	userBytes, err := user.Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Error while encoding user metadata %s", req.UserName)
		return nil, s3common.GetInternalErrorS3Error(req.UserName)
	}

	err = s.bdb.Set(ctx, dbstore.GetUserDBKeyWithNS(req.UserName), userBytes)
	if err != nil {
		s.log.WithError(err).Errorf("Error while creating user %s", req.UserName)
		return nil, s3common.GetInternalErrorS3Error(req.UserName)
	}

	s.log.Infof("User %s created.", req.UserName)
	return user, nil
}

func (s *UserService) GetUserEntity(ctx context.Context, username string) (*UserEntity, error) {
	entity := &UserEntity{
		log: s.log.WithField("username", username),
		bdb: s.bdb,
	}

	err := entity.fetchMetadataFromDB(ctx, username)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type UserEntity struct {
	log   logger.Logger
	bdb   *omashu.DistributedBadger
	dbKey string
	meta  *models.User
}

func (e *UserEntity) fetchMetadataFromDB(ctx context.Context, username string) error {
	e.dbKey = dbstore.GetUserDBKeyWithNS(username)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user metadata from db")
		return s3common.GetInternalErrorS3Error(username)
	}
	if !ok {
		return s3common.GetNoSuchEntityS3Error(username)
	}

	data, err := models.DecodeUser(b)
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling user from db")
		return s3common.GetInternalErrorS3Error(username)
	}

	e.meta = data
	return nil
}

func (e *UserEntity) GetMetadata(ctx context.Context) *models.User {
	return e.meta
}

func (e *UserEntity) GetUserTags(ctx context.Context) *models.TagMap {
	return e.meta.Tags
}

func (e *UserEntity) AddUserTags(ctx context.Context, newTags *models.TagMap) error {
	if newTags == nil {
		return nil
	}

	mergedTags, err := mergeTags(e.meta.Tags, newTags)
	if err != nil {
		return err
	}

	err = e.updateUser(ctx, &models.User{Tags: mergedTags})
	if err != nil {
		return err
	}
	return nil
}

func (e *UserEntity) RemoveUserTags(ctx context.Context, keys []string) error {
	if len(keys) == 0 || e.meta.Tags == nil || len(e.meta.Tags.Items) == 0 {
		return nil
	}

	err := e.updateUser(ctx, &models.User{Tags: nil})
	if err != nil {
		return err
	}
	return nil
}

func (e *UserEntity) UpdateUser(ctx context.Context, newPath string) (*models.User, error) {
	err := validatePath(newPath)
	if err != nil {
		return nil, err
	}

	delta := &models.User{
		Path: newPath,
		Arn:  s3common.GenerateUserARN(models.ConcatPathForArn(newPath, e.meta.Name)),
	}

	err = e.updateUser(ctx, delta)
	if err != nil {
		return nil, err
	}

	// Update in-memory metadata
	e.meta.Path = newPath
	e.meta.Arn = delta.Arn

	return e.meta, nil
}

func (e *UserEntity) updateUser(ctx context.Context, delta *models.User) error {
	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating user")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Info("User updated.")
	return nil
}

func (e *UserEntity) ListGroupsForUser(ctx context.Context, opts ListIAMResourceOptions) ([]*models.Group,
	string, error) {
	availableGroups := make([]string, 0)
	refMap, err := e.bdb.GetByPrefix(ctx, dbstore.GetUserAttachedGroupDBKeyWithNS(e.meta.Name, ""))
	if err != nil {
		e.log.WithError(err).Error("Error while reading attached group users from db")
		return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	for _, v := range refMap {
		ref, err := models.DecodeResourceRef(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling attached user groups from db")
			return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
		}

		availableGroups = append(availableGroups, ref.Reference)
	}

	numItems := len(availableGroups)
	if numItems == 0 {
		return []*models.Group{}, "", nil
	}

	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			e.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}

		markerIndex, found := utils.SearchSorted(availableGroups, decodedMarker)
		if !found {
			e.log.WithField("marker", decodedMarker).Error("Marker not found in user groups list")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}

		// Move to next item after marker
		markerIndex++
		if markerIndex >= numItems {
			return []*models.Group{}, "", nil
		}

		availableGroups = availableGroups[markerIndex:]
		numItems = len(availableGroups)
	}

	nextCursor := ""
	if opts.MaxItems > 0 && numItems > opts.MaxItems {
		numItems = opts.MaxItems
		nextCursor = availableGroups[numItems-1]
	}

	dbKeys := make([]string, 0, numItems)
	for _, groupname := range availableGroups[:numItems] {
		dbKeys = append(dbKeys, dbstore.GetGroupDBKeyWithNS(groupname))
	}

	groupsMap, err := e.bdb.BulkGet(ctx, dbKeys)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user group details from db")
		return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	groups := make([]*models.Group, 0)
	for _, b := range groupsMap {
		group, err := models.DecodeGroup(b)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling group from db")
			return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
		}
		groups = append(groups, group)
	}

	return groups, secure.EncodeSignedMarker(nextCursor), nil
}

func (e *UserEntity) AttachPolicyToUser(ctx context.Context, policyArn string) error {
	policypath, err := getPolicyPathFromARN(policyArn)
	if err != nil {
		return s3common.GetInvalidArgumentS3Error(e.meta.Name, "Invalid Policy ARN.")
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		exist := NewIAMPolicyService(ctx).IsExist(ctx, policypath)
		if !exist {
			return s3common.GetNoSuchEntityS3Error(e.meta.Name, "Policy does not exist.")
		}
		return nil
	})

	g.Go(func() error {
		currentPoliciesCount, err := e.getUserPoliciesCount(ctx)
		if err != nil {
			return err
		}
		if currentPoliciesCount >= s3common.MAX_ALLOWED_POLICIES_PER_ENTITY {
			return s3common.GetLimitExceededS3Error(e.meta.Name, "Maximum number of policies attached to user exceeded.")
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		if s3common.IsS3Error(err) {
			return err
		}
		e.log.WithError(err).Error("Error while counting user policies")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	policyRef := models.NewResourceRef(policypath)
	pb, err := policyRef.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding policy reference")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	userRef := models.NewResourceRef(e.meta.Name)
	ub, err := userRef.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding user reference")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	policyRefDbKey := dbstore.GetUserAttachedPolicyDBKeyWithNS(e.meta.Name, policypath)
	userRefDbKey := dbstore.GetPolicyAttachedUserDBKeyWithNS(policypath, e.meta.Name)
	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, policyRefDbKey, pb)
		if err != nil {
			return err
		}
		return txn.Set(ctx, userRefDbKey, ub)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while attaching policy %s to user", policyArn)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, e.meta.Name)
	e.log.Infof("Policy %s attached to user.", policyArn)
	return nil
}

func (e *UserEntity) DetachPolicyFromUser(ctx context.Context, policyArn string) error {
	policypath, err := getPolicyPathFromARN(policyArn)
	if err != nil {
		return s3common.GetInvalidArgumentS3Error(e.meta.Name, "Invalid Policy ARN.")
	}

	policyRefDbKey := dbstore.GetUserAttachedPolicyDBKeyWithNS(e.meta.Name, policypath)
	userRefDbKey := dbstore.GetPolicyAttachedUserDBKeyWithNS(policypath, e.meta.Name)
	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, policyRefDbKey)
		if err != nil {
			return err
		}
		return txn.Delete(ctx, userRefDbKey)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while detaching policy %s from user", policyArn)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, e.meta.Name)
	return nil
}

func (e *UserEntity) ListAttachedUserPolicies(ctx context.Context,
	opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	refResult, err := e.bdb.GetByPrefix(ctx, dbstore.GetUserAttachedPolicyDBKeyWithNS(e.meta.Name, ""))
	if err != nil {
		e.log.WithError(err).Error("Error while reading attached user policies from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	policyKeys := []string{}
	for _, v := range refResult {
		ref, err := models.DecodeResourceRef(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling attached user policy from db")
			return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
		}

		if opts.PathPrefix == "" || strings.HasPrefix(ref.Reference, opts.PathPrefix) {
			policyKeys = append(policyKeys, dbstore.GetPolicyDBKeyWithNS(ref.Reference))
		}
	}

	if len(policyKeys) == 0 {
		return []*models.IAMPolicy{}, nil
	}

	var availablePolicies []*models.IAMPolicy
	result, err := e.bdb.BulkGet(ctx, policyKeys)
	if err != nil {
		e.log.WithError(err).Error("Error while reading attached user policies from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	for _, v := range result {
		policy, err := models.DecodeIAMPolicy(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling attched user policy from db")
			return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
		}
		availablePolicies = append(availablePolicies, policy)
	}

	return availablePolicies, nil
}

func (e *UserEntity) listUserInlinePolicies(ctx context.Context,
	opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	result, err := e.bdb.GetByPrefix(ctx, dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, ""))
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling inline user policy from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	availablePolicies := []*models.IAMPolicy{}
	for _, v := range result {
		policy, err := models.DecodeIAMPolicy(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling inline user policy from db")
			return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
		}

		if opts.PathPrefix == "" || strings.HasPrefix(policy.Path, opts.PathPrefix) {
			availablePolicies = append(availablePolicies, policy)
		}
	}

	return availablePolicies, nil
}

func (e *UserEntity) ListUserPolicies(ctx context.Context, opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	var attachedPolicies, inlinePolicies []*models.IAMPolicy
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		attachedPolicies, err = e.ListAttachedUserPolicies(ctx, ListIAMResourceOptions{PathPrefix: opts.PathPrefix})
		return err
	})

	g.Go(func() error {
		var err error
		inlinePolicies, err = e.listUserInlinePolicies(ctx, ListIAMResourceOptions{PathPrefix: opts.PathPrefix})
		return err
	})

	if err := g.Wait(); err != nil {
		e.log.WithError(err).Error("Error while listing user policies")
		return nil, err
	}

	return append(attachedPolicies, inlinePolicies...), nil
}

func (e *UserEntity) getUserPoliciesCount(ctx context.Context) (int64, error) {
	var currentPoliciesCount int64 = 0
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		prefix := dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, "")
		atomic.AddInt64(&currentPoliciesCount, int64(e.bdb.Count(ctx, prefix)))
		return nil
	})

	g.Go(func() error {
		prefix := dbstore.GetUserAttachedPolicyDBKeyWithNS(e.meta.Name, "")
		atomic.AddInt64(&currentPoliciesCount, int64(e.bdb.Count(ctx, prefix)))
		return nil
	})

	if err := g.Wait(); err != nil {
		e.log.WithError(err).Error("Error while counting user policies")
		return 0, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return currentPoliciesCount, nil
}

// User Inline Policy
func (e *UserEntity) AddUserPolicy(ctx context.Context, req CreatePolicyRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	currentPoliciesCount, err := e.getUserPoliciesCount(ctx)
	if err != nil {
		return err
	}

	if currentPoliciesCount >= s3common.MAX_ALLOWED_POLICIES_PER_ENTITY {
		return s3common.GetLimitExceededS3Error(e.meta.Name, "Maximum number of policies attached to user exceeded.")
	}

	policy := models.NewIAMPolicy(req.PolicyName, req.Description, req.PolicyDocument, req.Path, nil)
	policy.DefaultVersionId = "v1"

	b, err := policy.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding inline policy")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	policyDBKey := dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, req.PolicyName)
	err = e.bdb.Set(ctx, policyDBKey, b)
	if err != nil {
		e.log.WithError(err).Errorf("Error while adding inline policy %s to user", req.PolicyName)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, e.meta.Name)
	return nil
}

// User Inline Policy
func (e *UserEntity) DeleteUserPolicy(ctx context.Context, policyName string) error {
	policyDBKey := dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, policyName)
	err := e.bdb.Delete(ctx, policyDBKey)
	if err != nil {
		e.log.WithError(err).Errorf("Error while deleting inline policy %s from user", policyName)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, e.meta.Name)
	return nil
}

func (e *UserEntity) GetUserPolicy(ctx context.Context, policyName string) (*models.IAMPolicy, error) {
	policyDBKey := dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, policyName)
	b, ok, err := e.bdb.Get(ctx, policyDBKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading inline policy from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(policyName, "Policy does not exist.")
	}

	policy, err := models.DecodeIAMPolicy(b)
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling inline policy from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return policy, nil
}

func (e *UserEntity) DeleteUser(ctx context.Context) error {
	if e.bdb.HasChild(ctx, dbstore.GetUserAttachedGroupDBKeyWithNS(e.meta.Name, "")) ||
		e.bdb.HasChild(ctx, dbstore.GetUserAttachedPolicyDBKeyWithNS(e.meta.Name, "")) ||
		e.bdb.HasChild(ctx, dbstore.GetUserInlinePolicyDBKeyWithNS(e.meta.Name, "")) {
		return s3common.GetDeleteConflictS3Error(e.meta.Name, "User has one or more attached entities. Detach all entities before deleting it.")
	}

	err := e.bdb.Delete(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while deleting user")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Info("User deleted.")
	return nil
}

func (e *UserEntity) MergeDelta(ctx context.Context, delta *models.User) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Name)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.User{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging policy delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return b, nil
}

func validatePath(path string) error {
	if len(path) == 0 {
		return s3common.GetInvalidArgumentS3Error(path, "Path cannot be empty.")
	}
	if len(path) > s3common.MAX_ALLOWED_IAM_RES_PATH_LENGTH {
		return s3common.GetInvalidArgumentS3Error(path, "Path length exceeds maximum allowed length.")
	}
	if !resourcePathRegex.MatchString(path) {
		return s3common.GetInvalidArgumentS3Error(path, "Invalid path format.")
	}
	return nil
}

func mergeTags(old, new *models.TagMap) (*models.TagMap, error) {
	if err := new.Validate(""); err != nil {
		return nil, err
	}

	if old == nil || len(old.Items) == 0 {
		return new, nil
	}

	merged := &models.TagMap{Items: map[string]string{}}
	maps.Copy(merged.Items, old.Items)
	maps.Copy(merged.Items, new.Items)
	if len(merged.Items) > s3common.MAX_ALLOWED_TAGS {
		return nil, s3common.GetLimitExceededS3Error("", "Maximum number of tags for user exceeded.")
	}

	return merged, nil
}
