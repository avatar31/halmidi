package models

// Ex: protoc --go_out=. --go-grpc_out=. person.proto
// 
// protoc --go_out=. models.proto

// type TagMap map[string]string

// type BucketMetadata struct {
// 	UUID            string               `json:"uuid"`
// 	Name            string               `json:"name"`
// 	CreationDate    string               `json:"creationDate"`
// 	Versioning      *BucketVersioning    `json:"versioning"`
// 	ObjectLocking   *BucketObjectLocking `json:"objectLocking"`
// 	Tags            TagMap               `json:"tags,omitempty"` // TODO: Do validation
// 	LifecycleConfig *LifecycleConfig     `json:"lifecycleConfig,omitempty"`
// 	Arn             string               `json:"arn"`
// 	Owner           *Owner               `json:"owner,omitempty"`

// 	// TODO: Do we need following fields?
// 	MfaDeleteEnabled bool               `json:"mfaDeleteEnabled"`
// 	Encryption       *BucketEncryption  `json:"encryption,omitempty"`
// 	Logging          *BucketLogging     `json:"logging,omitempty"`
// 	CorsRules        []CorsRule         `json:"corsRules,omitempty"`
// 	Website          *BucketWebsite     `json:"website,omitempty"`
// 	PolicyPublic     bool               `json:"policyPublic"`
// 	ObjectOwnership  string             `json:"objectOwnership,omitempty"` // BucketOwnerEnforced, etc.
// 	Replication      *ReplicationConfig `json:"replication,omitempty"`
// }

// type BucketVersioning struct {
// 	Status utils.VersioningStatus `json:"status"`
// }

// type BucketObjectLocking struct {
// 	ObjectLockEnabled utils.ObjectLockingStatus `json:"objectLockEnabled"`
// 	Rule              *LockRule                 `json:"rule,omitempty"`
// }

// type LockRule struct {
// 	DefaultRetention DefaultRetention `json:"defaultRetention"`
// }

// type DefaultRetention struct {
// 	Mode  utils.ObjectLockingMode `json:"mode,omitempty"` // "GOVERNANCE" or "COMPLIANCE"
// 	Days  int                     `json:"days,omitempty"`
// 	Years int                     `json:"years,omitempty"`
// }

// type LegalHold struct {
// 	Status utils.LegalHoldStatus `json:"status"` // "ON" or "OFF"
// }

// type BucketEncryption struct {
// 	Algorithm string `json:"algorithm"` // e.g., AES256 or aws:kms
// 	KmsKeyID  string `json:"kmsKeyId,omitempty"`
// }

// type BucketLogging struct {
// 	TargetBucket string `json:"targetBucket"`
// 	TargetPrefix string `json:"targetPrefix"`
// }

// type CorsRule struct {
// 	AllowedMethods []string `json:"allowedMethods"`
// 	AllowedOrigins []string `json:"allowedOrigins"`
// 	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
// 	ExposeHeaders  []string `json:"exposeHeaders,omitempty"`
// 	MaxAgeSeconds  int      `json:"maxAgeSeconds,omitempty"`
// }

// type BucketWebsite struct {
// 	IndexDocument         string `json:"indexDocument,omitempty"`
// 	ErrorDocument         string `json:"errorDocument,omitempty"`
// 	RedirectAllRequestsTo string `json:"redirectAllRequestsTo,omitempty"`
// }

// type ReplicationConfig struct {
// 	Role  string            `json:"role"`
// 	Rules []ReplicationRule `json:"rules"`
// }

// type ReplicationRule struct {
// 	ID           string `json:"id,omitempty"`
// 	Status       string `json:"status"`
// 	Prefix       string `json:"prefix,omitempty"`
// 	DestBucket   string `json:"destinationBucket"`
// 	StorageClass string `json:"storageClass,omitempty"`
// }

// TODO: Update LastModidfied everywhere
// type ObjectMetadata struct {
// 	// S3 fields
// 	UUID              string            `json:"uuid"`
// 	Key               string            `json:"key"`
// 	Size              uint64            `json:"size"`
// 	LastModified      string            `json:"lastModified"`
// 	ETag              string            `json:"etag"`
// 	ContentType       string            `json:"contentType,omitempty"`
// 	NextVersionNumber uint16            `json:"nextVersionNumber,omitempty"`
// 	ObjectLock        *ObjectLock       `json:"objectLock,omitempty"`
// 	Tags              TagMap            `json:"tags,omitempty"`
// 	Owner             *Owner            `json:"owner,omitempty"`
// 	Arn               string            `json:"arn"`
// 	Metadata          map[string]string `json:"metadata,omitempty"` // x-amz-meta-* headers

// 	// Internal fields
// 	DataShards   int             `json:"dataShards"`
// 	ParityShards int             `json:"parityShards"`
// 	Shards       []erasure.Shard `json:"shards"`

// 	// TODO: Do we need following fields?
// 	ContentEncoding string `json:"contentEncoding,omitempty"`
// 	ContentLanguage string `json:"contentLanguage,omitempty"`
// 	KmsKeyID        string `json:"kmsKeyId,omitempty"`     // if KMS is used
// 	SSEAlgorithm    string `json:"sseAlgorithm,omitempty"` // AES256 or aws:kms
// 	Expires         string `json:"expires,omitempty"`      // Expiration date if set
// }

