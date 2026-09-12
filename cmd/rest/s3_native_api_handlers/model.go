package s3nativeapihandlers

import (
	"encoding/xml"

	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
)

type ListAllMyBucketsResult struct {
	XMLName           xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns             string   `xml:"xmlns,attr"`
	Owner             *Owner   `xml:"Owner"`
	Buckets           Buckets  `xml:"Buckets"`
	ContinuationToken string   `xml:"ContinuationToken"`
	Prefix            string   `xml:"Prefix"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_Owner.html
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type Buckets struct {
	Bucket []BucketInfo `xml:"Bucket"`
}

type BucketInfo struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html#API_ListObjectsV2_ResponseSyntax
type ListBucketResult struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	Xmlns                 string         `xml:"xmlns,attr"`
	IsTruncated           bool           `xml:"IsTruncated"`
	Contents              []S3Object     `xml:"Contents"`
	Name                  string         `xml:"Name"`
	Prefix                string         `xml:"Prefix"`
	Delimiter             string         `xml:"Delimiter,omitempty"`
	MaxKeys               int            `xml:"MaxKeys"`
	CommonPrefixes        []CommonPrefix `xml:"CommonPrefixes,omitempty"`
	EncodingType          string         `xml:"EncodingType,omitempty"`
	KeyCount              int            `xml:"KeyCount"`
	ContinuationToken     string         `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string         `xml:"NextContinuationToken,omitempty"`
	StartAfter            string         `xml:"StartAfter,omitempty"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_Object.html
type S3Object struct {
	ChecksumAlgorithm string         `xml:"ChecksumAlgorithm,omitempty"` // CRC32 | CRC32C | SHA1 | SHA256 | CRC64NVME
	ChecksumType      string         `xml:"ChecksumType,omitempty"`      // COMPOSITE | FULL_OBJECT
	ETag              string         `xml:"ETag"`
	Key               string         `xml:"Key"`
	LastModified      string         `xml:"LastModified"`
	Owner             *Owner         `xml:"Owner,omitempty"`
	RestoreStatus     *RestoreStatus `xml:"RestoreStatus,omitempty"`
	Size              uint64         `xml:"Size"`
	StorageClass      string         `xml:"StorageClass"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_RestoreStatus.html
type RestoreStatus struct {
	IsRestoreInProgress bool `xml:"IsRestoreInProgress"`
	RestoreExpiryDate   bool `xml:"RestoreExpiryDate"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CommonPrefix.html
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html#API_CreateBucket_RequestSyntax
type CreateBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	Xmlns              string   `xml:"xmlns,attr"`
	LocationConstraint string   `xml:"LocationConstraint,omitempty"`
	Location           Location `xml:"Location"`
	Bucket             Bucket   `xml:"Bucket"`
	Tags               []Tag    `xml:"Tags"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_LocationInfo.html
type Location struct {
	XMLName xml.Name `xml:"Location"`
	Name    string   `xml:"Name"`
	Type    string   `xml:"Type"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_BucketInfo.html
type Bucket struct {
	XMLName        xml.Name `xml:"Bucket"`
	DataRedundancy string   `xml:"DataRedundancy"`
	Type           string   `xml:"Type"`
}

type VersioningConfiguration struct {
	XMLName xml.Name                  `xml:"VersioningConfiguration"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Status  s3common.VersioningStatus `xml:"Status"`
}

type ObjectLockConfiguration struct {
	XMLName           xml.Name                     `xml:"ObjectLockConfiguration"`
	Xmlns             string                       `xml:"xmlns,attr"`
	ObjectLockEnabled s3common.ObjectLockingStatus `xml:"ObjectLockEnabled"`
	Rule              *LockRuleConfiguration       `xml:"Rule,omitempty"`
}

type LockRuleConfiguration struct {
	DefaultRetention DefaultRetentionConfiguration `xml:"DefaultRetention"`
}

type DefaultRetentionConfiguration struct {
	Mode  s3common.ObjectLockingMode `xml:"Mode,omitempty"` // "GOVERNANCE" or "COMPLIANCE"
	Days  int32                      `xml:"Days,omitempty"`
	Years int32                      `xml:"Years,omitempty"`
}

type LegalHoldConfiguration struct {
	XMLName xml.Name                 `xml:"LegalHold"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Status  s3common.LegalHoldStatus `xml:"Status"` // "ON" or "OFF"
}

type InitiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadId string   `xml:"UploadId"`
}

type CompleteMultipartUpload struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Parts   []Part   `xml:"Part"`
}

type Part struct {
	XMLName           xml.Name `xml:"Part"`
	ChecksumCRC32     string   `xml:"ChecksumCRC32,omitempty"`
	ChecksumCRC32C    string   `xml:"ChecksumCRC32C,omitempty"`
	ChecksumCRC64NVME string   `xml:"ChecksumCRC64NVME,omitempty"`
	ChecksumSHA1      string   `xml:"ChecksumSHA1,omitempty"`
	ChecksumSHA256    string   `xml:"ChecksumSHA256,omitempty"`
	ETag              string   `xml:"ETag"`
	PartNumber        int32    `xml:"PartNumber"`
}

type CompleteMultipartUploadResult struct {
	XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}

type Tagging struct {
	XMLName xml.Name `xml:"Tagging"`
	Xmlns   string   `xml:"xmlns,attr"`
	TagSet  TagSet   `xml:"TagSet"`
}

type TagSet struct {
	XMLName xml.Name `xml:"TagSet"`
	Tags    []Tag    `xml:"Tag"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_Tag.html
type Tag struct {
	Key   string `xml:"Key" json:"Key"`
	Value string `xml:"Value" json:"Value"`
}

type ListVersionsResult struct {
	XMLName             xml.Name       `xml:"ListVersionsResult"`
	Xmlns               string         `xml:"xmlns,attr"`
	Name                string         `xml:"Name"`
	Prefix              string         `xml:"Prefix"`
	KeyMarker           string         `xml:"KeyMarker"`
	VersionIdMarker     string         `xml:"VersionIdMarker"`
	NextKeyMarker       string         `xml:"NextKeyMarker"`
	NextVersionIdMarker string         `xml:"NextVersionIdMarker"`
	MaxKeys             int            `xml:"MaxKeys"`
	IsTruncated         bool           `xml:"IsTruncated"`
	Versions            []Version      `xml:"Version"`
	DeleteMarkers       []DeleteMarker `xml:"DeleteMarker"`
}

type Version struct {
	Key          string  `xml:"Key"`
	VersionId    *string `xml:"VersionId"`
	IsLatest     bool    `xml:"IsLatest"`
	LastModified string  `xml:"LastModified"`
	ETag         string  `xml:"ETag"`
	Size         uint64  `xml:"Size"`
	StorageClass string  `xml:"StorageClass"`
	Owner        Owner   `xml:"Owner"`
}

type DeleteMarker struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
	Owner        Owner  `xml:"Owner"`
}

// XML response format similar to AWS
type CreateUserResponse struct {
	XMLName          xml.Name         `xml:"CreateUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	Result           CreateUserResult `xml:"CreateUserResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type CreateUserResult struct {
	XMLName xml.Name `xml:"CreateUserResult"`
	User    member   `xml:"User"`
}

type UpdateUserResponse struct {
	XMLName          xml.Name         `xml:"UpdateUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	UpdateUserResult UpdateUserResult `xml:"UpdateUserResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type UpdateUserResult struct {
	XMLName xml.Name `xml:"UpdateUserResult"`
	User    member   `xml:"User"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachUserPolicy.html#API_AttachUserPolicy_Examples
type AttachUserPolicyResponse struct {
	XMLName          xml.Name         `xml:"AttachUserPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedUserPolicies.html#API_ListAttachedUserPolicies_Examples
type ListAttachedUserPoliciesResponse struct {
	XMLName                        xml.Name                       `xml:"ListAttachedUserPoliciesResponse"`
	Xmlns                          string                         `xml:"xmlns,attr"`
	ListAttachedUserPoliciesResult ListAttachedUserPoliciesResult `xml:"ListAttachedUserPoliciesResult"`
	ResponseMetadata               ResponseMetadata               `xml:"ResponseMetadata"`
}

type ListAttachedUserPoliciesResult struct {
	XMLName          xml.Name         `xml:"ListAttachedUserPoliciesResult"`
	AttachedPolicies AttachedPolicies `xml:"AttachedPolicies"`
	IsTruncated      bool             `xml:"IsTruncated"`
	Marker           string           `xml:"Marker,omitempty"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserPolicies.html#API_ListUserPolicies_Examples
type ListUserPoliciesResponse struct {
	XMLName                xml.Name               `xml:"ListUserPoliciesResponse"`
	Xmlns                  string                 `xml:"xmlns,attr"`
	ListUserPoliciesResult ListUserPoliciesResult `xml:"ListUserPoliciesResult"`
	ResponseMetadata       ResponseMetadata       `xml:"ResponseMetadata"`
}

type ListUserPoliciesResult struct {
	XMLName     xml.Name    `xml:"ListUserPoliciesResult"`
	PolicyNames PolicyNames `xml:"PolicyNames"`
	IsTruncated bool        `xml:"IsTruncated"`
	Marker      string      `xml:"Marker,omitempty"`
}

type PolicyNames struct {
	XMLName xml.Name `xml:"PolicyNames"`
	Member  []string `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutUserPolicy.html#API_PutUserPolicy_Examples
type PutUserPolicyResponse struct {
	XMLName          xml.Name         `xml:"PutUserPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteUserPolicy.html#API_DeleteUserPolicy_Examples
type DeleteUserPolicyResponse struct {
	XMLName          xml.Name         `xml:"DeleteUserPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUserPolicy.html#API_GetUserPolicy_Examples
type GetUserPolicyResponse struct {
	XMLName             xml.Name            `xml:"GetUserPolicyResponse"`
	Xmlns               string              `xml:"xmlns,attr"`
	GetUserPolicyResult GetUserPolicyResult `xml:"GetUserPolicyResult"`
	ResponseMetadata    ResponseMetadata    `xml:"ResponseMetadata"`
}

type GetUserPolicyResult struct {
	XMLName        xml.Name `xml:"GetUserPolicyResult"`
	PolicyName     string   `xml:"PolicyName"`
	PolicyDocument string   `xml:"PolicyDocument"`
	UserName       string   `xml:"UserName"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachUserPolicy.html#API_DetachUserPolicy_Examples
type DetachUserPolicyResponse struct {
	XMLName          xml.Name         `xml:"DetachUserPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type DeleteUserResponse struct {
	XMLName          xml.Name         `xml:"DeleteUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type GetUserResponse struct {
	XMLName          xml.Name         `xml:"GetUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	Result           GetUserResult    `xml:"GetUserResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type GetUserResult struct {
	XMLName xml.Name `xml:"GetUserResult"`
	User    member   `xml:"User"`
}

type ListUsersResponse struct {
	XMLName          xml.Name         `xml:"ListUsersResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	Result           ListUsersResult  `xml:"ListUsersResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type ListUsersResult struct {
	XMLName     xml.Name `xml:"ListUsersResult"`
	Users       Users    `xml:"Users"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker,omitempty"`
}

type Users struct {
	XMLName xml.Name `xml:"Users"`
	Member  []member `xml:"member"`
}

type member struct {
	Path             string `xml:"Path"`
	UserName         string `xml:"UserName"`
	UserId           string `xml:"UserId"`
	Arn              string `xml:"Arn"`
	CreateDate       string `xml:"CreateDate"`
	PasswordLastUsed string `xml:"PasswordLastUsed,omitempty"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupsForUser.html#API_ListGroupsForUser_Examples
type ListGroupsForUserResponse struct {
	XMLName                 xml.Name                `xml:"ListGroupsForUserResponse"`
	Xmlns                   string                  `xml:"xmlns,attr"`
	ListGroupsForUserResult ListGroupsForUserResult `xml:"ListGroupsForUserResult"`
	ResponseMetadata        ResponseMetadata        `xml:"ResponseMetadata"`
}

type ListGroupsForUserResult struct {
	XMLName     xml.Name `xml:"ListGroupsForUserResult"`
	Groups      Groups   `xml:"Groups"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker,omitempty"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserTags.html#API_ListUserTags_Examples
type ListUserTagsResponse struct {
	XMLName            xml.Name           `xml:"ListUserTagsResponse"`
	Xmlns              string             `xml:"xmlns,attr"`
	ListUserTagsResult ListUserTagsResult `xml:"ListUserTagsResult"`
	ResponseMetadata   ResponseMetadata   `xml:"ResponseMetadata"`
}

type ListUserTagsResult struct {
	XMLName     xml.Name `xml:"ListUserTagsResult"`
	Tags        Tags     `xml:"Tags"`
	IsTruncated bool     `xml:"IsTruncated"`
}

type Tags struct {
	XMLName xml.Name `xml:"Tags"`
	Tags    []Tag    `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagUser.html#API_TagUser_Examples
type TagUserResponse struct {
	XMLName          xml.Name         `xml:"TagUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagUser.html#API_UntagUser_Examples
type UntagUserResponse struct {
	XMLName          xml.Name         `xml:"UntagUserResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type CreateAccessKeyResponse struct {
	XMLName          xml.Name              `xml:"CreateAccessKeyResponse"`
	Xmlns            string                `xml:"xmlns,attr"`
	Result           CreateAccessKeyResult `xml:"CreateAccessKeyResult"`
	ResponseMetadata ResponseMetadata      `xml:"ResponseMetadata"`
}

type CreateAccessKeyResult struct {
	XMLName   xml.Name      `xml:"CreateAccessKeyResult"`
	AccessKey UserAccessKey `xml:"AccessKey"`
}

type UserAccessKey struct {
	XMLName         xml.Name                 `xml:"AccessKey"`
	UserName        string                   `xml:"UserName"`
	AccessKeyId     string                   `xml:"AccessKeyId"`
	SecretAccessKey string                   `xml:"SecretAccessKey"`
	Status          s3common.AccessKeyStatus `xml:"Status"`
	CreateDate      string                   `xml:"CreateDate"`
}

type DeleteAccessKeyResponse struct {
	XMLName          xml.Name         `xml:"DeleteAccessKeyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type ListAccessKeysResponse struct {
	XMLName          xml.Name             `xml:"ListAccessKeysResponse"`
	Xmlns            string               `xml:"xmlns,attr"`
	Result           ListAccessKeysResult `xml:"ListAccessKeysResult"`
	ResponseMetadata ResponseMetadata     `xml:"ResponseMetadata"`
}

type ListAccessKeysResult struct {
	XMLName           xml.Name          `xml:"ListAccessKeysResult"`
	UserName          string            `xml:"UserName"`
	AccessKeyMetadata AccessKeyMetadata `xml:"AccessKeyMetadata"`
	IsTruncated       bool              `xml:"IsTruncated"`
}

type AccessKeyMetadata struct {
	XMLName xml.Name          `xml:"AccessKeyMetadata"`
	Member  []memberAccessKey `xml:"member"`
}

type memberAccessKey struct {
	UserName    string                   `xml:"UserName"`
	AccessKeyId string                   `xml:"AccessKeyId"`
	Status      s3common.AccessKeyStatus `xml:"Status"`
	CreateDate  string                   `xml:"CreateDate"`
}

type UpdateAccessKeyResponse struct {
	XMLName          xml.Name         `xml:"UpdateAccessKeyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetAccessKeyLastUsed.html#API_GetAccessKeyLastUsed_Examples
type GetAccessKeyLastUsedResponse struct {
	XMLName                    xml.Name                   `xml:"GetAccessKeyLastUsedResponse"`
	Xmlns                      string                     `xml:"xmlns,attr"`
	GetAccessKeyLastUsedResult GetAccessKeyLastUsedResult `xml:"GetAccessKeyLastUsedResult"`
	ResponseMetadata           ResponseMetadata           `xml:"ResponseMetadata"`
}

type GetAccessKeyLastUsedResult struct {
	XMLName           xml.Name          `xml:"GetAccessKeyLastUsedResult"`
	AccessKeyLastUsed AccessKeyLastUsed `xml:"AccessKeyLastUsed"`
	UserName          string            `xml:"UserName"`
}

type AccessKeyLastUsed struct {
	XMLName      xml.Name `xml:"AccessKeyLastUsed"`
	LastUsedDate *string  `xml:"LastUsedDate"`
	ServiceName  *string  `xml:"ServiceName"`
}

type ResponseMetadata struct {
	XMLName   xml.Name `xml:"ResponseMetadata"`
	RequestId string   `xml:"RequestId"`
}

type IAMActionRequest struct {
	Action  string         `json:"Action"`
	Version string         `json:"Version"`
	Tags    *models.TagMap `json:"Tags"`
	TagKeys []string       `json:"TagKeys"`
	Path    string         `json:"Path"`
	NewPath string         `json:"NewPath"`

	// User Related Fields
	UserName            string `json:"UserName"`
	AccessKeyId         string `json:"AccessKeyId"`
	Status              string `json:"Status"`
	PermissionsBoundary string `json:"PermissionsBoundary"`

	// Group Related Fields
	GroupName string `json:"GroupName"`

	// Policy Related Fields
	Description       string `json:"Description"`
	PolicyArn         string `json:"PolicyArn"`
	PolicyDocument    string `json:"PolicyDocument"`
	PolicyName        string `json:"PolicyName"`
	Marker            string `json:"Marker"`
	MaxItems          *int   `json:"MaxItems"`
	OnlyAttached      bool   `json:"OnlyAttached"`
	PolicyUsageFilter string `json:"PolicyUsageFilter"`
	PathPrefix        string `json:"PathPrefix"`
	SetAsDefault      bool   `json:"SetAsDefault"`
	VersionId         string `json:"VersionId"`
}

func (b *IAMActionRequest) GetMaxItems() int {
	if b.MaxItems != nil && *b.MaxItems < DefaultMaxItems {
		return *b.MaxItems
	}
	return DefaultMaxItems
}

type ErrorResponse struct {
	XMLName    xml.Name             `xml:"Error"`
	Xmlns      string               `xml:"xmlns,attr"`
	Code       s3common.S3ErrorCode `xml:"Code"`
	Message    string               `xml:"Message"`
	BucketName string               `xml:"BucketName,omitempty"`
	RequestID  string               `xml:"RequestId"`
	HostID     string               `xml:"HostId"`
}

type IAMError struct {
	XMLName xml.Name             `xml:"Error"`
	Type    string               `xml:"Type"`
	Message string               `xml:"Message"`
	Code    s3common.S3ErrorCode `xml:"Code"`
}

type IAMErrorResponse struct {
	XMLName   xml.Name `xml:"ErrorResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	Error     IAMError `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type Delete struct {
	XMLName xml.Name            `xml:"Delete"`
	Xmlns   string              `xml:"xmlns,attr"`
	Objects []DeleteObjectInput `xml:"Object"`
	Quiet   bool                `xml:"Quiet,omitempty"`
}

type DeleteObjectInput struct {
	ETag             string `xml:"ETag"`
	Key              string `xml:"Key"`
	LastModifiedTime string `xml:"LastModifiedTime"`
	Size             int64  `xml:"Size"`
	VersionId        string `xml:"VersionId,omitempty"`
}

type DeleteResult struct {
	XMLName xml.Name  `xml:"DeleteResult"`
	Xmlns   string    `xml:"xmlns,attr"`
	Deleted []Deleted `xml:"Deleted"`
}

type Deleted struct {
	XMLName xml.Name `xml:"Deleted"`
	Key     string   `xml:"Key"`
}

type CopyPartResult struct {
	XMLName           xml.Name `xml:"CopyPartResult"`
	Xmlns             string   `xml:"xmlns,attr"`
	ETag              string   `xml:"ETag"`
	LastModified      string   `xml:"LastModified"`
	ChecksumCRC32     string   `xml:"ChecksumCRC32"`
	ChecksumCRC32C    string   `xml:"ChecksumCRC32C"`
	ChecksumCRC64NVME string   `xml:"ChecksumCRC64NVME"`
	ChecksumSHA1      string   `xml:"ChecksumSHA1"`
	ChecksumSHA256    string   `xml:"ChecksumSHA256"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAcl.html#API_GetBucketAcl_ResponseSyntax
type AccessControlPolicy struct {
	XMLName           xml.Name          `xml:"AccessControlPolicy"`
	Xmlns             string            `xml:"xmlns,attr"`
	Owner             *Owner            `xml:"Owner"`
	AccessControlList AccessControlList `xml:"AccessControlList"`
}

type AccessControlList struct {
	XMLName xml.Name `xml:"AccessControlList"`
	Grants  []Grant  `xml:"Grant"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_Grant.html
type Grant struct {
	XMLName    xml.Name `xml:"Grant"`
	Grantee    Grantee  `xml:"Grantee"`
	Permission string   `xml:"Permission"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_Grantee.html
type Grantee struct {
	XMLName      xml.Name `xml:"Grantee"`
	XmlnsXSI     string   `xml:"xmlns:xsi,attr,omitempty"`
	XSIType      string   `xml:"xsi:type,attr,omitempty"`
	DisplayName  string   `xml:"DisplayName,omitempty"`
	EmailAddress string   `xml:"EmailAddress,omitempty"`
	ID           string   `xml:"ID,omitempty"`
	URI          string   `xml:"URI,omitempty"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketOwnershipControls.html#API_GetBucketOwnershipControls_ResponseSyntax
type OwnershipControls struct {
	XMLName xml.Name        `xml:"OwnershipControls"`
	Xmlns   string          `xml:"xmlns,attr"`
	Rules   []OwnershipRule `xml:"Rule"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_OwnershipControlsRule.html
type OwnershipRule struct {
	XMLName         xml.Name `xml:"Rule"`
	ObjectOwnership string   `xml:"ObjectOwnership"`
}

type LifecycleConfig struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration"`
	Xmlns   string          `xml:"xmlns,attr"`
	Rules   []LifeCycleRule `xml:"Rule"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_LifecycleRule.html
type LifeCycleRule struct {
	ID                             string                          `xml:"ID,omitempty"`
	Prefix                         string                          `xml:"Prefix,omitempty"`
	Status                         s3common.LifiCycleStatus        `xml:"Status"` // "Enabled" or "Disabled"
	Filter                         *Filter                         `xml:"Filter,omitempty"`
	Expiration                     *Expiration                     `xml:"Expiration,omitempty"`
	NoncurrentVersionExpiration    *NoncurrentVersionExpiration    `xml:"NoncurrentVersionExpiration,omitempty"`
	AbortIncompleteMultipartUpload *AbortIncompleteMultipartUpload `xml:"AbortIncompleteMultipartUpload,omitempty"`
}

type Filter struct {
	Prefix                string     `xml:"Prefix,omitempty"`
	Tag                   *Tag       `xml:"Tag,omitempty"`
	ObjectSizeGreaterThan *int64     `xml:"ObjectSizeGreaterThan,omitempty"`
	ObjectSizeLessThan    *int64     `xml:"ObjectSizeLessThan,omitempty"`
	And                   *AndFilter `xml:"And,omitempty"`
}

type AndFilter struct {
	Tags                  []Tag  `xml:"Tag"`
	ObjectSizeGreaterThan *int64 `xml:"ObjectSizeGreaterThan,omitempty"`
	ObjectSizeLessThan    *int64 `xml:"ObjectSizeLessThan,omitempty"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_LifecycleExpiration.html
type Expiration struct {
	Date                      *string `xml:"Date,omitempty"`
	Days                      *int32  `xml:"Days,omitempty"`
	ExpiredObjectDeleteMarker *bool   `xml:"ExpiredObjectDeleteMarker,omitempty"`
}

type NoncurrentVersionExpiration struct {
	NewerNoncurrentVersions int32 `xml:"NewerNoncurrentVersions"`
	NoncurrentDays          int32 `xml:"NoncurrentDays"`
}

type AbortIncompleteMultipartUpload struct {
	DaysAfterInitiation int32 `xml:"DaysAfterInitiation"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicy.html#API_CreatePolicy_Examples
type CreatePolicyResponse struct {
	XMLName            xml.Name           `xml:"CreatePolicyResponse"`
	Xmlns              string             `xml:"xmlns,attr"`
	CreatePolicyResult CreatePolicyResult `xml:"CreatePolicyResult"`
	ResponseMetadata   ResponseMetadata   `xml:"ResponseMetadata"`
}

type CreatePolicyResult struct {
	XMLName xml.Name `xml:"CreatePolicyResult"`
	Policy  Policy   `xml:"Policy"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_Policy.html
type Policy struct {
	PolicyName                    string `xml:"PolicyName"`
	DefaultVersionId              string `xml:"DefaultVersionId"`
	PolicyId                      string `xml:"PolicyId"`
	Path                          string `xml:"Path"`
	Arn                           string `xml:"Arn"`
	AttachmentCount               int32  `xml:"AttachmentCount"`
	CreateDate                    string `xml:"CreateDate"`
	UpdateDate                    string `xml:"UpdateDate"`
	IsAttachable                  bool   `xml:"IsAttachable"`
	PermissionsBoundaryUsageCount int32  `xml:"PermissionsBoundaryUsageCount"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicy.html#API_DeletePolicy_Examples
type DeletePolicyResponse struct {
	XMLName          xml.Name         `xml:"DeletePolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicy.html#API_GetPolicy_Examples
type GetPolicyResponse struct {
	XMLName          xml.Name         `xml:"GetPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	GetPolicyResult  GetPolicyResult  `xml:"GetPolicyResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type GetPolicyResult struct {
	XMLName xml.Name `xml:"GetPolicyResult"`
	Policy  Policy   `xml:"Policy"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicies.html#API_ListPolicies_Examples
type ListPoliciesResponse struct {
	XMLName            xml.Name           `xml:"ListPoliciesResponse"`
	Xmlns              string             `xml:"xmlns,attr"`
	ListPoliciesResult ListPoliciesResult `xml:"ListPoliciesResult"`
	ResponseMetadata   ResponseMetadata   `xml:"ResponseMetadata"`
}

type ListPoliciesResult struct {
	XMLName     xml.Name `xml:"ListPoliciesResult"`
	Policies    Policies `xml:"Policies"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker,omitempty"`
}

type Policies struct {
	XMLName xml.Name `xml:"Policies"`
	Member  []Policy `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListEntitiesForPolicy.html#API_ListEntitiesForPolicy_Examples
type ListEntitiesForPolicyResponse struct {
	XMLName                     xml.Name                    `xml:"ListEntitiesForPolicyResponse"`
	Xmlns                       string                      `xml:"xmlns,attr"`
	ListEntitiesForPolicyResult ListEntitiesForPolicyResult `xml:"ListEntitiesForPolicyResult"`
	ResponseMetadata            ResponseMetadata            `xml:"ResponseMetadata"`
}

type ListEntitiesForPolicyResult struct {
	XMLName      xml.Name     `xml:"ListEntitiesForPolicyResult"`
	PolicyGroups PolicyGroups `xml:"PolicyGroups"`
	PolicyUsers  PolicyUsers  `xml:"PolicyUsers"`
	IsTruncated  bool         `xml:"IsTruncated"`
}

type PolicyGroups struct {
	XMLName xml.Name      `xml:"PolicyGroups"`
	Member  []PolicyGroup `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PolicyGroup.html
type PolicyGroup struct {
	GroupName string `xml:"GroupName"`
	GroupId   string `xml:"GroupId"`
}

type PolicyUsers struct {
	XMLName xml.Name     `xml:"PolicyUsers"`
	Member  []PolicyUser `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PolicyUser.html
type PolicyUser struct {
	UserName string `xml:"UserName"`
	UserId   string `xml:"UserId"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserTags.html#API_ListUserTags_Examples
type ListPolicyTagsResponse struct {
	XMLName              xml.Name             `xml:"ListPolicyTagsResponse"`
	Xmlns                string               `xml:"xmlns,attr"`
	ListPolicyTagsResult ListPolicyTagsResult `xml:"ListPolicyTagsResult"`
	ResponseMetadata     ResponseMetadata     `xml:"ResponseMetadata"`
}

type ListPolicyTagsResult struct {
	XMLName     xml.Name `xml:"ListPolicyTagsResult"`
	Tags        Tags     `xml:"Tags"`
	IsTruncated bool     `xml:"IsTruncated"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagPolicy.html#API_TagPolicy_Examples
type TagPolicyResponse struct {
	XMLName          xml.Name         `xml:"TagPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagPolicy.html#API_UntagPolicy_Examples
type UntagPolicyResponse struct {
	XMLName          xml.Name         `xml:"UntagPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicyVersion.html#API_CreatePolicyVersion_Examples
type CreatePolicyVersionResponse struct {
	XMLName                   xml.Name                  `xml:"CreatePolicyVersionResponse"`
	Xmlns                     string                    `xml:"xmlns,attr"`
	CreatePolicyVersionResult CreatePolicyVersionResult `xml:"CreatePolicyVersionResult"`
	ResponseMetadata          ResponseMetadata          `xml:"ResponseMetadata"`
}

type CreatePolicyVersionResult struct {
	XMLName       xml.Name      `xml:"CreatePolicyVersionResult"`
	PolicyVersion PolicyVersion `xml:"PolicyVersion"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PolicyVersion.html
type PolicyVersion struct {
	VersionId        string `xml:"VersionId"`
	IsDefaultVersion bool   `xml:"IsDefaultVersion"`
	CreateDate       string `xml:"CreateDate"`
	Document         string `xml:"Document"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicyVersions.html#API_ListPolicyVersions_Examples
type ListPolicyVersionsResponse struct {
	XMLName                  xml.Name                 `xml:"ListPolicyVersionsResponse"`
	Xmlns                    string                   `xml:"xmlns,attr"`
	ListPolicyVersionsResult ListPolicyVersionsResult `xml:"ListPolicyVersionsResult"`
	ResponseMetadata         ResponseMetadata         `xml:"ResponseMetadata"`
}

type ListPolicyVersionsResult struct {
	XMLName     xml.Name       `xml:"ListPolicyVersionsResult"`
	Versions    PolicyVersions `xml:"Versions"`
	IsTruncated bool           `xml:"IsTruncated"`
	Marker      string         `xml:"Marker,omitempty"`
}

type PolicyVersions struct {
	XMLName xml.Name        `xml:"Versions"`
	Member  []PolicyVersion `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicyVersion.html#API_GetPolicyVersion_Examples
type GetPolicyVersionResponse struct {
	XMLName                xml.Name               `xml:"GetPolicyVersionResponse"`
	Xmlns                  string                 `xml:"xmlns,attr"`
	GetPolicyVersionResult GetPolicyVersionResult `xml:"GetPolicyVersionResult"`
	ResponseMetadata       ResponseMetadata       `xml:"ResponseMetadata"`
}

type GetPolicyVersionResult struct {
	XMLName       xml.Name      `xml:"GetPolicyVersionResult"`
	PolicyVersion PolicyVersion `xml:"PolicyVersion"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicyVersion.html#API_DeletePolicyVersion_Examples
type DeletePolicyVersionResponse struct {
	XMLName          xml.Name         `xml:"DeletePolicyVersionResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_SetDefaultPolicyVersion.html#API_SetDefaultPolicyVersion_Examples
type SetDefaultPolicyVersionResponse struct {
	XMLName          xml.Name         `xml:"SetDefaultPolicyVersionResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateGroup.html#API_CreateGroup_Examples
type CreateGroupResponse struct {
	XMLName           xml.Name          `xml:"CreateGroupResponse"`
	Xmlns             string            `xml:"xmlns,attr"`
	CreateGroupResult CreateGroupResult `xml:"CreateGroupResult"`
	ResponseMetadata  ResponseMetadata  `xml:"ResponseMetadata"`
}

type CreateGroupResult struct {
	XMLName xml.Name `xml:"CreateGroupResult"`
	Group   Group    `xml:"Group"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_Group.html
type Group struct {
	Path       string `xml:"Path"`
	GroupName  string `xml:"GroupName"`
	GroupId    string `xml:"GroupId"`
	Arn        string `xml:"Arn"`
	CreateDate string `xml:"CreateDate"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroups.html#API_ListGroups_Examples
type ListGroupsResponse struct {
	XMLName          xml.Name         `xml:"ListGroupsResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ListGroupsResult ListGroupsResult `xml:"ListGroupsResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type ListGroupsResult struct {
	XMLName     xml.Name `xml:"ListGroupsResult"`
	Groups      Groups   `xml:"Groups"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker,omitempty"`
}

type Groups struct {
	XMLName xml.Name `xml:"Groups"`
	Member  []Group  `xml:"member"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateGroup.html#API_UpdateGroup_Examples
type UpdateGroupResponse struct {
	XMLName          xml.Name         `xml:"UpdateGroupResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AddUserToGroup.html#API_AddUserToGroup_Examples
type AddUserToGroupResponse struct {
	XMLName          xml.Name         `xml:"AddUserToGroupResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroup.html#API_GetGroup_Examples
type GetGroupResponse struct {
	XMLName          xml.Name         `xml:"GetGroupResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	GetGroupResult   GetGroupResult   `xml:"GetGroupResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type GetGroupResult struct {
	XMLName     xml.Name `xml:"GetGroupResult"`
	Group       Group    `xml:"Group"`
	Users       Users    `xml:"Users"`
	IsTruncated bool     `xml:"IsTruncated"`
	Marker      string   `xml:"Marker,omitempty"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_RemoveUserFromGroup.html#API_RemoveUserFromGroup_Examples
type RemoveUserFromGroupResponse struct {
	XMLName          xml.Name         `xml:"RemoveUserFromGroupResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachGroupPolicy.html#API_AttachGroupPolicy_Examples
type AttachGroupPolicyResponse struct {
	XMLName          xml.Name         `xml:"AttachGroupPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroupPolicy.html#API_GetGroupPolicy_Examples
type GetGroupPolicyResponse struct {
	XMLName              xml.Name             `xml:"GetGroupPolicyResponse"`
	Xmlns                string               `xml:"xmlns,attr"`
	GetGroupPolicyResult GetGroupPolicyResult `xml:"GetGroupPolicyResult"`
	ResponseMetadata     ResponseMetadata     `xml:"ResponseMetadata"`
}

type GetGroupPolicyResult struct {
	XMLName        xml.Name `xml:"GetGroupPolicyResult"`
	PolicyName     string   `xml:"PolicyName"`
	GroupName      string   `xml:"GroupName"`
	PolicyDocument string   `xml:"PolicyDocument"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachGroupPolicy.html#API_DetachGroupPolicy_Examples
type DetachGroupPolicyResponse struct {
	XMLName          xml.Name         `xml:"DetachGroupPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedGroupPolicies.html#API_ListAttachedGroupPolicies_Examples
type ListAttachedGroupPoliciesResponse struct {
	XMLName                         xml.Name                        `xml:"ListAttachedGroupPoliciesResponse"`
	Xmlns                           string                          `xml:"xmlns,attr"`
	ListAttachedGroupPoliciesResult ListAttachedGroupPoliciesResult `xml:"ListAttachedGroupPoliciesResult"`
	ResponseMetadata                ResponseMetadata                `xml:"ResponseMetadata"`
}

type ListAttachedGroupPoliciesResult struct {
	XMLName          xml.Name         `xml:"ListAttachedGroupPoliciesResult"`
	AttachedPolicies AttachedPolicies `xml:"AttachedPolicies"`
	IsTruncated      bool             `xml:"IsTruncated"`
	Marker           string           `xml:"Marker,omitempty"`
}

type AttachedPolicies struct {
	XMLName xml.Name         `xml:"AttachedPolicies"`
	Member  []AttachedPolicy `xml:"member"`
}

type AttachedPolicy struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupPolicies.html#API_ListGroupPolicies_Examples
type ListGroupPoliciesResponse struct {
	XMLName                 xml.Name                `xml:"ListGroupPoliciesResponse"`
	Xmlns                   string                  `xml:"xmlns,attr"`
	ListGroupPoliciesResult ListGroupPoliciesResult `xml:"ListGroupPoliciesResult"`
	ResponseMetadata        ResponseMetadata        `xml:"ResponseMetadata"`
}

type ListGroupPoliciesResult struct {
	XMLName     xml.Name    `xml:"ListGroupPoliciesResult"`
	PolicyNames PolicyNames `xml:"PolicyNames"`
	IsTruncated bool        `xml:"IsTruncated"`
	Marker      string      `xml:"Marker,omitempty"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutGroupPolicy.html#API_PutGroupPolicy_Examples
type PutGroupPolicyResponse struct {
	XMLName          xml.Name         `xml:"PutGroupPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroupPolicy.html#API_DeleteGroupPolicy_Examples
type DeleteGroupPolicyResponse struct {
	XMLName          xml.Name         `xml:"DeleteGroupPolicyResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroup.html#API_DeleteGroup_Examples
type DeleteGroupResponse struct {
	XMLName          xml.Name         `xml:"DeleteGroupResponse"`
	Xmlns            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListMultipartUploads.html#API_ListMultipartUploads_ResponseSyntax
type ListMultipartUploadsResult struct {
	XMLName            xml.Name          `xml:"ListMultipartUploadsResult"`
	Xmlns              string            `xml:"xmlns,attr"`
	Bucket             string            `xml:"Bucket"`
	KeyMarker          string            `xml:"KeyMarker,omitempty"`
	UploadIdMarker     string            `xml:"UploadIdMarker,omitempty"`
	NextKeyMarker      string            `xml:"NextKeyMarker,omitempty"`
	Prefix             string            `xml:"Prefix,omitempty"`
	Delimiter          string            `xml:"Delimiter,omitempty"`
	NextUploadIdMarker string            `xml:"NextUploadIdMarker,omitempty"`
	MaxUploads         int32             `xml:"MaxUploads"`
	IsTruncated        bool              `xml:"IsTruncated"`
	Uploads            []MultipartUpload `xml:"Upload"`
	CommonPrefixes     []CommonPrefix    `xml:"CommonPrefixes"`
	EncodingType       string            `xml:"EncodingType,omitempty"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_MultipartUpload.html
type MultipartUpload struct {
	Key          string `xml:"Key"`
	UploadId     string `xml:"UploadId"`
	Initiator    *Owner `xml:"Initiator"`
	Owner        *Owner `xml:"Owner"`
	StorageClass string `xml:"StorageClass"`
	Initiated    string `xml:"Initiated"`
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html#API_ListParts_ResponseSyntax
type ListPartsResult struct {
	XMLName              xml.Name     `xml:"ListPartsResult"`
	Xmlns                string       `xml:"xmlns,attr"`
	Bucket               string       `xml:"Bucket"`
	Key                  string       `xml:"Key"`
	UploadId             string       `xml:"UploadId"`
	PartNumberMarker     int32        `xml:"PartNumberMarker,omitempty"`
	NextPartNumberMarker int32        `xml:"NextPartNumberMarker,omitempty"`
	MaxParts             int32        `xml:"MaxParts"`
	IsTruncated          bool         `xml:"IsTruncated"`
	Parts                []PartResult `xml:"Part"`
}

type PartResult struct {
	Part
	LastModified string `xml:"LastModified"`
	Size         int64  `xml:"Size"`
}
