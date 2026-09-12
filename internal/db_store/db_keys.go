package dbstore

import (
	"fmt"
	"strings"

	"github.com/avatar31/halmidi/internal/core/namespace"
)

const (
	BaseNS     = "namespace:" + namespace.DEFAULT_NAMESPACE
	InternalNS = "namespace:" + namespace.INTERNAL_NAMESPACE

	bucketsDBKey               = BaseNS + ":buckets:"
	bucketEntitesDBKey         = BaseNS + ":bucket:"
	objectEntitesDBKey         = BaseNS + ":object:"
	usersDBKey                 = BaseNS + ":users:"
	userAttachedEntitesDBKey   = BaseNS + ":user:"
	groupsDBKey                = BaseNS + ":groups:"
	groupAttachedEntitesDBKey  = BaseNS + ":group:"
	policiesDBKey              = BaseNS + ":policies:"
	policyAttachedEntitesDBKey = BaseNS + ":policy:"
	accessKeysDBKey            = BaseNS + ":accessKeys:"

	// Format: namespace:default:buckets:<bucket_name>
	//
	// Estimated Key Len:
	// "namespace:default:buckets:" (28) + MaxNameLen (256) = 284
	BucketsDBKey = bucketsDBKey + "%s"

	// Format: namespace:default:bucket:<bucket_name>:spaceUsage
	//
	// Estimated Key Len:
	// "namespace:default:bucket:" (26) + MaxNameLen (256) + ":spaceUsage" (11) = 293
	BucketSpaceUsageDBKey = bucketEntitesDBKey + "%s:spaceUsage"

	// Format: namespace:default:bucket:<bucket_name>:objectCount
	//
	// Estimated Key Len:
	// "namespace:default:bucket:" (26) + MaxNameLen (256) + ":objectCount" (12) = 294
	BucketObjectCountDBKey = bucketEntitesDBKey + "%s:objectCount"

	// Format: namespace:default:bucket:<bucket_name>:objects:<object_key>
	//
	// Estimated Key Len:
	// "namespace:default:bucket:" (26) + MaxNameLen (256) + ":objects:" (9) + MaxObjectKeyLen (1024) = 1315
	ObjectsDBKey = bucketEntitesDBKey + "%s:objects:%s"

	// Format: namespace:default:object:<object_uuid>:versioning:<version_uuid>
	//
	// Estimated Key Len:
	// "namespace:default:object:" (27) + UUIDLen (36) + ":versioning:" (12) + UUIDLen (36) = 111
	ObjectVersioningDBKey = objectEntitesDBKey + "%s:versioning:%s"

	// Format: namespace:default:multipartUploads:<upload_uuid>
	//
	// Estimated Key Len:
	// "namespace:default:multipartUploads:" (34) + UUIDLen (36) = 70
	MultipartUploadDBKey = BaseNS + ":multipartUploads:%s"

	// Format: namespace:default:multipartUpload:<upload_uuid>:part:<part_number>
	//
	// Estimated Key Len:
	// "namespace:default:multipartUpload:" (33) + UUIDLen (36) + ":part:" (6) + MaxPartNumberLen (10) = 85
	MultipartUploadPartDBKey = BaseNS + ":multipartUpload:%s:part:%s"

	// Format: namespace:default:usersDBKey:<user_name>
	//
	// Estimated Key Len:
	// "namespace:default:usersDBKey:" (26) + MaxNameLen (64) = 90
	UsersDBKey = usersDBKey + "%s"

	// Format: namespace:default:user:<user_name>:policies:<policy_name>
	//
	// Estimated Key Len:
	// "namespace:default:user:" (25) + MaxNameLen (64) + ":inlinePolicies:" (16) + MaxNameLen (128) = 233
	UserInlinePoliciesDBKey = userAttachedEntitesDBKey + "%s:inlinePolicies:%s"

	// Format: namespace:default:user:<user_name>:attachedGroups:<group_name>
	//
	// Estimated Key Len:
	// "namespace:default:user:" (25) + MaxNameLen (64) + ":attachedGroups:" (16) + MaxNameLen (128) = 233
	UserAttachedGroupsDBKey = userAttachedEntitesDBKey + "%s:attachedGroups:%s"

	// Format: namespace:default:user:<user_name>:attachedPolicies:<policy_name>
	//
	// Estimated Key Len:
	// "namespace:default:user:" (25) + MaxNameLen (64) + ":attachedPolicies:" (18) + MaxNameLen (128) = 235
	UserAttachedPoliciesDBKey = userAttachedEntitesDBKey + "%s:attachedPolicies:%s"

	// Format: namespace:default:user:<user_name>:accessKeys:<access_key>
	//
	// Estimated Key Len:
	// "namespace:default:user:" (25) + MaxNameLen (64) + ":accessKeys:" (11) + AccessKeyLen (20) = 120
	UserAccessKeysDBKey = userAttachedEntitesDBKey + "%s:accessKeys:%s"

	// Format: namespace:default:groupsDBKey:<group_name>
	//
	// Estimated Key Len:
	// "namespace:default:groupsDBKey:" (27) + MaxNameLen (128) = 155
	GroupsDBKey = groupsDBKey + "%s"

	// Format: namespace:default:group:<group_name>:attachedUsers:<user_name>
	//
	// Estimated Key Len:
	// "namespace:default:group:" (26) + MaxNameLen (128) + ":attachedUsers:" (15) + MaxNameLen (64) = 233
	GroupAttachedUsersDBKey = groupAttachedEntitesDBKey + "%s:attachedUsers:%s"

	// Format: namespace:default:group:<group_name>:policies:<policy_name>
	//
	// Estimated Key Len:
	// "namespace:default:group:" (26) + MaxNameLen (128) + ":inlinePolicies:" (16) + MaxNameLen (128) = 298
	GroupInlinePoliciesDBKey = groupAttachedEntitesDBKey + "%s:inlinePolicies:%s"

	// Format: namespace:default:group:<group_name>:attachedPolicies:<policy_name>
	//
	// Estimated Key Len:
	// "namespace:default:group:" (26) + MaxNameLen (128) + ":attachedPolicies:" (18) + MaxNameLen (128) = 300
	GroupAttachedPoliciesDBKey = groupAttachedEntitesDBKey + "%s:attachedPolicies:%s"

	// Format: namespace:default:policiesDBKey:<policy_path>
	//
	// Estimated Key Len:
	// "namespace:default:policiesDBKey:" (27) + MaxPathLen (512) = 539
	PoliciesDBKey = policiesDBKey + "%s"

	// Format: namespace:default:policy:<policy_path>:attachedUsers:<user_name>
	//
	// Estimated Key Len:
	// "namespace:default:policy:" (27) + MaxPathLen (512) + ":attachedUsers:" (15) + MaxNameLen (64) = 618
	PolicyAttachedUsersDBKey = policyAttachedEntitesDBKey + "%s:attachedUsers:%s"

	// Format: namespace:default:policy:<policy_path>:attachedUsers:<group_name>
	//
	// Estimated Key Len:
	// "namespace:default:policy:" (27) + MaxPathLen (512) + ":attachedGroups:" (16) + MaxNameLen (128) = 683
	PolicyAttachedGroupsDBKey = policyAttachedEntitesDBKey + "%s:attachedGroups:%s"

	// Format: namespace:default:policy:<policy_uuid>:versions:v<version_number>
	//
	// Estimated Key Len:
	// "namespace:default:policy:" (27) + UUIDLen (36) + ":versions:v" (11) + MaxVersionNumberLen (10) = 84
	PolicyVersionDBKey = policyAttachedEntitesDBKey + "%s:versions:%s"

	// Format: namespace:default:userAccessKeys:<access_key>
	//
	// Estimated Key Len:
	// "namespace:default:accessKeys:" (29) + AccessKeyLen (20) = 49
	AccessKeysDBKey = accessKeysDBKey + "%s"

	// Format: namespace:default:userAccessKeys:<access_key>:lastUsed
	//
	// Estimated Key Len:
	// "namespace:default:userAccessKeys:" (34) + AccessKeyLen (20) + ":lastUsed:" (9) = 63
	UserAccessKeysLastUsedDBKey = BaseNS + ":userAccessKeys:%s:lastUsed"

	// namespace:internal:queue:<queue_name>
	InternalQueueDBKey = InternalNS + ":queue:%s"
)

