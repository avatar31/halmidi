package s3nativeapihandlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/core/s3"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/utils"
)

var (
	authzEngine = iam.GetAuthzEngine()
)

func IAMActionsHandler(c *gin.Context) {
	body, err := readActionRequestBody(c.Request)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if body.Version != "" && body.Version != IAM_API_VERSION {
		SendIAMErrorResp(c, s3common.GetInvalidRequestS3Error("", "Invalid Version parameter"), nil)
		return
	}

	switch body.Action {
	case AddUserToGroupAction:
		AddUserToGroupHandler(c, *body)
	case AttachGroupPolicyAction:
		AttachGroupPolicyHandler(c, *body)
	case AttachUserPolicyAction:
		AttachUserPolicyHandler(c, *body)
	case CreateAccessKeyAction:
		CreateAccessKeyHandler(c, *body)
	case CreateGroupAction:
		CreateGroupHandler(c, *body)
	case CreatePolicyAction:
		CreatePolicyHandler(c, *body)
	case CreatePolicyVersionAction:
		CreatePolicyVersionHandler(c, *body)
	case CreateUserAction:
		CreateUserHandler(c, *body)
	case DeleteAccessKeyAction:
		DeleteAccessKeyHandler(c, *body)
	case DeleteGroupAction:
		DeleteGroupHandler(c, *body)
	case DeleteGroupPolicyAction:
		DeleteGroupPolicyHandler(c, *body)
	case DeletePolicyAction:
		DeletePolicyHandler(c, *body)
	case DeletePolicyVersionAction:
		DeletePolicyVersionHandler(c, *body)
	case DeleteUserAction:
		DeleteUserHandler(c, *body)
	case DeleteUserPolicyAction:
		DeleteUserPolicyHandler(c, *body)
	case DetachGroupPolicyAction:
		DetachGroupPolicyHandler(c, *body)
	case DetachUserPolicyAction:
		DetachUserPolicyHandler(c, *body)
	case GetAccessKeyLastUsedAction:
		GetAccessKeyLastUsedHandler(c, *body)
	case GetGroupAction:
		GetGroupHandler(c, *body)
	case GetGroupPolicyAction:
		GetGroupPolicyHandler(c, *body)
	case GetPolicyAction:
		GetPolicyHandler(c, *body)
	case GetPolicyVersionAction:
		GetPolicyVersionHandler(c, *body)
	case GetUserAction:
		GetUserHandler(c, *body)
	case GetUserPolicyAction:
		GetUserPolicyHandler(c, *body)
	case ListAccessKeysAction:
		ListAccessKeysHandler(c, *body)
	case ListAttachedGroupPoliciesAction:
		ListAttachedGroupPoliciesHandler(c, *body)
	case ListAttachedUserPoliciesAction:
		ListAttachedUserPoliciesHandler(c, *body)
	case ListEntitiesForPolicyAction:
		ListEntitiesForPolicyHandler(c, *body)
	case ListGroupPoliciesAction:
		ListGroupPoliciesHandler(c, *body)
	case ListGroupsAction:
		ListGroupsHandler(c, *body)
	case ListGroupsForUserAction:
		ListGroupsForUserHandler(c, *body)
	case ListPoliciesAction:
		ListPoliciesHandler(c, *body)
	case ListPolicyTagsAction:
		ListPolicyTagsHandler(c, *body)
	case ListPolicyVersionsAction:
		ListPolicyVersionsHandler(c, *body)
	case ListUserPoliciesAction:
		ListUserPoliciesHandler(c, *body)
	case ListUsersAction:
		ListUsersHandler(c, *body)
	case ListUserTagsAction:
		ListUserTagsHandler(c, *body)
	case PutGroupPolicyAction:
		PutGroupPolicyHandler(c, *body)
	case PutUserPolicyAction:
		PutUserPolicyHandler(c, *body)
	case RemoveUserFromGroupAction:
		RemoveUserFromGroupHandler(c, *body)
	case SetDefaultPolicyVersionAction:
		SetDefaultPolicyVersionHandler(c, *body)
	case TagPolicyAction:
		TagPolicyHandler(c, *body)
	case TagUserAction:
		TagUserHandler(c, *body)
	case UntagPolicyAction:
		UntagPolicyHandler(c, *body)
	case UntagUserAction:
		UntagUserHandler(c, *body)
	case UpdateAccessKeyAction:
		UpdateAccessKeyHandler(c, *body)
	case UpdateGroupAction:
		UpdateGroupHandler(c, *body)
	case UpdateUserAction:
		UpdateUserHandler(c, *body)
	default:
		SendIAMErrorResp(c, s3common.GetMethodNotAllowedS3Error(body.UserName), nil)
	}
}

