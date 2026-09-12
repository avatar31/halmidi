package iam

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/avatar31/omashu"
	"golang.org/x/sync/errgroup"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/core/secure"
	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

type CreateGroupRequest struct {
	Name string
	Path string
}

func (r *CreateGroupRequest) Validate(ctx context.Context) error {
	if len(r.Name) == 0 ||
		len(r.Name) > s3common.MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH ||
		!resourceNameRegex.MatchString(r.Name) {
		return s3common.GetInvalidArgumentS3Error(r.Name, "Invalid Group name.")
	}

	if len(r.Path) == 0 {
		r.Path = s3common.DEFAULT_IAM_RESOURCE_PATH
	}

	if err := validatePath(r.Path); err != nil {
		return err
	}

	return nil
}

type UpdateGroupRequest struct {
	Path  *string
	Users *[]string
}

func (r *UpdateGroupRequest) Validate(ctx context.Context) error {
	if r.Path != nil {
		err := validatePath(*r.Path)
		if err != nil {
			return err
		}
	}

	if r.Users != nil {
		for _, username := range *r.Users {
			if len(username) == 0 {
				return s3common.GetInvalidArgumentS3Error(username, "Invalid User name.")
			}
		}
	}

	return nil
}

type GroupService struct {
	log logger.Logger
	bdb *omashu.DistributedBadger
}

func NewGroupService(ctx context.Context) *GroupService {
	return &GroupService{
		log: logger.GetLogger(ctx),
		bdb: dbstore.GetDBStore(ctx),
	}
}

func (s *GroupService) IsExist(ctx context.Context, groupname string) bool {
	return s.bdb.Exists(ctx, dbstore.GetGroupDBKeyWithNS(groupname))
}

func (s *GroupService) CreateGroup(ctx context.Context, req CreateGroupRequest) (*models.Group, error) {
	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	if s.IsExist(ctx, req.Name) {
		return nil, s3common.GetEntityAlreadyExistsS3Error(req.Name)
	}

	group := models.NewGroup(req.Name, req.Path)
	groupBytes, err := group.Encode()
	if err != nil {
		s.log.WithError(err).Errorf("Failed to encode group metadata %s", req.Name)
		return nil, s3common.GetInternalErrorS3Error(req.Name)
	}

	err = s.bdb.Set(ctx, dbstore.GetGroupDBKeyWithNS(req.Name), groupBytes)
	if err != nil {
		s.log.Errorf("Failed to create group %s: %v", req.Name, err)
		return nil, s3common.GetInternalErrorS3Error(req.Name)
	}

	s.log.Infof("Group %s created.", req.Name)
	return group, nil
}

