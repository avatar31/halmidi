package s3nativeapihandlers

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/avatar31/omashu"
	"github.com/gin-gonic/gin"

	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
)

const (
	xmlHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
)

func SendS3ErrorResp(c *gin.Context, err error, headers map[string]string) {
	// TODO: P0: Remove this check once we start accepting requests on followers
	if errors.Is(err, omashu.ErrNotLeader) {
		resp := ErrorResponse{
			Code:      "ServiceUnavailable",
			Message:   "Request must be directed to the leader node",
			RequestID: c.GetHeader(AMZ_REQUEST_ID),
			HostID:    ServerName,
		}
		c.Header("Retry-After", "1")
		finalResp, _ := xml.MarshalIndent(resp, "", "  ")
		c.Status(http.StatusServiceUnavailable)
		_, _ = c.Writer.Write([]byte(xmlHeader))
		_, _ = c.Writer.Write(finalResp)
		c.Abort()
		return
	}

	s3Error, ok := err.(s3common.S3Error)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	resp := ErrorResponse{
		// Xmlns:      AMZ_XMLNS,
		Code:       s3Error.S3ErrorCode,
		Message:    s3Error.Message,
		BucketName: s3Error.Bucket,
		RequestID:  c.GetHeader(AMZ_REQUEST_ID),
		HostID:     ServerName,
	}

	sendErrorResp(c, s3Error, headers, resp)
}

func SendIAMErrorResp(c *gin.Context, err error, headers map[string]string) {
	s3Error, ok := err.(s3common.S3Error)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	resp := IAMErrorResponse{
		Xmlns: IAM_AMZ_XMLNS,
		Error: IAMError{
			Type:    "Sender", // TODO: Check this
			Message: s3Error.Message,
			Code:    s3Error.S3ErrorCode,
		},
		RequestID: c.GetHeader(AMZ_REQUEST_ID),
	}

	sendErrorResp(c, s3Error, headers, resp)
}

func sendErrorResp(c *gin.Context, s3Error s3common.S3Error, headers map[string]string, resp any) {
	log := logger.GetLogger(c.Request.Context())

	finalResp, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.WithError(err).Error("Error marshalling success response")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.WithFields(map[string]any{
		"statusCode": s3Error.HttpStatusCode,
		"errMessage": s3Error.Message,
		"s3Code":     s3Error.S3ErrorCode,
	}).Error("Error in api. Throwing error")

	c.Header(SERVER_RESP_HEADER, ServerName)
	c.Header(CONTENT_TYPE_HEADER, "application/xml; charset=utf-8")
	for k, v := range headers {
		c.Header(k, v)
	}

	c.Status(s3Error.HttpStatusCode)
	_, _ = c.Writer.Write([]byte(xmlHeader))
	_, _ = c.Writer.Write(finalResp)
}

func SendS3SuccessResp(c *gin.Context, httpStatus int, respHeaders map[string]string, resp any) {
	c.Header(SERVER_RESP_HEADER, ServerName)
	for k, v := range respHeaders {
		c.Header(k, v)
	}

	if resp == nil {
		c.Status(httpStatus)
		return
	}

	finalResp, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		log := logger.GetLogger(c.Request.Context())
		log.WithError(err).Error("Error marshalling success response")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(httpStatus)
	c.Header(CONTENT_TYPE_HEADER, "application/xml; charset=utf-8")
	_, _ = c.Writer.Write([]byte(xmlHeader))
	_, _ = c.Writer.Write(finalResp)
}

func getBucketNameAndObjectKeyFromHttpReq(c *gin.Context) (string, string) {
	bucketName := c.Param("bucket")
	objectKey := strings.TrimPrefix(c.Param("object"), "/")

	return bucketName, objectKey
}

// https://datatracker.ietf.org/doc/html/rfc7232#section-2.3.3
func RFC7232ETag(etag string) string {
	return "\"" + etag + "\""
}

func parseRangeHeader(rangeHeader, bucket string) (*int64, *int64, error) {
	if rangeHeader == "" {
		return nil, nil, nil
	}

	errMsg := fmt.Sprintf("The requested range '%s' cannot be satisfied.", rangeHeader)

	splitValues := strings.SplitN(rangeHeader, "=", 2)
	if len(splitValues) != 2 || splitValues[0] != RangeUnitBytes {
		return nil, nil, s3common.GetInvalidRangeS3Error(bucket, errMsg)
	}

	rangeParts := strings.SplitN(splitValues[1], "-", 2)
	if len(rangeParts) != 2 {
		return nil, nil, s3common.GetInvalidRangeS3Error(bucket, errMsg)
	}

	start, err := strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil || start < 0 {
		return nil, nil, s3common.GetInvalidRangeS3Error(bucket, errMsg)
	}

	if rangeParts[1] == "" {
		return &start, nil, nil
	}

	end, err := strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil || end < start {
		return nil, nil, s3common.GetInvalidRangeS3Error(bucket, errMsg)
	}

	return &start, &end, nil
}

// Expected format: /source-bucket/source-object-key
func splitCopySourceHeader(bucket, source string) (string, string, error) {
	parts := strings.SplitN(strings.TrimPrefix(source, "/"), "/", 2)
	if len(parts) != 2 {
		return "", "", s3common.GetInvalidRequestS3Error(bucket, "Invalid x-amz-copy-source header")
	}

	return parts[0], parts[1], nil
}

func getResponseMetadata(c *gin.Context) ResponseMetadata {
	return ResponseMetadata{RequestId: c.GetHeader(AMZ_REQUEST_ID)}
}

func getPrefixParam(c *gin.Context) (string, error) {
	prefix, err := url.QueryUnescape(c.Query(PrefixParam))
	if err != nil {
		return "", s3common.GetInvalidArgumentS3Error("", "Invalid prefix")
	}
	return prefix, nil
}