// type ObjectLock struct {
// 	Mode            *utils.ObjectLockingMode `json:"mode"` // "GOVERNANCE" or "COMPLIANCE"
// 	RetainUntil     *string                  `json:"retainUntil"`
// 	LegalHoldStatus *utils.LegalHoldStatus   `json:"legalHoldStatus,omitempty"`
// }

// type ObjectVersion struct {
// 	ID             string      `json:"id"`
// 	Name           string      `json:"name"` // v<VersionNumber> i.e v1, v2 ...
// 	Key            string      `json:"key"`
// 	LastModified   string      `json:"lastModified"`
// 	ETag           string      `json:"eTag,omitempty"`
// 	IsLatest       bool        `json:"isLatest"`
// 	IsDeleteMarker *bool       `json:"isDeleteMarker,omitempty"`
// 	Size           uint64      `json:"size"`
// 	ObjectLock     *ObjectLock `json:"objectLock,omitempty"`
// 	Owner          *Owner      `json:"owner,omitempty"`

// 	// Internal fields
// 	DataShards   int             `json:"dataShards"`
// 	ParityShards int             `json:"parityShards"`
// 	Shards       []erasure.Shard `json:"shards"`
// }

// type MultipartUploadPart struct {
// 	UUID       string `json:"uploadUUID"`
// 	PartNumber int    `json:"partNumber"`
// 	ETag       string `json:"etag"`
// 	Size       int64  `json:"size"`
// 	Path       string `json:"path"`
// }

// TODO: Add content-type
// type MultipartUpload struct {
// 	Bucket          string                  `json:"bucket"`
// 	Key             string                  `json:"key"`
// 	LockMode        utils.ObjectLockingMode `json:"mode,omitempty"`
// 	RetainUntil     string                  `json:"retainUntil,omitempty"`
// 	LegalHoldStatus utils.LegalHoldStatus   `json:"legalHoldStatus,omitempty"`
// }

// type Owner struct {
// 	ID          string `json:"id"`
// 	DisplayName string `json:"displayName"`
// }

// type LifecycleConfig struct {
// 	Rules []LifeCycleRule `json:"rules"`
// }

// // TODO: DOCUMENTATION: Prefix not supported
// type LifeCycleRule struct {
// 	ID                             string                          `json:"id,omitempty"`
// 	Prefix                         string                          `json:"prefix,omitempty"`
// 	Status                         utils.LifiCycleStatus           `json:"status"` // "Enabled" or "Disabled"
// 	Filter                         *Filter                         `json:"filter,omitempty"`
// 	Expiration                     *Expiration                     `json:"expiration,omitempty"`
// 	NoncurrentVersionExpiration    *NoncurrentVersionExpiration    `json:"noncurrentVersionExpiration,omitempty"`
// 	AbortIncompleteMultipartUpload *AbortIncompleteMultipartUpload `json:"abortIncompleteMultipartUpload,omitempty"`
// }

// type Filter struct {
// 	Prefix                string     `json:"prefix,omitempty"`
// 	Tag                   *Tag       `json:"tag,omitempty"`
// 	ObjectSizeGreaterThan *int64     `json:"objectSizeGreaterThan,omitempty"`
// 	ObjectSizeLessThan    *int64     `json:"objectSizeLessThan,omitempty"`
// 	And                   *AndFilter `json:"and,omitempty"`
// }

// type AndFilter struct {
// 	Tags                  []Tag  `json:"tags"`
// 	ObjectSizeGreaterThan *int64 `json:"objectSizeGreaterThan,omitempty"`
// 	ObjectSizeLessThan    *int64 `json:"objectSizeLessThan,omitempty"`
// }

// type Tag struct {
// 	Key   string `json:"key"`
// 	Value string `json:"value"`
// }

// type Expiration struct {
// 	Date                      *string `json:"date,omitempty"`
// 	Days                      *int    `json:"days,omitempty"`
// 	ExpiredObjectDeleteMarker *bool   `json:"expiredObjectDeleteMarker,omitempty"`
// }

// type NoncurrentVersionExpiration struct {
// 	NewerNoncurrentVersions int `json:"newerNoncurrentVersions"`
// 	NoncurrentDays          int `json:"noncurrentDays"`
// }

// type AbortIncompleteMultipartUpload struct {
// 	DaysAfterInitiation int `json:"daysAfterInitiation"`
// }

// type User struct {
// 	Path             string   `json:"path"`
// 	UserName         string   `json:"userName"`
// 	UserId           string   `json:"userId"`
// 	Arn              string   `json:"arn,omitempty"`
// 	CreateDate       string   `json:"createDate"`
// 	PasswordLastUsed string   `json:"passwordLastUsed,omitempty"`
// 	AccessKeys       []string `json:"accessKeys,omitempty"`
// }

// type UserAccessKey struct {
// 	UserName        string                `json:"userName"`
// 	AccessKeyId     string                `json:"accessKeyId"`
// 	SecretAccessKey string                `json:"secretAccessKey"`
// 	Status          utils.AccessKeyStatus `json:"status"`
// 	CreateDate      string                `json:"createDate"`
// }