func GetBucketDBKeyWithNS(bucket string) string {
	return fmt.Sprintf(BucketsDBKey, bucket)
}

func GetBucketSpaceUsageDBKeyWithNS(bucket string) string {
	return fmt.Sprintf(BucketSpaceUsageDBKey, bucket)
}

func GetBucketObjectCountDBKeyWithNS(bucket string) string {
	return fmt.Sprintf(BucketObjectCountDBKey, bucket)
}

func GetUserDBKeyWithNS(username string) string {
	return fmt.Sprintf(UsersDBKey, strings.ToLower(username))
}

func GetUserAttachedGroupDBKeyWithNS(username, groupname string) string {
	return fmt.Sprintf(UserAttachedGroupsDBKey, strings.ToLower(username), strings.ToLower(groupname))
}

func GetUserAttachedPolicyDBKeyWithNS(username, policyPath string) string {
	return fmt.Sprintf(UserAttachedPoliciesDBKey, strings.ToLower(username), strings.ToLower(policyPath))
}

func GetUserInlinePolicyDBKeyWithNS(username, policyname string) string {
	return fmt.Sprintf(UserInlinePoliciesDBKey, strings.ToLower(username), strings.ToLower(policyname))
}

func GetUserAccessKeyRefDBKeyWithNS(username, accessKey string) string {
	return fmt.Sprintf(UserAccessKeysDBKey, strings.ToLower(username), accessKey)
}

