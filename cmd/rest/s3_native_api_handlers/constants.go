package s3nativeapihandlers

const (
	S3_AMZ_XMLNS  = "http://s3.amazonaws.com/doc/2006-03-01/"
	IAM_AMZ_XMLNS = "https://iam.amazonaws.com/doc/2010-05-08/"

	IAM_API_VERSION = "2010-05-08"

	AMZ_BUCKET_OBJECT_LOCK_ENABLED_HEADER = "x-amz-bucket-object-lock-enabled"
	AMZ_OBJECT_LOCK_MODE_HEADER           = "x-amz-object-lock-mode"
	AMZ_OBJECT_LOCK_RETAIN_UNTIL_HEADER   = "x-amz-object-lock-retain-until-date"
	AMZ_OBJECT_LOCK_LEGAL_HOLD_HEADER     = "x-amz-object-lock-legal-hold"
	AMZ_BYPASS_GOV_RETENTION_HEADER       = "x-amz-bypass-governance-retention"
	AMZ_VERSION_ID_HEADER                 = "x-amz-version-id"
	AMZ_COPY_SOURCE_HEADER                = "x-amz-copy-source"
	AMZ_COPY_SOURCE_RANGE_HEADER          = "x-amz-copy-source-range"
	AMZ_REQUEST_ID                        = "x-amz-request-id"
	AMZ_BUCKET_ARN                        = "x-amz-bucket-arn"

	CONTENT_LENGTH_HEADER = "Content-Length"
	CONTENT_TYPE_HEADER   = "Content-Type"
	ETAG_HEADER           = "ETag"
	LAST_MODIFIED_HEADER  = "Last-Modified"
	RANGE_HEADER          = "Range"

	SERVER_RESP_HEADER        = "Server"
	LOCATION_RESP_HEADER      = "Location"
	ACCEPT_RANGES_RESP_HEADER = "Accept-Ranges"
	CONTENT_RANGE_RESP_HEADER = "Content-Range"
)

// Query Params
const (
	ContinuationTokenParam = "continuation-token"
	DelimiterParam         = "delimiter"
	EncodingTypeParam      = "encoding-type"
	MaxKeysParam           = "max-keys"
	StartAfterParam        = "start-after"
	LegalHoldParam         = "legal-hold"
	ListTypeParam          = "list-type"
	MaxBucketsParam        = "max-buckets"
	ObjectLockParam        = "object-lock"
	PartNumberParam        = "partNumber"
	PrefixParam            = "prefix"
	TaggingParam           = "tagging"
	UploadsParam           = "uploads"
	UploadIdParam          = "uploadId"
	VersioningParam        = "versioning"
	VersionsParam          = "versions"
	VersionIdParam         = "versionId"
	LifeCycleParam         = "lifecycle"
	DeleteParam            = "delete"
	ACLParam               = "acl"
	OwnershipControlsParam = "ownershipControls"
)

// AWS IAM Actions
const (
	AddUserToGroupAction            = "AddUserToGroup"
	AttachGroupPolicyAction         = "AttachGroupPolicy"
	AttachUserPolicyAction          = "AttachUserPolicy"
	CreateAccessKeyAction           = "CreateAccessKey"
	CreateGroupAction               = "CreateGroup"
	CreatePolicyAction              = "CreatePolicy"
	CreatePolicyVersionAction       = "CreatePolicyVersion"
	CreateUserAction                = "CreateUser"
	DeleteAccessKeyAction           = "DeleteAccessKey"
	DeleteGroupAction               = "DeleteGroup"
	DeleteGroupPolicyAction         = "DeleteGroupPolicy"
	DeletePolicyAction              = "DeletePolicy"
	DeletePolicyVersionAction       = "DeletePolicyVersion"
	DeleteUserAction                = "DeleteUser"
	DeleteUserPolicyAction          = "DeleteUserPolicy"
	DetachGroupPolicyAction         = "DetachGroupPolicy"
	DetachUserPolicyAction          = "DetachUserPolicy"
	GetAccessKeyLastUsedAction      = "GetAccessKeyLastUsed"
	GetGroupAction                  = "GetGroup"
	GetGroupPolicyAction            = "GetGroupPolicy"
	GetPolicyAction                 = "GetPolicy"
	GetPolicyVersionAction          = "GetPolicyVersion"
	GetUserAction                   = "GetUser"
	GetUserPolicyAction             = "GetUserPolicy"
	ListAccessKeysAction            = "ListAccessKeys"
	ListAttachedGroupPoliciesAction = "ListAttachedGroupPolicies"
	ListAttachedUserPoliciesAction  = "ListAttachedUserPolicies"
	ListEntitiesForPolicyAction     = "ListEntitiesForPolicy"
	ListGroupPoliciesAction         = "ListGroupPolicies"
	ListGroupsAction                = "ListGroups"
	ListGroupsForUserAction         = "ListGroupsForUser"
	ListPoliciesAction              = "ListPolicies"
	ListPolicyTagsAction            = "ListPolicyTags"
	ListPolicyVersionsAction        = "ListPolicyVersions"
	ListUserPoliciesAction          = "ListUserPolicies"
	ListUsersAction                 = "ListUsers"
	ListUserTagsAction              = "ListUserTags"
	PutGroupPolicyAction            = "PutGroupPolicy"
	PutUserPolicyAction             = "PutUserPolicy"
	RemoveUserFromGroupAction       = "RemoveUserFromGroup"
	SetDefaultPolicyVersionAction   = "SetDefaultPolicyVersion"
	TagPolicyAction                 = "TagPolicy"
	TagUserAction                   = "TagUser"
	UntagPolicyAction               = "UntagPolicy"
	UntagUserAction                 = "UntagUser"
	UpdateAccessKeyAction           = "UpdateAccessKey"
	UpdateGroupAction               = "UpdateGroup"
	UpdateUserAction                = "UpdateUser"
)

// Default Values
const (
	ServerName      = "Halmidi"
	DefaultMaxItems = 1000
)

// Misc
const (
	RangeUnitBytes = "bytes"
)
