package s3common

import "github.com/google/uuid"

// AWS Specific
const (
	PROXY_AWS_REGION          = "us-east-1"
	S3_STANDARD_STORAGE_CLASS = "STANDARD"
	SUPPORTED_POLICY_VERSION  = "2012-10-17"
)

var (
	// Regex
	BucketNameRegex      = `^[a-z0-9][a-z0-9\-\.]{1,61}[a-z0-9]$`
	IAMResourceNameRegex = `^[\w+=,.@-]+$`
	TagKeyValueRegex     = `^[a-zA-Z+\-=\._:/]+$`
	IAMResourcePathRegex = `^((/[A-Za-z0-9\.,\+@=_-]+)*)/$`
	PolicyDocumentRegex  = `^[\t\n\r -\xFF]+$` // `^[\u0009\u000A\u000D\u0020-\u00FF]+$`
	VersionIdRegex       = `^v[1-9][0-9]*(\.[A-Za-z0-9-]*)?$`

	SUPPORTED_SERVICES = []string{S3_SERVICE, IAM_SERVICE}
)

// S3 Specific Limits
const (
	MIN_ALLOWED_BUCKET_NAME_LENGTH       = 3
	MAX_ALLOWED_BUCKET_NAME_LENGTH       = 256
	MAX_ALLOWED_OBJECT_KEY_LENGTH        = 1024
	MAX_ALLOWED_USER_NAME_LENGTH         = 64
	MAX_ALLOWED_IAM_RESOURCE_NAME_LENGTH = 128
	MIN_ALLOWED_PERM_BOUNDARY_LENGTH     = 20
	MAX_ALLOWED_PERM_BOUNDARY_LENGTH     = 2048
	MAX_ALLOWED_TAGS                     = 50
	MAX_ALLOWED_TAG_KEY_LENGTH           = 128
	MAX_ALLOWED_TAG_VALUE_LENGTH         = 256
	MAX_ALLOWED_LIFECYCLE_RULES          = 1000
	MAX_ALLOWED_USER_ACCESS_KEYS         = 2
	MAX_ALLOWED_ARN_LENGTH               = 2048
	MAX_ALLOWED_POLICY_DESC_LENGTH       = 1000
	MAX_ALLOWED_IAM_RES_PATH_LENGTH      = 512
	MAX_ALLOWED_POLICY_DOCUMENT_LENGTH   = 131072 // 128 KB
	MAX_ALLOWED_POLICY_VERSIONS          = 5
	MAX_ALLOWED_POLICIES_PER_ENTITY      = 10
)

// Internal
const (
	NO_VERSION                = "NO_VERSION"
	S3_SERVICE                = "s3"
	IAM_SERVICE               = "iam"
	DEFAULT_SERVICE           = S3_SERVICE
	DEFAULT_IAM_RESOURCE_PATH = "/"
)

var (
	ObjectNameSpaceUUID = uuid.MustParse("69556665-5812-4c18-b03f-55dda4078815")
)

type VersioningStatus string

func (v VersioningStatus) Validate(bucket string) error {
	if v == "" || (v != VersioningStatusEnabled && v != VersioningStatusSuspended) {
		return GetInvalidArgumentS3Error(bucket, "Invalid value for Versioning status.")
	}
	return nil
}

func (v VersioningStatus) String() string {
	return string(v)
}

const (
	VersioningStatusEnabled   VersioningStatus = "Enabled"
	VersioningStatusSuspended VersioningStatus = "Suspended"
)

type ObjectLockingStatus string

func (v ObjectLockingStatus) String() string {
	return string(v)
}

const (
	ObjectLockingStatusEnabled ObjectLockingStatus = "Enabled"
)

type ObjectLockingMode string

func (v ObjectLockingMode) String() string {
	return string(v)
}

func (v ObjectLockingMode) Validate(bucket string) error {
	if v == "" || (v != ObjectLockingModeGovernance && v != ObjectLockingModeCompliance) {
		return GetInvalidArgumentS3Error(bucket, "Invalid value for Object Locking Mode.")
	}
	return nil
}

const (
	ObjectLockingModeGovernance ObjectLockingMode = "GOVERNANCE"
	ObjectLockingModeCompliance ObjectLockingMode = "COMPLIANCE"
)

type LegalHoldStatus string

func (v LegalHoldStatus) String() string {
	return string(v)
}

func (v LegalHoldStatus) Validate(bucket string) error {
	if v == "" || (v != LegalHoldStatusOn && v != LegalHoldStatusOff) {
		return GetInvalidArgumentS3Error(bucket, "Invalid value for Object Legal Hold.")
	}
	return nil
}

const (
	LegalHoldStatusOn  LegalHoldStatus = "ON"
	LegalHoldStatusOff LegalHoldStatus = "OFF"
)

type LifiCycleStatus string

func (v LifiCycleStatus) String() string {
	return string(v)
}

func (v LifiCycleStatus) Validate(bucket string) error {
	if v == "" || (v != LifiCycleStatusEnabled && v != LifiCycleStatusDisabled) {
		return GetInvalidArgumentS3Error(bucket, "Invalid value for Lifecycle status.")
	}
	return nil
}

const (
	LifiCycleStatusEnabled  LifiCycleStatus = "Enabled"
	LifiCycleStatusDisabled LifiCycleStatus = "Disabled"
)

type AccessKeyStatus string

func (v AccessKeyStatus) String() string {
	return string(v)
}

func (v AccessKeyStatus) Validate(username string) error {
	if v == "" || (v != AccessKeyStatusActive && v != AccessKeyStatusInactive) {
		return GetInvalidArgumentS3Error(username, "Invalid value for AccessKey status.")
	}
	return nil
}

const (
	AccessKeyStatusActive   AccessKeyStatus = "Active"
	AccessKeyStatusInactive AccessKeyStatus = "Inactive"
	AccessKeyStatusExpired  AccessKeyStatus = "Expired"
)

type PolicyStatementEffect string

func (v PolicyStatementEffect) String() string {
	return string(v)
}

func (v PolicyStatementEffect) Validate() error {
	if v == "" || (v != PolicyStatementEffectAllow && v != PolicyStatementEffectDeny) {
		return GetMalformedPolicyDocumentS3Error("", "Invalid value for Policy Statement Effect.")
	}
	return nil
}

const (
	PolicyStatementEffectAllow PolicyStatementEffect = "Allow"
	PolicyStatementEffectDeny  PolicyStatementEffect = "Deny"
)