func GetGroupDBKeyWithNS(groupname string) string {
	return fmt.Sprintf(GroupsDBKey, strings.ToLower(groupname))
}

func GetGroupAttachedUserDBKeyWithNS(groupname, username string) string {
	return fmt.Sprintf(GroupAttachedUsersDBKey, strings.ToLower(groupname), strings.ToLower(username))
}

func GetGroupAttachedPolicyDBKeyWithNS(groupname, policyPath string) string {
	return fmt.Sprintf(GroupAttachedPoliciesDBKey, strings.ToLower(groupname), strings.ToLower(policyPath))
}

func GetGroupInlinePolicyDBKeyWithNS(groupname, policyname string) string {
	return fmt.Sprintf(GroupInlinePoliciesDBKey, strings.ToLower(groupname), strings.ToLower(policyname))
}

func GetPolicyDBKeyWithNS(path string) string {
	return fmt.Sprintf(PoliciesDBKey, strings.ToLower(path))
}

func GetPolicyVersionDBKeyWithNS(policyUUID, versionID string) string {
	return fmt.Sprintf(PolicyVersionDBKey, policyUUID, versionID)
}

func GetPolicyAttachedUserDBKeyWithNS(path, username string) string {
	return fmt.Sprintf(PolicyAttachedUsersDBKey, path, strings.ToLower(username))
}

func GetPolicyAttachedGroupDBKeyWithNS(path, groupname string) string {
	return fmt.Sprintf(PolicyAttachedGroupsDBKey, path, strings.ToLower(groupname))
}

func GetAccessKeyDBKeyWithNS(accessKey string) string {
	return fmt.Sprintf(AccessKeysDBKey, accessKey)
}

func GetUserAccessKeyLastUsedDBKeyWithNS(accessKey string) string {
	return fmt.Sprintf(UserAccessKeysLastUsedDBKey, accessKey)
}

func GetObjectDBKeyWithNS(bucket, objectKey string) string {
	return fmt.Sprintf(ObjectsDBKey, bucket, objectKey)
}

func GetObjectVersioningDBKeyWithNS(objectUUID, versionUUID string) string {
	return fmt.Sprintf(ObjectVersioningDBKey, objectUUID, versionUUID)
}

func GetMultipartUploadDBKeyWithNS(uploadUUID string) string {
	return fmt.Sprintf(MultipartUploadDBKey, uploadUUID)
}

func GetMultipartUploadPartDBKeyWithNS(uploadUUID, part string) string {
	return fmt.Sprintf(MultipartUploadPartDBKey, uploadUUID, part)
}

func GetSystemQueueDBKeyWithNS(name string) string {
	return fmt.Sprintf(InternalQueueDBKey, name)
}
