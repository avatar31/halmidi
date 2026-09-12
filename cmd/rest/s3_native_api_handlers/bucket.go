package s3nativeapihandlers

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/iam"
	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html
func ListObjectsV2Handler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket := c.Param("bucket")

	if c.Query(ListTypeParam) != "2" {
		logger.GetLogger(ctx).Warning("Only ListObjectsV2 is supported")
		return
	}

	prefix, err := getPrefixParam(c)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	arn := s3common.GenerateS3ObjectARN(bucket, prefix)
	if before, ok := strings.CutSuffix(arn, "*"); ok {
		arn = before
	}

	err = iam.GetAuthzEngine().EvaluateRequest(*c.Request, s3common.PolicyActionS3ListBucket, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	delimiter := c.Query(DelimiterParam)
	encodingType := c.Query(EncodingTypeParam)
	maxKeys := utils.AtoiDefault(c.Query(MaxKeysParam), DefaultMaxItems)
	startAfter := c.Query(StartAfterParam)
	continuationToken := c.Query(ContinuationTokenParam)
	if startAfter == "" && continuationToken != "" {
		startAfter = continuationToken
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	opts := &s3.ListObjectOptions{
		Prefix:     prefix,
		Delimiter:  delimiter,
		StartAfter: startAfter,
		MaxKeys:    maxKeys,
	}
	objects, commonPrefixes, nextCursor, err := service.ListObjects(ctx, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := formatObjectListResp(bucket, nextCursor, maxKeys, objects)
	response.EncodingType = encodingType
	response.Prefix = prefix
	response.Delimiter = delimiter
	response.StartAfter = c.Query(StartAfterParam)
	response.ContinuationToken = continuationToken
	if nextCursor != "" {
		response.NextContinuationToken = nextCursor
	}
	if len(commonPrefixes) > 0 {
		var encodedCommonPrefixes []CommonPrefix
		for _, cp := range commonPrefixes {
			encodedCommonPrefixes = append(encodedCommonPrefixes, CommonPrefix{Prefix: cp})
		}
		response.CommonPrefixes = encodedCommonPrefixes
		response.KeyCount += len(encodedCommonPrefixes)
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func ListObjectWithPrefixHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, prefix := getBucketNameAndObjectKeyFromHttpReq(c)

	if c.Query(ListTypeParam) != "2" {
		logger.GetLogger(ctx).Warning("Only ListObjectsV2 is supported")
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	objectsMetaData, err := service.ListObjectsWithPrefix(ctx, prefix)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := formatObjectListResp(bucket, "", DefaultMaxItems, objectsMetaData)
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func formatObjectListResp(bucket, nextCursor string, maxKeys int, objList []*models.ObjectMeta) ListBucketResult {
	var s3RespObjects []S3Object

	for _, object := range objList {
		var owner *Owner
		if object.Owner != nil {
			owner = &Owner{
				ID:          object.Owner.Id,
				DisplayName: object.Owner.DisplayName,
			}
		}

		s3RespObjects = append(s3RespObjects, S3Object{
			Key:          object.Key,
			ETag:         object.Etag,
			LastModified: object.LastModified,
			Size:         object.Size,
			StorageClass: s3common.S3_STANDARD_STORAGE_CLASS,
			Owner:        owner,
		})
	}

	return ListBucketResult{
		Xmlns:       S3_AMZ_XMLNS,
		Name:        bucket,
		KeyCount:    len(s3RespObjects),
		MaxKeys:     maxKeys,
		IsTruncated: nextCursor != "",
		Contents:    s3RespObjects,
	}
}

func GetObjectLockConfigurationHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetBucketObjectLockConfiguration, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var response *ObjectLockConfiguration
	if objLockingInfo := bucket.GetObjectLocking(ctx); objLockingInfo != nil {
		response = &ObjectLockConfiguration{
			Xmlns:             S3_AMZ_XMLNS,
			ObjectLockEnabled: s3common.ObjectLockingStatus(objLockingInfo.ObjectLockEnabled),
		}

		if objLockingInfo.Rule != nil {
			response.Rule = &LockRuleConfiguration{
				DefaultRetention: DefaultRetentionConfiguration{
					Mode:  s3common.ObjectLockingMode(objLockingInfo.Rule.DefaultRetention.Mode),
					Days:  objLockingInfo.Rule.DefaultRetention.Days,
					Years: objLockingInfo.Rule.DefaultRetention.Years,
				},
			}
		}
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func ListObjectVersionsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	prefix, err := getPrefixParam(c)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	arn := s3common.GenerateS3ObjectARN(bucketName, prefix)
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3ListBucketVersions, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	versions, deleteMarkers, err := bucket.ListObjectVersions(ctx, prefix)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := ListVersionsResult{
		Xmlns:         S3_AMZ_XMLNS,
		Name:          bucketName,
		IsTruncated:   false,
		Prefix:        prefix,
		Versions:      []Version{},
		DeleteMarkers: []DeleteMarker{},
	}

	for _, v := range deleteMarkers {
		response.DeleteMarkers = append(response.DeleteMarkers, DeleteMarker{
			Key:          v.Key,
			VersionId:    v.Id,
			IsLatest:     v.IsLatest,
			LastModified: v.LastModified,
		})
	}

	for _, v := range versions {
		var id *string
		if v.Id != s3common.NO_VERSION {
			id = &v.Id
		}

		response.Versions = append(response.Versions, Version{
			Key:          v.Key,
			VersionId:    id,
			IsLatest:     v.IsLatest,
			LastModified: v.LastModified,
			ETag:         v.Etag,
			Size:         v.Size,
			StorageClass: s3common.S3_STANDARD_STORAGE_CLASS,
		})
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func GetBucketVersioningHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetBucketVersioning, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var response *VersioningConfiguration
	if info := bucket.GetVersioningConfig(ctx); info != nil {
		response = &VersioningConfiguration{Xmlns: S3_AMZ_XMLNS, Status: s3common.VersioningStatus(info.Status)}
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func GetBucketTaggingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetBucketTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	tags, err := bucket.GetTags(ctx)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := Tagging{Xmlns: S3_AMZ_XMLNS, TagSet: convertTagMapToS3TagSet(tags)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func PutBucketTaggingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutBucketTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config Tagging
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucketName), nil)
		return
	}

	tags, err := convertS3TagSetToTagMap(bucketName, config.TagSet)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.UpdateTags(ctx, tags); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func DeleteBucketTaggingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteBucketTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.UpdateTags(ctx, &models.TagMap{}); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

func CreateBucketHandler(c *gin.Context) {
	ctx := c.Request.Context()
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3CreateBucket, "")
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket := c.Param("bucket")
	lockEnabled := c.GetHeader(AMZ_BUCKET_OBJECT_LOCK_ENABLED_HEADER) == "true"

	var config CreateBucketConfiguration
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucket), nil)
		return
	}

	tags := &models.TagMap{}
	for _, item := range config.Tags {
		tags.Items[item.Key] = item.Value
	}
	createReq := s3.CreateBucketRequest{Name: bucket, Tags: tags}
	if lockEnabled {
		createReq.ObjectLocking = models.NewBucketObjectLocking(s3common.ObjectLockingStatusEnabled)
	}

	service := s3.NewBucketService(ctx)
	meta, err := service.CreateBucket(ctx, createReq)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	respHeaders := map[string]string{
		LOCATION_RESP_HEADER: "/" + bucket,
		AMZ_BUCKET_ARN:       meta.Arn,
	}
	SendS3SuccessResp(c, http.StatusOK, respHeaders, nil)
}

func PutObjectLockConfigurationHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObjectLockConfiguration, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config ObjectLockConfiguration
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucketName), nil)
		return
	}

	req := models.NewBucketObjectLocking(config.ObjectLockEnabled)
	req.UpdateModeAndRetention(config.Rule.DefaultRetention.Mode, config.Rule.DefaultRetention.Days,
		config.Rule.DefaultRetention.Years)

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.UpdateObjectLocking(ctx, req); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func PutBucketVersioningHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutBucketVersioning, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config VersioningConfiguration
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucketName), nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	err = bucket.UpdateVersioning(ctx, s3common.VersioningStatus(config.Status))
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func DeleteBucketHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteBucket, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.DeleteBucket(ctx); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

func GetBucketLifecycleConfigurationHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetLifecycleConfiguration, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	lifeCycleConfig, err := bucket.GetLifeCycleConfig(ctx)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := &LifecycleConfig{
		Xmlns: S3_AMZ_XMLNS,
		Rules: []LifeCycleRule{},
	}

	for _, rule := range lifeCycleConfig.Rules {
		var filter *Filter
		if rule.Filter != nil {
			var tag *Tag
			if rule.Filter.Tag != nil {
				tag = &Tag{
					Key:   rule.Filter.Tag.Key,
					Value: rule.Filter.Tag.Value,
				}
			}

			var and *AndFilter
			if rule.Filter.And != nil {
				and = &AndFilter{
					Tags:                  []Tag{},
					ObjectSizeGreaterThan: &rule.Filter.And.ObjectSizeGreaterThan,
					ObjectSizeLessThan:    &rule.Filter.And.ObjectSizeLessThan,
				}
				for _, t := range rule.Filter.And.Tags {
					and.Tags = append(and.Tags, Tag{
						Key:   t.Key,
						Value: t.Value,
					})
				}
			}

			filter = &Filter{
				Prefix:                rule.Filter.Prefix,
				Tag:                   tag,
				ObjectSizeGreaterThan: &rule.Filter.ObjectSizeGreaterThan,
				ObjectSizeLessThan:    &rule.Filter.ObjectSizeLessThan,
				And:                   and,
			}
		}

		var expiration *Expiration
		if rule.Expiration != nil {
			expiration = &Expiration{
				Days:                      &rule.Expiration.Days,
				Date:                      &rule.Expiration.Date,
				ExpiredObjectDeleteMarker: &rule.Expiration.ExpiredObjectDeleteMarker,
			}
		}

		var noncurrentVersionExpiration *NoncurrentVersionExpiration
		if rule.NoncurrentVersionExpiration != nil {
			noncurrentVersionExpiration = &NoncurrentVersionExpiration{
				NewerNoncurrentVersions: rule.NoncurrentVersionExpiration.NewerNoncurrentVersions,
				NoncurrentDays:          rule.NoncurrentVersionExpiration.NoncurrentDays,
			}
		}

		var abortIncompleteMultipartUpload *AbortIncompleteMultipartUpload
		if rule.AbortIncompleteMultipartUpload != nil {
			abortIncompleteMultipartUpload = &AbortIncompleteMultipartUpload{
				DaysAfterInitiation: rule.AbortIncompleteMultipartUpload.DaysAfterInitiation,
			}
		}

		response.Rules = append(response.Rules, LifeCycleRule{
			ID:                             rule.Id,
			Prefix:                         rule.Prefix,
			Status:                         s3common.LifiCycleStatus(rule.Status),
			Filter:                         filter,
			Expiration:                     expiration,
			NoncurrentVersionExpiration:    noncurrentVersionExpiration,
			AbortIncompleteMultipartUpload: abortIncompleteMultipartUpload,
		})
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func PutBucketLifecycleConfigurationHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutLifecycleConfiguration, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config models.LifecycleConfig
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucketName), nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.UpdateLifeCycleConfig(ctx, &config); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func DeleteBucketLifecycleHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteLifecycleConfiguration, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucket, err := s3.NewBucketService(ctx).GetBucketEntity(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := bucket.DeleteLifeCycleConfig(ctx); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAcl.html
func GetBucketACLHandler(c *gin.Context) {
	resp := AccessControlPolicy{
		Xmlns:             S3_AMZ_XMLNS,
		Owner:             &Owner{},
		AccessControlList: AccessControlList{Grants: []Grant{}},
	}

	SendS3SuccessResp(c, http.StatusOK, nil, resp)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketAcl.html
func PutBucketACLHandler(c *gin.Context) {
	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketOwnershipControls.html
func GetBucketOwnershipControlsHandler(c *gin.Context) {
	resp := OwnershipControls{
		Xmlns: S3_AMZ_XMLNS,
		Rules: []OwnershipRule{{
			ObjectOwnership: "BucketOwnerEnforced",
		}},
	}

	SendS3SuccessResp(c, http.StatusOK, nil, resp)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketOwnershipControls.html
func PutBucketOwnershipControlsHandler(c *gin.Context) {
	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketOwnershipControls.html
func DeleteBucketOwnershipControlsHandler(c *gin.Context) {
	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

func readRequestBody(r *http.Request, output any) error {
	log := logger.GetLogger(r.Context())
	if r.ContentLength > 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Errorf("Error while parsing request body")
			return err
		}

		if err := xml.Unmarshal(body, output); err != nil {
			return err
		}
	}

	return nil
}

func convertTagMapToS3TagSet(tagMap *models.TagMap) TagSet {
	var tagSet TagSet
	for k, v := range tagMap.Items {
		tagSet.Tags = append(tagSet.Tags, Tag{
			Key:   k,
			Value: v,
		})
	}
	return tagSet
}

func convertS3TagSetToTagMap(bucketName string, s3TagSet TagSet) (*models.TagMap, error) {
	tags := &models.TagMap{}
	for _, item := range s3TagSet.Tags {
		if _, ok := tags.Items[item.Key]; ok {
			return nil, s3common.GetInvalidTagS3Error(bucketName)
		}
		tags.Items[item.Key] = item.Value
	}
	return tags, nil
}
