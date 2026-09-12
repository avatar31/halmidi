package s3common

import (
	"fmt"
	"slices"
	"strings"

	"github.com/avatar31/halmidi/internal/core/namespace"
)

// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html
// 
// Example ARN for an IAM user:
// arn:aws:iam::123456789012:user/sachin
//
// | Part           | Value | Description                                               |
// | -------------- | ----- | --------------------------------------------------------- |
// | `arn`          | -     | Prefix for all AWS ARNs                                   |
// | `aws`          | -     | Partition (usually `aws`, unless using GovCloud or China) |
// | `iam`          | -     | AWS service name                                          |
// | *empty*        | -     | IAM is global, so no region                               |
// | `123456789012` | -     | AWS Account ID                                            |
// | `user/sachin`  | -     | Resource type and name                                    |

const (
	ARN_PREFIX      = "arn:aws"
	ARN_PARTS_COUNT = 6
)

func GenerateUserARN(resourceName string) string {
	return fmt.Sprintf("%s:%s::%s:user/%s", ARN_PREFIX, IAM_SERVICE, namespace.DEFAULT_NAMESPACE, resourceName)
}

func GenerateGroupARN(resourceName string) string {
	return fmt.Sprintf("%s:%s::%s:group/%s", ARN_PREFIX, IAM_SERVICE, namespace.DEFAULT_NAMESPACE, resourceName)
}

func GeneratePolicyARN(resourceName string) string {
	return fmt.Sprintf("%s:%s::%s:policy/%s", ARN_PREFIX, IAM_SERVICE, namespace.DEFAULT_NAMESPACE, resourceName)
}

func GenerateS3BucketARN(resourceName string) string {
	return fmt.Sprintf("%s:%s:::%s", ARN_PREFIX, S3_SERVICE, resourceName)
}

func GenerateS3ObjectARN(bucket, resourceName string) string {
	return fmt.Sprintf("%s/%s", GenerateS3BucketARN(bucket), resourceName)
}

func IsValidARNLength(arn string) bool {
	length := len(arn)
	return length <= MAX_ALLOWED_ARN_LENGTH
}

func IsValidARN(arn string) bool {
	if !IsValidARNLength(arn) {
		return false
	}

	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 {
		return false
	}

	if parts[0] != "arn" || parts[1] != "aws" {
		return false
	}

	// TODO:P1: Should we accept policies of all services?
	service := parts[2]
	if !slices.Contains(SUPPORTED_SERVICES, service) {
		return false
	}

	region := parts[3]
	if region != "" {
		return false
	}

	accountID := parts[4]
	if (service == S3_SERVICE && accountID != "") {
		return false
	}

	resource := parts[5]
	return resource != ""
}