func (s *GroupService) ListGroups(ctx context.Context, opts ListIAMResourceOptions) ([]*models.Group, string, error) {
	startAfter := ""
	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			s.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}
		startAfter = dbstore.GetGroupDBKeyWithNS(decodedMarker)
	}

	result := []*models.Group{}
	dbPrefix := dbstore.GetGroupDBKeyWithNS("")
	nextCursor, err := s.bdb.IterateByPrefix(ctx, dbPrefix, startAfter, &opts.MaxItems, func(k, v []byte) bool {
		meta, err := models.DecodeGroup(v)
		if err != nil {
			s.log.WithError(err).Error("Error while unmarshalling iterated data from group list")
			return false
		}

		if opts.PathPrefix != "" && !strings.HasPrefix(meta.Path, opts.PathPrefix) {
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

	return result, secure.EncodeSignedMarker(nextCursor), nil
}

func (s *GroupService) GetGroupEntity(ctx context.Context, groupname string) (*GroupEntity, error) {
	entity := &GroupEntity{
		log: s.log.WithField("groupname", groupname),
		bdb: s.bdb,
	}

	err := entity.fetchMetadataFromDB(ctx, groupname)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type GroupEntity struct {
	log   logger.Logger
	bdb   *omashu.DistributedBadger
	dbKey string
	meta  *models.Group
}

func (e *GroupEntity) fetchMetadataFromDB(ctx context.Context, groupname string) error {
	e.dbKey = dbstore.GetGroupDBKeyWithNS(groupname)
	b, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading group metadata from db")
		return s3common.GetInternalErrorS3Error(groupname)
	}
	if !ok {
		return s3common.GetNoSuchEntityS3Error(groupname)
	}

	data, err := models.DecodeGroup(b)
	if err != nil {
		e.log.WithError(err).Error("Error while decoding group metadata from db")
		return s3common.GetInternalErrorS3Error(groupname)
	}

	e.meta = data
	return nil
}

func (e *GroupEntity) GetMetadata(ctx context.Context) *models.Group {
	return e.meta
}

func (e *GroupEntity) GetGroupUsers(ctx context.Context, opts ListIAMResourceOptions) ([]*models.User,
	string, error) {
	availableUsers := make([]string, 0)
	refMap, err := e.bdb.GetByPrefix(ctx, dbstore.GetGroupAttachedUserDBKeyWithNS(e.meta.Name, ""))
	if err != nil {
		e.log.WithError(err).Error("Error while reading attached group users from db")
		return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	for _, v := range refMap {
		ref, err := models.DecodeResourceRef(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling attached group users from db")
			return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
		}

		availableUsers = append(availableUsers, ref.Reference)
	}

	numItems := len(availableUsers)
	if numItems == 0 {
		return []*models.User{}, "", nil
	}

	if opts.Marker != "" {
		decodedMarker, err := secure.DecodeSignedMarker(opts.Marker)
		if err != nil {
			e.log.WithError(err).Error("Error while decoding marker")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}

		markerIndex, found := utils.SearchSorted(availableUsers, decodedMarker)
		if !found {
			e.log.WithField("marker", decodedMarker).Error("Marker not found in group users list")
			return nil, "", s3common.GetInvalidArgumentS3Error("", "Invalid Marker")
		}

		// Move to next item after marker
		markerIndex++
		if markerIndex >= numItems {
			return []*models.User{}, "", nil
		}

		availableUsers = availableUsers[markerIndex:]
		numItems = len(availableUsers)
	}

	nextCursor := ""
	if opts.MaxItems > 0 && numItems > opts.MaxItems {
		numItems = opts.MaxItems
		nextCursor = availableUsers[numItems-1]
	}

	dbKeys := make([]string, 0)
	for _, username := range availableUsers[:numItems] {
		dbKeys = append(dbKeys, dbstore.GetUserDBKeyWithNS(username))
	}

	usersMap, err := e.bdb.BulkGet(ctx, dbKeys)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user details from db")
		return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	users := []*models.User{}
	for _, b := range usersMap {
		user, err := models.DecodeUser(b)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling user from db")
			return nil, "", s3common.GetInternalErrorS3Error(e.meta.Name)
		}
		users = append(users, user)
	}

	return users, secure.EncodeSignedMarker(nextCursor), nil
}

func (e *GroupEntity) UpdateGroupPath(ctx context.Context, newPath string) error {
	err := validatePath(newPath)
	if err != nil {
		return err
	}

	delta := &models.Group{
		Path: newPath,
		Arn:  s3common.GenerateGroupARN(models.ConcatPathForArn(newPath, e.meta.Name)),
	}

	updated, err := e.MergeDelta(ctx, delta)
	if err != nil {
		return err
	}

	err = e.bdb.Set(ctx, e.dbKey, updated)
	if err != nil {
		e.log.WithError(err).Error("Error while updating group path in db")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Info("Group path updated.")
	return nil
}

func (e *GroupEntity) AddUserToGroup(ctx context.Context, userName string) error {
	exist := NewUserService(ctx).IsExist(ctx, userName)
	if !exist {
		return s3common.GetNoSuchEntityS3Error(e.meta.Name, "User does not exist.")
	}

	userRef := models.NewResourceRef(userName)
	ub, err := userRef.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding user reference")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	groupRef := models.NewResourceRef(e.meta.Name)
	gb, err := groupRef.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding group reference")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		userRefDbKey := dbstore.GetGroupAttachedUserDBKeyWithNS(e.meta.Name, userName)
		groupRefDbKey := dbstore.GetUserAttachedGroupDBKeyWithNS(userName, e.meta.Name)

		err := txn.Set(ctx, groupRefDbKey, gb)
		if err != nil {
			return err
		}
		return txn.Set(ctx, userRefDbKey, ub)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while adding user %s to group", userName)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, userName)
	e.log.Infof("User %s added to group.", userName)
	return nil
}

func (e *GroupEntity) RemoveUserFromGroup(ctx context.Context, userName string) error {
	err := e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		userRefDbKey := dbstore.GetGroupAttachedUserDBKeyWithNS(e.meta.Name, userName)
		groupRefDbKey := dbstore.GetUserAttachedGroupDBKeyWithNS(userName, e.meta.Name)

		err := txn.Delete(ctx, groupRefDbKey)
		if err != nil {
			return err
		}
		return txn.Delete(ctx, userRefDbKey)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while removing user %s from group", userName)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	GetAuthzEngine().Reset(ctx, userName)
	return nil
}

func (e *GroupEntity) AttachPolicyToGroup(ctx context.Context, policyArn string) error {
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
		currentPoliciesCount, err := e.getGroupPoliciesCount(ctx)
		if err != nil {
			return err
		}
		if currentPoliciesCount >= s3common.MAX_ALLOWED_POLICIES_PER_ENTITY {
			return s3common.GetLimitExceededS3Error(e.meta.Name, "Maximum number of policies attached to group exceeded.")
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

	groupRef := models.NewResourceRef(e.meta.Name)
	gb, err := groupRef.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding group reference")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		policyRefDbKey := dbstore.GetGroupAttachedPolicyDBKeyWithNS(e.meta.Name, policypath)
		groupRefDbKey := dbstore.GetPolicyAttachedGroupDBKeyWithNS(policypath, e.meta.Name)

		err := txn.Set(ctx, policyRefDbKey, pb)
		if err != nil {
			return err
		}
		return txn.Set(ctx, groupRefDbKey, gb)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while attaching policy %s to group", policyArn)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	go func() {
		_, _ = e.resetAuthzEngineForGroupUsers(context.WithoutCancel(ctx))
	}()

	return nil
}

func (e *GroupEntity) DetachPolicyFromGroup(ctx context.Context, policyArn string) error {
	policypath, err := getPolicyPathFromARN(policyArn)
	if err != nil {
		return s3common.GetInvalidArgumentS3Error(e.meta.Name, "Invalid Policy ARN.")
	}

	err = e.bdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		policyRefDbKey := dbstore.GetGroupAttachedPolicyDBKeyWithNS(e.meta.Name, policypath)
		groupRefDbKey := dbstore.GetPolicyAttachedGroupDBKeyWithNS(policypath, e.meta.Name)

		err := txn.Delete(ctx, policyRefDbKey)
		if err != nil {
			return err
		}
		return txn.Delete(ctx, groupRefDbKey)
	})
	if err != nil {
		e.log.WithError(err).Errorf("Error while detaching policy %s from group", policyArn)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	go func() {
		_, _ = e.resetAuthzEngineForGroupUsers(context.WithoutCancel(ctx))
	}()

	return nil
}

func (e *GroupEntity) resetAuthzEngineForGroupUsers(ctx context.Context) (map[string]struct{}, error) {
	users, _, err := e.GetGroupUsers(ctx, ListIAMResourceOptions{})
	if err != nil {
		e.log.WithError(err).Warning("Error while getting group users to reset authz engine")
		return nil, err
	}

	usersMap := make(map[string]struct{}, 0)
	authzEngine := GetAuthzEngine()
	for _, user := range users {
		authzEngine.Reset(ctx, user.Name)
		usersMap[user.Name] = struct{}{}
	}

	return usersMap, nil
}

func (e *GroupEntity) ListAttachedGroupPolicies(ctx context.Context, opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	refResult, err := e.bdb.GetByPrefix(ctx, dbstore.GetGroupAttachedPolicyDBKeyWithNS(e.meta.Name, ""))
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
		e.log.WithError(err).Error("Error while reading attached group policies from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	for _, v := range result {
		policy, err := models.DecodeIAMPolicy(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling attched group policy from db")
			return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
		}
		availablePolicies = append(availablePolicies, policy)
	}

	return availablePolicies, nil
}

func (e *GroupEntity) listGroupInlinePolicies(ctx context.Context,
	opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	result, err := e.bdb.GetByPrefix(ctx, dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, ""))
	if err != nil {
		e.log.WithError(err).Error("Error while unmarshalling inline group policy from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	availablePolicies := []*models.IAMPolicy{}
	for _, v := range result {
		policy, err := models.DecodeIAMPolicy(v)
		if err != nil {
			e.log.WithError(err).Error("Error while unmarshalling inline group policy from db")
			return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
		}

		if opts.PathPrefix == "" || strings.HasPrefix(policy.Path, opts.PathPrefix) {
			availablePolicies = append(availablePolicies, policy)
		}
	}

	return availablePolicies, nil
}

func (e *GroupEntity) ListGroupPolicies(ctx context.Context, opts ListIAMResourceOptions) ([]*models.IAMPolicy, error) {
	var attachedPolicies, inlinePolicies []*models.IAMPolicy
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		attachedPolicies, err = e.ListAttachedGroupPolicies(ctx, ListIAMResourceOptions{PathPrefix: opts.PathPrefix})
		return err
	})

	g.Go(func() error {
		var err error
		inlinePolicies, err = e.listGroupInlinePolicies(ctx, ListIAMResourceOptions{PathPrefix: opts.PathPrefix})
		return err
	})

	if err := g.Wait(); err != nil {
		e.log.WithError(err).Error("Error while listing user policies")
		return nil, err
	}

	return append(attachedPolicies, inlinePolicies...), nil
}

func (e *GroupEntity) getGroupPoliciesCount(ctx context.Context) (int64, error) {
	var currentPoliciesCount int64 = 0
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		prefix := dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, "")
		atomic.AddInt64(&currentPoliciesCount, int64(e.bdb.Count(ctx, prefix)))
		return nil
	})

	g.Go(func() error {
		prefix := dbstore.GetGroupAttachedPolicyDBKeyWithNS(e.meta.Name, "")
		atomic.AddInt64(&currentPoliciesCount, int64(e.bdb.Count(ctx, prefix)))
		return nil
	})

	if err := g.Wait(); err != nil {
		e.log.WithError(err).Error("Error while counting group policies")
		return 0, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return currentPoliciesCount, nil
}

