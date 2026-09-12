package s3nativeapihandlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

// Notes:
// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_iam-quotas.html

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateUser.html
func CreateUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMCreateUser, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	service := iam.NewUserService(ctx)

	// TODO: Handle Tags
	req := iam.CreateUserRequest{
		UserName:            body.UserName,
		Path:                body.Path,
		PermissionsBoundary: body.PermissionsBoundary,
		Tags:                body.Tags,
	}
	user, err := service.CreateUser(ctx, req)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := CreateUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		Result:           CreateUserResult{User: convertUserToMember(user)},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateUser.html
func UpdateUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMUpdateUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	user, err := entity.UpdateUser(ctx, body.NewPath)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := UpdateUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		UpdateUserResult: UpdateUserResult{User: convertUserToMember(user)},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteUser.html
func DeleteUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeleteUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeleteUser(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeleteUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUser.html
func GetUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := GetUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		Result:           GetUserResult{User: convertUserToMember(entity.GetMetadata(ctx))},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUsers.html
func ListUsersHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListUsers, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		PathPrefix: body.PathPrefix,
		Marker:     body.Marker,
		MaxItems:   body.GetMaxItems(),
	}
	users, nextCursor, err := iam.NewUserService(ctx).ListUsers(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []member{}
	for i := range users {
		members = append(members, convertUserToMember(users[i]))
	}

	response := ListUsersResponse{
		Xmlns: IAM_AMZ_XMLNS,
		Result: ListUsersResult{
			Users:       Users{Member: members},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachUserPolicy.html
func AttachUserPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMAttachUserPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.AttachPolicyToUser(ctx, body.PolicyArn); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := AttachUserPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachUserPolicy.html
func DetachUserPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDetachUserPolicy, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DetachPolicyFromUser(ctx, body.PolicyArn); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DetachUserPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedUserPolicies.html
func ListAttachedUserPoliciesHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListAttachedUserPolicies, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:     body.Marker,
		MaxItems:   body.GetMaxItems(),
		PathPrefix: body.PathPrefix,
	}
	policies, err := entity.ListAttachedUserPolicies(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []AttachedPolicy{}
	for i := range policies {
		result = append(result, AttachedPolicy{PolicyName: policies[i].Name, PolicyArn: policies[i].Arn})
	}

	response := ListAttachedUserPoliciesResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListAttachedUserPoliciesResult: ListAttachedUserPoliciesResult{
			AttachedPolicies: AttachedPolicies{Member: result},
			IsTruncated:      false,
			Marker:           "",
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutUserPolicy.html
func PutUserPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMPutUserPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	req := iam.CreatePolicyRequest{
		PolicyName:     body.PolicyName,
		PolicyDocument: body.PolicyDocument,
	}
	if err := entity.AddUserPolicy(ctx, req); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := PutUserPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUserPolicy.html
func GetUserPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetUserPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policy, err := entity.GetUserPolicy(ctx, body.PolicyName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := GetUserPolicyResponse{
		Xmlns: IAM_AMZ_XMLNS,
		GetUserPolicyResult: GetUserPolicyResult{
			PolicyName:     policy.Name,
			UserName:       body.UserName,
			PolicyDocument: policy.DefaultPolicyDocument,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteUserPolicy.html
func DeleteUserPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeleteUserPolicy, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeleteUserPolicy(ctx, body.PolicyName); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeleteUserPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserPolicies.html
func ListUserPoliciesHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListUserPolicies, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:   body.Marker,
		MaxItems: body.GetMaxItems(),
	}
	policies, err := entity.ListUserPolicies(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []string{}
	for i := range policies {
		result = append(result, policies[i].Name)
	}

	response := ListUserPoliciesResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListUserPoliciesResult: ListUserPoliciesResult{
			PolicyNames: PolicyNames{Member: result},
			IsTruncated: false,
			Marker:      "",
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupsForUser.html
func ListGroupsForUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListGroupsForUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:   body.Marker,
		MaxItems: body.GetMaxItems(),
	}
	groups, nextCursor, err := entity.ListGroupsForUser(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []Group{}
	for i := range groups {
		members = append(members, convertGroupModelToGroupResp(groups[i]))
	}

	response := ListGroupsForUserResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListGroupsForUserResult: ListGroupsForUserResult{
			Groups:      Groups{Member: members},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserTags.html
func ListUserTagsHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListUserTags, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []Tag{}
	tags := entity.GetUserTags(ctx)
	if tags != nil {
		for key, value := range tags.Items {
			members = append(members, Tag{Key: key, Value: value})
		}
	}

	response := ListUserTagsResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListUserTagsResult: ListUserTagsResult{
			Tags:        Tags{Tags: members},
			IsTruncated: false,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagUser.html
func TagUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMTagUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.AddUserTags(ctx, body.Tags); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := TagUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagUser.html
func UntagUserHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewUserService(ctx).GetUserEntity(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMUntagUser, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.RemoveUserTags(ctx, body.TagKeys); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := UntagUserResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateAccessKey.html
func CreateAccessKeyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	service, err := iam.NewUserAccessKeyService(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMCreateAccessKey, service.GetUser().Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	accessKey, err := service.CreateAccessKey(ctx)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := CreateAccessKeyResponse{
		Xmlns: IAM_AMZ_XMLNS,
		Result: CreateAccessKeyResult{AccessKey: UserAccessKey{
			UserName:        accessKey.UserName,
			AccessKeyId:     accessKey.AccessKeyId,
			SecretAccessKey: accessKey.SecretAccessKey,
			Status:          s3common.AccessKeyStatus(accessKey.Status),
			CreateDate:      accessKey.CreateDate,
		}},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteAccessKey.html
func DeleteAccessKeyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	service, err := iam.NewUserAccessKeyService(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeleteAccessKey, service.GetUser().Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := service.GetUserAccessKeyEntity(ctx, body.AccessKeyId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeleteAccessKey(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeleteAccessKeyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccessKeys.html
func ListAccessKeysHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	service, err := iam.NewUserAccessKeyService(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListAccessKeys, service.GetUser().Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	accessKeys, err := service.ListUsersAccessKeys(ctx)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []memberAccessKey{}
	for _, item := range accessKeys {
		result = append(result, memberAccessKey{
			UserName:    item.UserName,
			AccessKeyId: item.AccessKeyId,
			Status:      s3common.AccessKeyStatus(item.Status),
			CreateDate:  item.CreateDate,
		})
	}

	response := ListAccessKeysResponse{
		Xmlns: IAM_AMZ_XMLNS,
		Result: ListAccessKeysResult{
			UserName:          body.UserName,
			AccessKeyMetadata: AccessKeyMetadata{Member: result},
		},
		ResponseMetadata: getResponseMetadata(c),
	}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateAccessKey.html
func UpdateAccessKeyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	service, err := iam.NewUserAccessKeyService(ctx, body.UserName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMUpdateAccessKey, service.GetUser().Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := service.GetUserAccessKeyEntity(ctx, body.AccessKeyId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = entity.UpdateUsersAccessKey(ctx, s3common.AccessKeyStatus(body.Status))
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := UpdateAccessKeyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetAccessKeyLastUsed.html
func GetAccessKeyLastUsedHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewAccessKeyService(ctx).GetUserAccessKeyEntity(ctx, body.AccessKeyId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	userEntity, err := iam.NewUserService(ctx).GetUserEntity(ctx, entity.GetMetadata(ctx).UserName)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Warning("Failed to get user details for access key")
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetAccessKeyLastUsed, userEntity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := GetAccessKeyLastUsedResponse{
		Xmlns:                      IAM_AMZ_XMLNS,
		GetAccessKeyLastUsedResult: GetAccessKeyLastUsedResult{UserName: entity.GetMetadata(ctx).UserName},
		ResponseMetadata:           getResponseMetadata(c),
	}

	lastUsed, err := entity.GetAccessKeyLastUsed(ctx)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if lastUsed != "" {
		service := s3common.DEFAULT_SERVICE
		response.GetAccessKeyLastUsedResult.AccessKeyLastUsed = AccessKeyLastUsed{
			LastUsedDate: &lastUsed,
			ServiceName:  &service,
		}
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateGroup.html
func CreateGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMCreateGroup, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	service := iam.NewGroupService(ctx)
	group, err := service.CreateGroup(ctx, iam.CreateGroupRequest{Name: body.GroupName, Path: body.Path})
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := CreateGroupResponse{
		Xmlns:             IAM_AMZ_XMLNS,
		CreateGroupResult: CreateGroupResult{Group: convertGroupModelToGroupResp(group)},
		ResponseMetadata:  getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroups.html
func ListGroupsHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListGroups, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		PathPrefix: body.PathPrefix,
		Marker:     body.Marker,
		MaxItems:   body.GetMaxItems(),
	}

	groups, nextCursor, err := iam.NewGroupService(ctx).ListGroups(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []Group{}
	for i := range groups {
		members = append(members, convertGroupModelToGroupResp(groups[i]))
	}

	response := ListGroupsResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListGroupsResult: ListGroupsResult{
			Groups:      Groups{Member: members},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroup.html
func GetGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetGroup, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:   body.Marker,
		MaxItems: body.GetMaxItems(),
	}
	users, nextCursor, err := entity.GetGroupUsers(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []member{}
	for i := range users {
		members = append(members, convertUserToMember(users[i]))
	}
	response := GetGroupResponse{
		Xmlns: IAM_AMZ_XMLNS,
		GetGroupResult: GetGroupResult{
			Group:       convertGroupModelToGroupResp(entity.GetMetadata(ctx)),
			Users:       Users{Member: members},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateGroup.html
func UpdateGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMUpdateGroup, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = entity.UpdateGroupPath(ctx, body.NewPath)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := UpdateGroupResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AddUserToGroup.html
func AddUserToGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMAddUserToGroup, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.AddUserToGroup(ctx, body.UserName); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := AddUserToGroupResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_RemoveUserFromGroup.html
func RemoveUserFromGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMRemoveUserFromGroup, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.RemoveUserFromGroup(ctx, body.UserName); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := RemoveUserFromGroupResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachGroupPolicy.html
func AttachGroupPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMAttachGroupPolicy, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.AttachPolicyToGroup(ctx, body.PolicyArn); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := AttachGroupPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachGroupPolicy.html
func DetachGroupPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDetachGroupPolicy, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DetachPolicyFromGroup(ctx, body.PolicyArn); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DetachGroupPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedGroupPolicies.html
func ListAttachedGroupPoliciesHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	arn := entity.GetMetadata(ctx).Arn
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListAttachedGroupPolicies, arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:     body.Marker,
		MaxItems:   body.GetMaxItems(),
		PathPrefix: body.PathPrefix,
	}
	policies, err := entity.ListAttachedGroupPolicies(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []AttachedPolicy{}
	for i := range policies {
		result = append(result, AttachedPolicy{PolicyName: policies[i].Name, PolicyArn: policies[i].Arn})
	}

	response := ListAttachedGroupPoliciesResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListAttachedGroupPoliciesResult: ListAttachedGroupPoliciesResult{
			AttachedPolicies: AttachedPolicies{Member: result},
			IsTruncated:      false,
			Marker:           "",
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupPolicies.html
func ListGroupPoliciesHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListGroupPolicies, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListIAMResourceOptions{
		Marker:   body.Marker,
		MaxItems: body.GetMaxItems(),
	}
	policies, err := entity.ListGroupPolicies(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []string{}
	for i := range policies {
		result = append(result, policies[i].Name)
	}

	response := ListGroupPoliciesResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListGroupPoliciesResult: ListGroupPoliciesResult{
			PolicyNames: PolicyNames{Member: result},
			IsTruncated: false,
			Marker:      "",
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutGroupPolicy.html
func PutGroupPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMPutGroupPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	req := iam.CreatePolicyRequest{
		PolicyName:     body.PolicyName,
		PolicyDocument: body.PolicyDocument,
	}
	if err := entity.AddGroupPolicy(ctx, req); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := PutGroupPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroupPolicy.html
func DeleteGroupPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeleteGroupPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeleteGroupPolicy(ctx, body.PolicyName); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeleteGroupPolicyResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroupPolicy.html
func GetGroupPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetGroupPolicy, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policy, err := entity.GetGroupPolicy(ctx, body.PolicyName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
	}

	response := GetGroupPolicyResponse{
		Xmlns: IAM_AMZ_XMLNS,
		GetGroupPolicyResult: GetGroupPolicyResult{
			PolicyName:     policy.Name,
			GroupName:      body.GroupName,
			PolicyDocument: policy.DefaultPolicyDocument,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroup.html
func DeleteGroupHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	entity, err := iam.NewGroupService(ctx).GetGroupEntity(ctx, body.GroupName)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeleteGroup, entity.GetMetadata(ctx).Arn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeleteGroup(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeleteGroupResponse{Xmlns: IAM_AMZ_XMLNS, ResponseMetadata: getResponseMetadata(c)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicy.html
func CreatePolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMCreatePolicy, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	service := iam.NewIAMPolicyService(ctx)

	req := iam.CreatePolicyRequest{
		PolicyName:     body.PolicyName,
		PolicyDocument: body.PolicyDocument,
		Description:    body.Description,
		Path:           body.Path,
		Tags:           body.Tags,
	}
	policy, err := service.CreateIAMPolicy(ctx, req)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := CreatePolicyResponse{
		Xmlns:              IAM_AMZ_XMLNS,
		CreatePolicyResult: CreatePolicyResult{Policy: convertPolicyModelToPolicyResp(0, policy)},
		ResponseMetadata:   getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicy.html
func DeletePolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeletePolicy, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.DeletePolicy(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeletePolicyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicy.html
func GetPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetPolicy, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policy := entity.GetMetadata(ctx)
	response := GetPolicyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		GetPolicyResult:  GetPolicyResult{Policy: convertPolicyModelToPolicyResp(0, policy)},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicies.html
func ListPoliciesHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	service := iam.NewIAMPolicyService(ctx)

	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListPolicies, "")
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	opts := iam.ListPoliciesOptions{
		ListIAMResourceOptions: iam.ListIAMResourceOptions{
			PathPrefix: body.PathPrefix,
			Marker:     body.Marker,
			MaxItems:   body.GetMaxItems(),
		},
		PolicyUsageFilter: body.PolicyUsageFilter,
		OnlyAttached:      &body.OnlyAttached,
	}
	policies, nextCursor, err := service.ListIAMPolicies(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []Policy{}
	for i := range policies {
		result = append(result, convertPolicyModelToPolicyResp(0, policies[i]))
	}

	response := ListPoliciesResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListPoliciesResult: ListPoliciesResult{
			Policies:    Policies{Member: result},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListEntitiesForPolicy.html
func ListEntitiesForPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListEntitiesForPolicy, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	users, groups, err := entity.ListEntities(ctx, body.PathPrefix)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	userMembers := []PolicyUser{}
	for i := range users {
		userMembers = append(userMembers, PolicyUser{UserName: users[i].Name, UserId: users[i].Uuid})
	}

	groupMembers := []PolicyGroup{}
	for i := range groups {
		groupMembers = append(groupMembers, PolicyGroup{GroupName: groups[i].Name, GroupId: groups[i].Uuid})
	}

	response := ListEntitiesForPolicyResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListEntitiesForPolicyResult: ListEntitiesForPolicyResult{
			PolicyGroups: PolicyGroups{Member: groupMembers},
			PolicyUsers:  PolicyUsers{Member: userMembers},
			IsTruncated:  false,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicyTags.html
func ListPolicyTagsHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListPolicyTags, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	members := []Tag{}
	tags := entity.GetPolicyTags(ctx)
	if tags != nil {
		for key, value := range tags.Items {
			members = append(members, Tag{Key: key, Value: value})
		}
	}

	response := ListPolicyTagsResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListPolicyTagsResult: ListPolicyTagsResult{
			Tags:        Tags{Tags: members},
			IsTruncated: false,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagPolicy.html
func TagPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMTagPolicy, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.AddPolicyTags(ctx, body.Tags); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := TagPolicyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagPolicy.html
func UntagPolicyHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMUntagPolicy, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := entity.RemovePolicyTags(ctx, body.TagKeys); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := UntagPolicyResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicyVersion.html
func CreatePolicyVersionHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMCreatePolicyVersion, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	service := iam.NewIAMPolicyService(ctx)
	entity, err := service.GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	iamPolicyVersion, err := entity.CreatePolicyVersion(ctx, body.PolicyDocument, body.SetAsDefault)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, _ = iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	updatedPolicy := entity.GetMetadata(ctx)

	response := CreatePolicyVersionResponse{
		Xmlns: IAM_AMZ_XMLNS,
		CreatePolicyVersionResult: CreatePolicyVersionResult{
			PolicyVersion: convertPolicyVerModelToPolicyVerResp(updatedPolicy.DefaultVersionId, iamPolicyVersion),
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicyVersions.html
func ListPolicyVersionsHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMListPolicyVersions, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	entity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	iamPolicy := entity.GetMetadata(ctx)

	opts := iam.ListIAMResourceOptions{
		Marker:   body.Marker,
		MaxItems: body.GetMaxItems(),
	}
	policyVersions, nextCursor, err := iam.NewIAMPolicyVersionService(ctx, iamPolicy).ListIAMPolicyVersions(ctx, opts)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	result := []PolicyVersion{}
	for i := range policyVersions {
		result = append(result, convertPolicyVerModelToPolicyVerResp(iamPolicy.DefaultVersionId, policyVersions[i]))
	}

	response := ListPolicyVersionsResponse{
		Xmlns: IAM_AMZ_XMLNS,
		ListPolicyVersionsResult: ListPolicyVersionsResult{
			Versions:    PolicyVersions{Member: result},
			IsTruncated: nextCursor != "",
			Marker:      nextCursor,
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_SetDefaultPolicyVersion.html
func SetDefaultPolicyVersionHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMSetDefaultPolicyVersion, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policyEntity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	versionEntity, err := iam.NewIAMPolicyVersionService(ctx, policyEntity.GetMetadata(ctx)).
		GetIAMPolicyVersionEntity(ctx, body.VersionId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := versionEntity.SetDefaultIAMPolicyVersion(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := SetDefaultPolicyVersionResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicyVersion.html
func GetPolicyVersionHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMGetPolicyVersion, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policyEntity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	iamPolicy := policyEntity.GetMetadata(ctx)
	versionEntity, err := iam.NewIAMPolicyVersionService(ctx, iamPolicy).
		GetIAMPolicyVersionEntity(ctx, body.VersionId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	meta := versionEntity.GetMetadata(ctx)
	response := GetPolicyVersionResponse{
		Xmlns: IAM_AMZ_XMLNS,
		GetPolicyVersionResult: GetPolicyVersionResult{
			PolicyVersion: convertPolicyVerModelToPolicyVerResp(iamPolicy.DefaultVersionId, meta),
		},
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicyVersion.html
func DeletePolicyVersionHandler(c *gin.Context, body IAMActionRequest) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionIAMDeletePolicyVersion, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	policyEntity, err := iam.NewIAMPolicyService(ctx).GetIAMPolicyEntity(ctx, body.PolicyArn)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	versionEntity, err := iam.NewIAMPolicyVersionService(ctx, policyEntity.GetMetadata(ctx)).
		GetIAMPolicyVersionEntity(ctx, body.VersionId)
	if err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	if err := versionEntity.DeleteIAMPolicyVersion(ctx); err != nil {
		SendIAMErrorResp(c, err, nil)
		return
	}

	response := DeletePolicyVersionResponse{
		Xmlns:            IAM_AMZ_XMLNS,
		ResponseMetadata: getResponseMetadata(c),
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// TODO: P0: Support these parameters https://docs.aws.amazon.com/IAM/latest/APIReference/CommonParameters.html
func readActionRequestBody(r *http.Request) (*IAMActionRequest, error) {
	var req IAMActionRequest
	if r.ContentLength > 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.GetLogger(r.Context()).WithError(err).Error("Error while parsing request body")
			return nil, s3common.GetMalformedPOSTRequestS3Error("")
		}

		tags := map[string]*Tag{}
		strBody := string(body)
		splitBody := strings.SplitSeq(strBody, "&")

		for item := range splitBody {
			if kvPair := strings.Split(item, "="); len(kvPair) == 2 {
				switch kvPair[0] {
				case "AccessKeyId":
					if len(kvPair[1]) == 0 || len(kvPair[1]) > s3common.MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH {
						return nil, s3common.GetInvalidArgumentS3Error("", "AccessKeyId length is invalid")
					}
					req.AccessKeyId = kvPair[1]
				case "Action":
					req.Action = kvPair[1]
				case "Description":
					req.Description = kvPair[1]
				case "GroupName":
					if len(kvPair[1]) == 0 || len(kvPair[1]) > s3common.MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH {
						return nil, s3common.GetInvalidArgumentS3Error("", "GroupName length is invalid")
					}
					req.GroupName = kvPair[1]
				case "Marker":
					req.Marker = kvPair[1]
				case "MaxItems":
					max := utils.AtoiDefault(kvPair[1], 0)
					if max > 0 {
						req.MaxItems = &max
					}
				case "NewPath":
					decoded, err := url.QueryUnescape(kvPair[1])
					if err != nil {
						return nil, s3common.GetMalformedPOSTRequestS3Error("")
					}
					req.NewPath = decoded
				case "OnlyAttached":
					req.OnlyAttached = kvPair[1] == "true"
				case "Path":
					decoded, err := url.QueryUnescape(kvPair[1])
					if err != nil {
						return nil, s3common.GetMalformedPOSTRequestS3Error("")
					}
					req.Path = decoded
				case "PathPrefix":
					decoded, err := url.QueryUnescape(kvPair[1])
					if err != nil {
						return nil, s3common.GetMalformedPOSTRequestS3Error("")
					}
					req.PathPrefix = decoded
				case "PermissionsBoundary":
					req.PermissionsBoundary = kvPair[1]
				case "PolicyArn":
					if len(kvPair[1]) == 0 || len(kvPair[1]) > s3common.MAX_ALLOWED_ARN_LENGTH {
						return nil, s3common.GetInvalidArgumentS3Error("", "PolicyArn length is invalid")
					}
					decoded, err := url.QueryUnescape(kvPair[1])
					if err != nil {
						return nil, s3common.GetMalformedPOSTRequestS3Error("")
					}
					req.PolicyArn = decoded
				case "PolicyDocument":
					decoded, err := url.QueryUnescape(kvPair[1])
					if err != nil {
						return nil, s3common.GetMalformedPOSTRequestS3Error("")
					}
					req.PolicyDocument = decoded
				case "PolicyName":
					if len(kvPair[1]) == 0 || len(kvPair[1]) > s3common.MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH {
						return nil, s3common.GetInvalidArgumentS3Error("", "PolicyName length is invalid")
					}
					req.PolicyName = kvPair[1]
				case "PolicyUsageFilter":
					req.PolicyUsageFilter = kvPair[1]
				case "SetAsDefault":
					req.SetAsDefault = kvPair[1] == "true"
				case "Status":
					req.Status = kvPair[1]
				case "UserName":
					if len(kvPair[1]) == 0 || len(kvPair[1]) > s3common.MAX_ALLOWED_USER_NAME_LENGTH {
						return nil, s3common.GetInvalidArgumentS3Error("", "UserName length is invalid")
					}
					req.UserName = kvPair[1]
				case "Version":
					req.Version = kvPair[1]
				case "VersionId":
					req.VersionId = kvPair[1]
				case "NewUserName", "NewGroupName":
					return nil, s3common.GetInvalidArgumentS3Error("", fmt.Sprintf("unsupported parameter %s", kvPair[0]))
				default:
					if strings.HasPrefix(kvPair[0], "Tags.member.") {
						tagParts := strings.SplitN(kvPair[0], ".", 4)
						if len(tagParts) == 4 {
							index := tagParts[2]
							field := tagParts[3]

							// Set the appropriate field
							switch field {
							case "Key":
								if tags[index] == nil {
									tags[index] = &Tag{}
								}
								tags[index].Key = kvPair[1]
							case "Value":
								if tags[index] == nil {
									tags[index] = &Tag{}
								}
								tags[index].Value = kvPair[1]
							}
						}
					}

					if strings.HasPrefix(kvPair[0], "TagKeys.member.") {
						tagParts := strings.SplitN(kvPair[0], ".", 3)
						if len(tagParts) == 3 {
							req.TagKeys = append(req.TagKeys, kvPair[1])
						}
					}
				}
			}
		}

		if len(tags) > 0 {
			req.Tags = &models.TagMap{Items: map[string]string{}}
			for _, tag := range tags {
				if tag != nil {
					req.Tags.Items[tag.Key] = tag.Value
				}
			}
		}
	}

	return &req, nil
}

func convertUserToMember(user *models.User) member {
	return member{
		Path:             user.Path,
		UserName:         user.Name,
		UserId:           user.Uuid,
		Arn:              user.Arn,
		CreateDate:       user.CreateDate,
		PasswordLastUsed: user.PasswordLastUsed,
	}
}

func convertGroupModelToGroupResp(group *models.Group) Group {
	return Group{
		Path:       group.Path,
		GroupName:  group.Name,
		GroupId:    group.Uuid,
		Arn:        group.Arn,
		CreateDate: group.CreateDate,
	}
}

func convertPolicyModelToPolicyResp(count int32, policy *models.IAMPolicy) Policy {
	return Policy{
		Arn:              policy.Arn,
		AttachmentCount:  int32(count),
		CreateDate:       policy.CreateDate,
		DefaultVersionId: policy.DefaultVersionId,
		IsAttachable:     policy.IsAttachable,
		Path:             policy.Path,
		PolicyId:         policy.Uuid,
		PolicyName:       policy.Name,
		UpdateDate:       policy.UpdateDate,
	}
}

func convertPolicyVerModelToPolicyVerResp(defaultVersionId string, policy *models.IAMPolicyVersion) PolicyVersion {
	return PolicyVersion{
		IsDefaultVersion: defaultVersionId == policy.VersionId,
		CreateDate:       policy.CreateDate,
		VersionId:        policy.VersionId,
		Document:         policy.Document,
	}
}