func BucketGetHandler(c *gin.Context) {
	if _, ok := c.GetQuery(VersionsParam); ok {
		ListObjectVersionsHandler(c)
		return
	}

	if _, ok := c.GetQuery(VersioningParam); ok {
		GetBucketVersioningHandler(c)
		return
	}

	if _, ok := c.GetQuery(ObjectLockParam); ok {
		GetObjectLockConfigurationHandler(c)
		return
	}

	if _, ok := c.GetQuery(TaggingParam); ok {
		GetBucketTaggingHandler(c)
		return
	}

	if _, ok := c.GetQuery(LifeCycleParam); ok {
		GetBucketLifecycleConfigurationHandler(c)
		return
	}

	if _, ok := c.GetQuery(UploadsParam); ok {
		ListMultipartUploadsHandler(c)
		return
	}

	if _, ok := c.GetQuery(ACLParam); ok {
		GetBucketACLHandler(c)
		return
	}

	if _, ok := c.GetQuery(OwnershipControlsParam); ok {
		GetBucketOwnershipControlsHandler(c)
		return
	}

	ListObjectsV2Handler(c)
}

func BucketPutHandler(c *gin.Context) {
	if _, ok := c.GetQuery(VersioningParam); ok {
		PutBucketVersioningHandler(c)
		return
	}

	if _, ok := c.GetQuery(ObjectLockParam); ok {
		PutObjectLockConfigurationHandler(c)
		return
	}

	if _, ok := c.GetQuery(TaggingParam); ok {
		PutBucketTaggingHandler(c)
		return
	}

	if _, ok := c.GetQuery(LifeCycleParam); ok {
		PutBucketLifecycleConfigurationHandler(c)
		return
	}

	if _, ok := c.GetQuery(ACLParam); ok {
		PutBucketACLHandler(c)
		return
	}

	if _, ok := c.GetQuery(OwnershipControlsParam); ok {
		PutBucketOwnershipControlsHandler(c)
		return
	}

	CreateBucketHandler(c)
}

func BucketDeleteHandler(c *gin.Context) {
	if _, ok := c.GetQuery(TaggingParam); ok {
		DeleteBucketTaggingHandler(c)
		return
	}

	if _, ok := c.GetQuery(LifeCycleParam); ok {
		DeleteBucketLifecycleHandler(c)
		return
	}

	if _, ok := c.GetQuery(OwnershipControlsParam); ok {
		DeleteBucketOwnershipControlsHandler(c)
		return
	}

	DeleteBucketHandler(c)
}

func BucketHeadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3ListBucket, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service := s3.NewBucketService(ctx)
	if !service.IsExist(ctx, bucket) {
		SendS3ErrorResp(c, s3common.GetNoSuchBucketS3Error(bucket), nil)
		return
	}

	c.Writer.Header().Set(AMZ_BUCKET_ARN, arn)
	c.Status(http.StatusOK)
}

func ListBucketsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var buckets []BucketInfo

	err := iam.GetAuthzEngine().EvaluateRequest(*c.Request, s3common.PolicyActionS3ListAllMyBuckets, "")
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	prefix, err := url.QueryUnescape(c.Query(PrefixParam))
	if err != nil {
		SendS3ErrorResp(c, s3common.GetInvalidArgumentS3Error("", "Invalid prefix"), nil)
		return
	}
	maxItems := utils.AtoiDefault(c.Query(MaxBucketsParam), DefaultMaxItems)
	continuationToken := c.Query(ContinuationTokenParam)

	opts := s3.ListBucketOptions{
		Prefix:            prefix,
		MaxItems:          maxItems,
		ContinuationToken: continuationToken,
	}
	service := s3.NewBucketService(ctx)
	bucketsMetaData, nextCursor, err := service.ListBuckets(ctx, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var owner *Owner
	for _, meta := range bucketsMetaData {
		if owner == nil && meta.Owner != nil {
			owner = &Owner{
				ID:          meta.Owner.Id,
				DisplayName: meta.Owner.DisplayName,
			}
		}
		buckets = append(buckets, BucketInfo{
			Name:         meta.Name,
			CreationDate: meta.CreationDate,
		})
	}

	result := ListAllMyBucketsResult{
		Xmlns:             S3_AMZ_XMLNS,
		Owner:             owner,
		Buckets:           Buckets{Bucket: buckets},
		Prefix:            prefix,
		ContinuationToken: nextCursor,
	}

	SendS3SuccessResp(c, http.StatusOK, nil, result)
}

func ObjectGetHandler(c *gin.Context) {
	if _, ok := c.GetQuery(LegalHoldParam); ok {
		GetObjectLegalHoldHandler(c)
		return
	}

	if _, ok := c.GetQuery(TaggingParam); ok {
		GetObjectTaggingHandler(c)
		return
	}

	if _, ok := c.GetQuery(UploadIdParam); ok {
		ListPartsHandler(c)
		return
	}

	if _, ok := c.GetQuery(ACLParam); ok {
		GetObjectACLHandler(c)
		return
	}

	ObjectListOrDownloadHandler(c)
}

func ObjectPostHandler(c *gin.Context) {
	if _, ok := c.GetQuery(UploadsParam); ok {
		CreateMultipartUploadHandler(c)
		return
	}

	if _, ok := c.GetQuery(UploadIdParam); ok {
		CompleteMultiPartUploadHandler(c)
		return
	}

	if _, ok := c.GetQuery(DeleteParam); ok {
		DeleteObjectsHandler(c)
		return
	}

	SendS3ErrorResp(c, s3common.GetMethodNotAllowedS3Error(""), nil)
}

func ObjectPutHandler(c *gin.Context) {
	_, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	if objectKey == "" {
		BucketPutHandler(c)
		return
	}

	if _, ok := c.GetQuery(UploadIdParam); ok {
		UploadPartHandler(c)
		return
	}

	if _, ok := c.GetQuery(LegalHoldParam); ok {
		PutObjectLegalHoldHandler(c)
		return
	}

	if _, ok := c.GetQuery(TaggingParam); ok {
		PutObjectTaggingHandler(c)
		return
	}

	if _, ok := c.GetQuery(ACLParam); ok {
		PutObjectACLHandler(c)
		return
	}

	UploadObjectHandler(c)
}

func ObjectDeleteHandler(c *gin.Context) {
	if _, ok := c.GetQuery(UploadIdParam); ok {
		AbortMultipartUploadHandler(c)
		return
	}

	if _, ok := c.GetQuery(TaggingParam); ok {
		DeleteObjectTaggingHandler(c)
		return
	}

	DeleteObjectHandler(c)
}

func ObjectHeadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	if objectKey == "" {
		BucketHeadHandler(c)
		return
	}

	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetObject, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	// versionID := c.Query(VersionIdParam)

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	// TODO: Do we need to handle version
	obj, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	info := obj.GetMetadata(ctx)
	c.Header(CONTENT_LENGTH_HEADER, fmt.Sprintf("%d", info.Size))
	c.Header(CONTENT_TYPE_HEADER, info.ContentType)
	c.Header(ETAG_HEADER, RFC7232ETag(info.Etag))
	c.Header(LAST_MODIFIED_HEADER, formatLastModified(info.LastModified))

	c.Status(http.StatusOK)
}