// Group Inline Policy
func (e *GroupEntity) AddGroupPolicy(ctx context.Context, req CreatePolicyRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	currentPoliciesCount, err := e.getGroupPoliciesCount(ctx)
	if err != nil {
		return err
	}

	if currentPoliciesCount >= s3common.MAX_ALLOWED_POLICIES_PER_ENTITY {
		return s3common.GetLimitExceededS3Error(e.meta.Name, "Maximum number of policies attached to group exceeded.")
	}

	policy := models.NewIAMPolicy(req.PolicyName, req.Description, req.PolicyDocument, req.Path, nil)
	policy.DefaultVersionId = "v1"

	b, err := policy.Encode()
	if err != nil {
		e.log.WithError(err).Error("Error while encoding inline policy")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	policyDBKey := dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, req.PolicyName)
	err = e.bdb.Set(ctx, policyDBKey, b)
	if err != nil {
		e.log.WithError(err).Error("Error while adding inline policy to group")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	go func() {
		_, _ = e.resetAuthzEngineForGroupUsers(context.WithoutCancel(ctx))
	}()

	return nil
}

// Group Inline Policy
func (e *GroupEntity) DeleteGroupPolicy(ctx context.Context, policyName string) error {
	policyDBKey := dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, policyName)
	err := e.bdb.Delete(ctx, policyDBKey)
	if err != nil {
		e.log.WithError(err).Errorf("Error while deleting inline policy %s from group", policyName)
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	go func() {
		_, _ = e.resetAuthzEngineForGroupUsers(context.WithoutCancel(ctx))
	}()

	return nil
}

func (e *GroupEntity) GetGroupPolicy(ctx context.Context, policyName string) (*models.IAMPolicy, error) {
	policyDBKey := dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, policyName)
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

func (e *GroupEntity) DeleteGroup(ctx context.Context) error {
	if e.bdb.HasChild(ctx, dbstore.GetGroupAttachedUserDBKeyWithNS(e.meta.Name, "")) ||
		e.bdb.HasChild(ctx, dbstore.GetGroupAttachedPolicyDBKeyWithNS(e.meta.Name, "")) ||
		e.bdb.HasChild(ctx, dbstore.GetGroupInlinePolicyDBKeyWithNS(e.meta.Name, "")) {
		return s3common.GetDeleteConflictS3Error(e.meta.Name, "Group has one or more attached entities. Detach all entities before deleting it.")
	}

	err := e.bdb.Delete(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while deleting group")
		return s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	e.log.Info("Group deleted.")
	return nil
}

func (e *GroupEntity) MergeDelta(ctx context.Context, delta *models.Group) ([]byte, error) {
	existingVal, ok, err := e.bdb.Get(ctx, e.dbKey)
	if err != nil {
		e.log.WithError(err).Error("Error while reading user group details from db")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}
	if !ok {
		return nil, s3common.GetNoSuchEntityS3Error(e.meta.Name)
	}

	b, err := models.MergeProtoMessages(existingVal, &models.Group{}, delta)
	if err != nil {
		e.log.WithError(err).Error("Error while merging group delta")
		return nil, s3common.GetInternalErrorS3Error(e.meta.Name)
	}

	return b, nil
}
