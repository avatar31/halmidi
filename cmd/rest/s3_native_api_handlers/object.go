package s3nativeapihandlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

func GetObjectTaggingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetObjectTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	tags, err := object.GetTags(ctx)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := Tagging{Xmlns: S3_AMZ_XMLNS, TagSet: convertTagMapToS3TagSet(tags)}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

func PutObjectTaggingHandler(c *gin.Context) {
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObjectTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config Tagging
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucket), nil)
		return
	}

	tags, err := convertS3TagSetToTagMap(bucket, config.TagSet)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	ctx := c.Request.Context()
	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := object.UpdateTags(ctx, tags); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func DeleteObjectTaggingHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteObjectTagging, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if err := object.UpdateTags(ctx, &models.TagMap{}); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html
func CreateMultipartUploadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObject, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	lockMode := c.GetHeader(AMZ_OBJECT_LOCK_MODE_HEADER)
	retainUntil := c.GetHeader(AMZ_OBJECT_LOCK_RETAIN_UNTIL_HEADER)
	legalHold := c.GetHeader(AMZ_OBJECT_LOCK_LEGAL_HOLD_HEADER)

	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	opts := s3.InitMultipartUploadOptions{
		LockMode:    lockMode,
		RetainUntil: retainUntil,
		LegalHold:   legalHold,
	}
	uploadID, err := service.InitMultipartUpload(ctx, objectKey, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	res := InitiateMultipartUploadResult{
		Xmlns:    S3_AMZ_XMLNS,
		Bucket:   bucket,
		Key:      objectKey,
		UploadId: uploadID,
	}

	SendS3SuccessResp(c, http.StatusOK, nil, res)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListMultipartUploads.html
func ListMultipartUploadsHandler(c *gin.Context) {
	bucketName := c.Param("bucket")
	arn := s3common.GenerateS3BucketARN(bucketName)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3ListBucketMultipartUploads, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	ctx := c.Request.Context()
	service, err := s3.NewMultiPartUploadService(ctx, bucketName)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	uploads, err := service.ListMultipartUploads(ctx)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := ListMultipartUploadsResult{
		Xmlns:   S3_AMZ_XMLNS,
		Bucket:  bucketName,
		Uploads: []MultipartUpload{},
	}

	for _, upload := range uploads {
		response.Uploads = append(response.Uploads, MultipartUpload{
			Key:          upload.Key,
			UploadId:     upload.Uuid,
			Initiated:    upload.Initiated,
			StorageClass: s3common.S3_STANDARD_STORAGE_CLASS,
			Owner: &Owner{
				ID:          upload.Owner.Id,
				DisplayName: upload.Owner.DisplayName,
			},
			Initiator: &Owner{
				ID:          upload.Initiator.Id,
				DisplayName: upload.Initiator.DisplayName,
			},
		})
	}

	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html
func ListPartsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3ListMultipartUploadParts, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	uploadId := c.Query(UploadIdParam)
	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	parts, err := service.ListParts(ctx, uploadId)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	resp := ListPartsResult{
		Xmlns:    S3_AMZ_XMLNS,
		Bucket:   bucket,
		Key:      objectKey,
		UploadId: uploadId,
		Parts:    []PartResult{},
	}

	for _, p := range parts {
		resp.Parts = append(resp.Parts, PartResult{
			Part: Part{
				ETag:       RFC7232ETag(p.Etag),
				PartNumber: p.PartNumber,
			},
			LastModified: p.LastModified,
			Size:         p.Size,
		})
	}

	SendS3SuccessResp(c, http.StatusOK, nil, resp)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPart.html
func UploadPartHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket := c.Param("bucket")
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObject, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	uploadID := c.Query(UploadIdParam)
	part := c.Query(PartNumberParam)
	partNumber, err := strconv.ParseInt(part, 10, 32)
	if err != nil {
		err = s3common.GetInvalidPartS3Error(bucket, fmt.Sprintf("Invalid part '%s'", part))
		SendS3ErrorResp(c, err, nil)
	}

	opts := s3.UploadPartOptions{
		UploadID:   uploadID,
		PartNumber: int32(partNumber),
	}

	if source := c.GetHeader(AMZ_COPY_SOURCE_HEADER); source != "" {
		copySourceBucket, copySourceKey, err := splitCopySourceHeader(bucket, source)
		if err != nil {
			SendS3ErrorResp(c, err, nil)
			return
		}

		opts.CopySourceBucket = copySourceBucket
		opts.CopySourceKey = copySourceKey
		UploadPartCopyHandler(c, bucket, opts)
		return
	}

	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	etag, err := service.UploadPart(ctx, c.Request.Body, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, map[string]string{ETAG_HEADER: RFC7232ETag(etag)}, nil)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPartCopy.html
func UploadPartCopyHandler(c *gin.Context, bucket string, opts s3.UploadPartOptions) {
	ctx := c.Request.Context()
	rangeHeader := c.GetHeader(AMZ_COPY_SOURCE_RANGE_HEADER)
	if rangeHeader == "" {
		SendS3ErrorResp(c, s3common.GetInvalidArgumentS3Error(bucket,
			"Missing required header 'x-amz-copy-source-range' for Upload Part Copy"), nil)
		return
	}
	start, end, err := parseRangeHeader(rangeHeader, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if end == nil {
		SendS3ErrorResp(c, s3common.GetInvalidArgumentS3Error(bucket,
			"Invalid 'x-amz-copy-source-range' header: End range is missing"), nil)
		return
	}

	opts.CopySourceRangeStart = *start
	opts.CopySourceRangeEnd = *end

	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	etag, err := service.UploadPart(ctx, nil, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	resp := CopyPartResult{
		Xmlns:             S3_AMZ_XMLNS,
		ETag:              RFC7232ETag(etag),
		LastModified:      utils.ConvertTimeToString(time.Now()),
		ChecksumCRC32:     "",
		ChecksumCRC32C:    "",
		ChecksumSHA1:      "",
		ChecksumSHA256:    "",
		ChecksumCRC64NVME: "",
	}

	SendS3SuccessResp(c, http.StatusOK, nil, resp)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html
func CompleteMultiPartUploadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObject, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var completeReq CompleteMultipartUpload
	if err := readRequestBody(c.Request, &completeReq); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucket), nil)
		return
	}

	parts := []int32{}
	for _, p := range completeReq.Parts {
		parts = append(parts, p.PartNumber)
	}

	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	uploadId := c.Query(UploadIdParam)
	etag, versionId, err := service.CompleteMultipartUpload(ctx, uploadId, parts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
	}

	res := CompleteMultipartUploadResult{
		Xmlns:    S3_AMZ_XMLNS,
		Location: fmt.Sprintf("s3://%s/%s", bucket, objectKey),
		Bucket:   bucket,
		Key:      objectKey,
		ETag:     RFC7232ETag(etag),
	}

	if versionId != "" {
		c.Header(AMZ_VERSION_ID_HEADER, versionId)
	}

	SendS3SuccessResp(c, http.StatusOK, nil, res)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html
func AbortMultipartUploadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket := c.Param("bucket")
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3AbortMultipartUpload, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	uploadID := c.Query("uploadId")
	service, err := s3.NewMultiPartUploadService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	err = service.AbortMultipartUpload(ctx, uploadID)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

func UploadObjectHandler(c *gin.Context) {
	if c.GetHeader(AMZ_COPY_SOURCE_HEADER) != "" {
		CopyObjectHandler(c)
		return
	}

	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObject, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	// Parse headers for lock
	lockMode := c.GetHeader(AMZ_OBJECT_LOCK_MODE_HEADER)
	retainUntil := c.GetHeader(AMZ_OBJECT_LOCK_RETAIN_UNTIL_HEADER)
	contentType := c.GetHeader(CONTENT_TYPE_HEADER)
	contentLength, err := strconv.ParseInt(c.GetHeader(CONTENT_LENGTH_HEADER), 10, 64)
	if err != nil {
		SendS3ErrorResp(c, s3common.GetInvalidArgumentS3Error(bucket, "Invalid Content-Length"), nil)
		return
	}

	ctx := c.Request.Context()
	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	opts := s3.PutObjectOptions{
		LockMode:      lockMode,
		RetainUntil:   retainUntil,
		ContentType:   contentType,
		ContentLength: contentLength,
	}
	etag, versionId, err := service.PutObject(ctx, objectKey, c.Request.Body, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if versionId != "" {
		c.Header(AMZ_VERSION_ID_HEADER, versionId)
	}
	SendS3SuccessResp(c, http.StatusOK, map[string]string{ETAG_HEADER: RFC7232ETag(etag)}, nil)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CopyObject.html
func CopyObjectHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	lockMode := c.GetHeader(AMZ_OBJECT_LOCK_MODE_HEADER)
	retainUntil := c.GetHeader(AMZ_OBJECT_LOCK_RETAIN_UNTIL_HEADER)
	copySourceBucket, copySourceKey, err := splitCopySourceHeader(bucket, c.GetHeader(AMZ_COPY_SOURCE_HEADER))
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}
	srcObjectArn := s3common.GenerateS3ObjectARN(copySourceBucket, copySourceKey)
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetObject, srcObjectArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	bucketArn := s3common.GenerateS3BucketARN(bucket)
	err = authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObject, bucketArn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	opts := s3.PutObjectOptions{
		CopySourceBucket: copySourceBucket,
		CopySourceKey:    copySourceKey,
		LockMode:         lockMode,
		RetainUntil:      retainUntil,
	}
	etag, versionId, err := service.CopyObject(ctx, objectKey, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if versionId != "" {
		c.Header(AMZ_VERSION_ID_HEADER, versionId)
	}
	SendS3SuccessResp(c, http.StatusOK, map[string]string{ETAG_HEADER: RFC7232ETag(etag)}, nil)
}

func PutObjectLegalHoldHandler(c *gin.Context) {
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3PutObjectLegalHold, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	var config LegalHoldConfiguration
	if err := readRequestBody(c.Request, &config); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucket), nil)
		return
	}

	ctx := c.Request.Context()
	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	versionId := c.Query(VersionIdParam)
	if err := object.UpdateObjectLegaHold(ctx, versionId, config.Status); err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func ObjectListOrDownloadHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetObject, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	// TODO: P0: Can we fetch instead of checking existence
	if service.IsExist(ctx, objectKey) {
		GetObjectHandler(c)
		return
	}

	ListObjectWithPrefixHandler(c)
}

func GetObjectHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	versionID := c.Query(VersionIdParam)

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	start, end, err := parseRangeHeader(c.GetHeader(RANGE_HEADER), bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	size := int64(object.GetMetadata(ctx).Size)
	if end != nil && *end >= size {
		err = s3common.GetInvalidRangeS3Error(bucket)
		SendS3ErrorResp(c, err, nil)
		return
	}

	opts := s3.GetObjectOptions{VersionId: versionID}
	if start != nil {
		opts.RangeStart = start
		if end == nil {
			t := size - 1
			end = &t
		}
		opts.RangeEnd = end
	}

	info, file, err := object.GetObject(ctx, opts)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	// Removing temp file that's reconstructed from shards
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	c.Header(CONTENT_TYPE_HEADER, info.ContentType)
	c.Header(ETAG_HEADER, RFC7232ETag(info.Etag))
	c.Header(LAST_MODIFIED_HEADER, formatLastModified(info.LastModified))

	contentLen := int64(info.Size)
	if opts.RangeStart != nil && opts.RangeEnd != nil {
		contentLen = *opts.RangeEnd - *opts.RangeStart + 1

		c.Header(ACCEPT_RANGES_RESP_HEADER, RangeUnitBytes)
		c.Header(CONTENT_RANGE_RESP_HEADER,
			fmt.Sprintf("%s %d-%d/%d", RangeUnitBytes, *opts.RangeStart, *opts.RangeEnd, size))
	}
	c.Header(CONTENT_LENGTH_HEADER, fmt.Sprintf("%d", contentLen))
	c.Status(http.StatusOK)

	_, _ = file.Seek(0, 0)
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Error("Error while sending object data to client")
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func GetObjectLegalHoldHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	versionId := c.Query(VersionIdParam)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3GetObjectLegalHold, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	status, err := object.GetObjectLegaHold(ctx, versionId)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	response := LegalHoldConfiguration{Xmlns: S3_AMZ_XMLNS, Status: status}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html
func DeleteObjectHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket, objectKey := getBucketNameAndObjectKeyFromHttpReq(c)
	arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
	err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteObject, arn)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	versionId := c.Query(VersionIdParam)
	if versionId != "" {
		arn := s3common.GenerateS3ObjectARN(bucket, objectKey)
		err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteObjectVersion, arn)
		if err != nil {
			SendS3ErrorResp(c, err, nil)
			return
		}
	}

	bypassGovRetention := c.GetHeader(AMZ_BYPASS_GOV_RETENTION_HEADER) == "true"
	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	object, err := service.GetObjectEntity(ctx, objectKey)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	deleteMarkerId, err := object.DeleteObject(ctx, versionId, bypassGovRetention)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	if deleteMarkerId != "" {
		c.Header(AMZ_VERSION_ID_HEADER, deleteMarkerId)
	}

	SendS3SuccessResp(c, http.StatusNoContent, nil, nil)
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html
func DeleteObjectsHandler(c *gin.Context) {
	ctx := c.Request.Context()
	bucket := c.Param("bucket")

	var deleteReq Delete
	if err := readRequestBody(c.Request, &deleteReq); err != nil {
		SendS3ErrorResp(c, s3common.GetMalformedXMLS3Error(bucket), nil)
		return
	}

	keys := make([]string, 0, len(deleteReq.Objects))
	for _, object := range deleteReq.Objects {
		arn := s3common.GenerateS3ObjectARN(bucket, object.Key)
		err := authzEngine.EvaluateRequest(*c.Request, s3common.PolicyActionS3DeleteObject, arn)
		if err != nil {
			logger.GetLogger(ctx).WithError(err).WithField("objectKey", object.Key).
				Error("Authorization failed for object deletion, skipping this object")
			continue
		}
		keys = append(keys, object.Key)
	}

	service, err := s3.NewObjectService(ctx, bucket)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	deletedKeys, err := service.DeleteObjects(ctx, keys)
	if err != nil {
		SendS3ErrorResp(c, err, nil)
		return
	}

	respKeys := make([]Deleted, 0, len(deletedKeys))
	for _, k := range deletedKeys {
		respKeys = append(respKeys, Deleted{Key: k})
	}

	response := DeleteResult{Xmlns: S3_AMZ_XMLNS, Deleted: respKeys}
	SendS3SuccessResp(c, http.StatusOK, nil, response)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAcl.html
func GetObjectACLHandler(c *gin.Context) {
	resp := AccessControlPolicy{
		Xmlns:             S3_AMZ_XMLNS,
		Owner:             &Owner{},
		AccessControlList: AccessControlList{Grants: []Grant{}},
	}

	SendS3SuccessResp(c, http.StatusOK, nil, resp)
}

// Documentation: Not supported in our application
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectAcl.html
func PutObjectACLHandler(c *gin.Context) {
	SendS3SuccessResp(c, http.StatusOK, nil, nil)
}

func formatLastModified(t string) string {
	lastModified, _ := utils.ConvertStringToTime(t)
	return lastModified.In(time.FixedZone("GMT", 0)).Format(time.RFC1123)
}
