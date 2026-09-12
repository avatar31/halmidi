package s3common

// Supported Actions
type PolicyAction string

const (
	AllActionsOrResourcePattern = "*"
	AllS3ActionsPattern         = "s3:*"
	AllIAMActionsPattern        = "iam:*"

	// S3 - https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html#amazons3-actions-as-permissions

	// Access Grants & Identity Center
	PolicyActionS3CreateAccessGrant                        PolicyAction = "s3:CreateAccessGrant"
	PolicyActionS3GetAccessGrant                           PolicyAction = "s3:GetAccessGrant"
	PolicyActionS3DeleteAccessGrant                        PolicyAction = "s3:DeleteAccessGrant"
	PolicyActionS3CreateAccessGrantsInstance               PolicyAction = "s3:CreateAccessGrantsInstance"
	PolicyActionS3GetAccessGrantsInstance                  PolicyAction = "s3:GetAccessGrantsInstance"
	PolicyActionS3GetAccessGrantsInstanceForPrefix         PolicyAction = "s3:GetAccessGrantsInstanceForPrefix"
	PolicyActionS3GetAccessGrantsInstanceResourcePolicy    PolicyAction = "s3:GetAccessGrantsInstanceResourcePolicy"
	PolicyActionS3DeleteAccessGrantsInstance               PolicyAction = "s3:DeleteAccessGrantsInstance"
	PolicyActionS3DeleteAccessGrantsInstanceResourcePolicy PolicyAction = "s3:DeleteAccessGrantsInstanceResourcePolicy"
	PolicyActionS3CreateAccessGrantsLocation               PolicyAction = "s3:CreateAccessGrantsLocation"
	PolicyActionS3GetAccessGrantsLocation                  PolicyAction = "s3:GetAccessGrantsLocation"
	PolicyActionS3DeleteAccessGrantsLocation               PolicyAction = "s3:DeleteAccessGrantsLocation"
	PolicyActionS3AssociateAccessGrantsIdentityCenter      PolicyAction = "s3:AssociateAccessGrantsIdentityCenter"
	PolicyActionS3DissociateAccessGrantsIdentityCenter     PolicyAction = "s3:DissociateAccessGrantsIdentityCenter"

	// Access Points & Multi-Region Access Points (MRAP)
	PolicyActionS3CreateAccessPoint                          PolicyAction = "s3:CreateAccessPoint"
	PolicyActionS3GetAccessPoint                             PolicyAction = "s3:GetAccessPoint"
	PolicyActionS3DeleteAccessPoint                          PolicyAction = "s3:DeleteAccessPoint"
	PolicyActionS3PutAccessPointPolicy                       PolicyAction = "s3:PutAccessPointPolicy"
	PolicyActionS3CreateAccessPointForObjectLambda           PolicyAction = "s3:CreateAccessPointForObjectLambda"
	PolicyActionS3GetAccessPointConfigurationForObjectLambda PolicyAction = "s3:GetAccessPointConfigurationForObjectLambda"
	PolicyActionS3DeleteAccessPointForObjectLambda           PolicyAction = "s3:DeleteAccessPointForObjectLambda"
	PolicyActionS3DeleteAccessPointPolicy                    PolicyAction = "s3:DeleteAccessPointPolicy"
	PolicyActionS3DeleteAccessPointPolicyForObjectLambda     PolicyAction = "s3:DeleteAccessPointPolicyForObjectLambda"
	PolicyActionS3CreateMultiRegionAccessPoint               PolicyAction = "s3:CreateMultiRegionAccessPoint"
	PolicyActionS3DeleteMultiRegionAccessPoint               PolicyAction = "s3:DeleteMultiRegionAccessPoint"
	PolicyActionS3DescribeMultiRegionAccessPointOperation    PolicyAction = "s3:DescribeMultiRegionAccessPointOperation"

	// Bucket lifecycle, configuration, metadata & management
	PolicyActionS3CreateBucket                           PolicyAction = "s3:CreateBucket"
	PolicyActionS3DeleteBucket                           PolicyAction = "s3:DeleteBucket"
	PolicyActionS3CreateBucketMetadataTableConfiguration PolicyAction = "s3:CreateBucketMetadataTableConfiguration"
	PolicyActionS3DeleteBucketMetadataTableConfiguration PolicyAction = "s3:DeleteBucketMetadataTableConfiguration"
	PolicyActionS3GetBucketLocation                      PolicyAction = "s3:GetBucketLocation"
	PolicyActionS3ListAllMyBuckets                       PolicyAction = "s3:ListAllMyBuckets"
	PolicyActionS3ListBucket                             PolicyAction = "s3:ListBucket"
	PolicyActionS3ListBucketVersions                     PolicyAction = "s3:ListBucketVersions"
	PolicyActionS3GetBucketAcl                           PolicyAction = "s3:GetBucketAcl"
	PolicyActionS3PutBucketAcl                           PolicyAction = "s3:PutBucketAcl"
	PolicyActionS3GetBucketCors                          PolicyAction = "s3:GetBucketCors"
	PolicyActionS3PutBucketCors                          PolicyAction = "s3:PutBucketCors"
	PolicyActionS3DeleteBucketCors                       PolicyAction = "s3:DeleteBucketCors"
	PolicyActionS3GetBucketLogging                       PolicyAction = "s3:GetBucketLogging"
	PolicyActionS3PutBucketLogging                       PolicyAction = "s3:PutBucketLogging"
	PolicyActionS3GetBucketNotification                  PolicyAction = "s3:GetBucketNotification"
	PolicyActionS3PutBucketNotification                  PolicyAction = "s3:PutBucketNotification"
	PolicyActionS3GetBucketNotificationConfiguration     PolicyAction = "s3:GetBucketNotificationConfiguration"
	PolicyActionS3PutBucketNotificationConfiguration     PolicyAction = "s3:PutBucketNotificationConfiguration"
	PolicyActionS3PutBucketNotificationFilter            PolicyAction = "s3:PutBucketNotificationFilter"
	PolicyActionS3GetBucketPolicy                        PolicyAction = "s3:GetBucketPolicy"
	PolicyActionS3PutBucketPolicy                        PolicyAction = "s3:PutBucketPolicy"
	PolicyActionS3GetBucketPolicyStatus                  PolicyAction = "s3:GetBucketPolicyStatus"
	PolicyActionS3DeleteBucketPolicy                     PolicyAction = "s3:DeleteBucketPolicy"
	PolicyActionS3GetBucketRequestPayment                PolicyAction = "s3:GetBucketRequestPayment"
	PolicyActionS3PutBucketRequestPayment                PolicyAction = "s3:PutBucketRequestPayment"
	PolicyActionS3GetBucketTagging                       PolicyAction = "s3:GetBucketTagging"
	PolicyActionS3PutBucketTagging                       PolicyAction = "s3:PutBucketTagging"
	PolicyActionS3DeleteBucketTagging                    PolicyAction = "s3:DeleteBucketTagging"
	PolicyActionS3GetBucketVersioning                    PolicyAction = "s3:GetBucketVersioning"
	PolicyActionS3PutBucketVersioning                    PolicyAction = "s3:PutBucketVersioning"
	PolicyActionS3GetBucketWebsite                       PolicyAction = "s3:GetBucketWebsite"
	PolicyActionS3PutBucketWebsite                       PolicyAction = "s3:PutBucketWebsite"
	PolicyActionS3DeleteBucketWebsite                    PolicyAction = "s3:DeleteBucketWebsite"

	// Lifecycle configuration (legacy and unified)
	PolicyActionS3GetLifecycleConfiguration    PolicyAction = "s3:GetLifecycleConfiguration"
	PolicyActionS3PutLifecycleConfiguration    PolicyAction = "s3:PutLifecycleConfiguration"
	PolicyActionS3DeleteLifecycleConfiguration PolicyAction = "s3:DeleteLifecycleConfiguration"
	PolicyActionS3GetBucketLifecycle           PolicyAction = "s3:GetBucketLifecycle"
	PolicyActionS3PutBucketLifecycle           PolicyAction = "s3:PutBucketLifecycle"

	// Encryption, replication, object-lock & retention
	PolicyActionS3GetBucketEncryption              PolicyAction = "s3:GetBucketEncryption"
	PolicyActionS3PutBucketEncryption              PolicyAction = "s3:PutBucketEncryption"
	PolicyActionS3GetReplicationConfiguration      PolicyAction = "s3:GetReplicationConfiguration"
	PolicyActionS3PutReplicationConfiguration      PolicyAction = "s3:PutReplicationConfiguration"
	PolicyActionS3DeleteReplicationConfiguration   PolicyAction = "s3:DeleteReplicationConfiguration"
	PolicyActionS3ReplicateObject                  PolicyAction = "s3:ReplicateObject"
	PolicyActionS3ReplicateDelete                  PolicyAction = "s3:ReplicateDelete"
	PolicyActionS3ReplicateTags                    PolicyAction = "s3:ReplicateTags"
	PolicyActionS3GetObjectLockConfiguration       PolicyAction = "s3:GetObjectLockConfiguration"
	PolicyActionS3PutObjectLockConfiguration       PolicyAction = "s3:PutObjectLockConfiguration"
	PolicyActionS3GetBucketObjectLockConfiguration PolicyAction = "s3:GetBucketObjectLockConfiguration"
	PolicyActionS3GetObjectRetention               PolicyAction = "s3:GetObjectRetention"
	PolicyActionS3PutObjectRetention               PolicyAction = "s3:PutObjectRetention"
	PolicyActionS3GetObjectLegalHold               PolicyAction = "s3:GetObjectLegalHold"
	PolicyActionS3PutObjectLegalHold               PolicyAction = "s3:PutObjectLegalHold"
	PolicyActionS3BypassGovernanceRetention        PolicyAction = "s3:BypassGovernanceRetention"

	// Object operations (GET/PUT/DELETE, versions, ACLs, tagging, restore, select)
	PolicyActionS3GetObject                  PolicyAction = "s3:GetObject"
	PolicyActionS3PutObject                  PolicyAction = "s3:PutObject"
	PolicyActionS3DeleteObject               PolicyAction = "s3:DeleteObject"
	PolicyActionS3GetObjectAcl               PolicyAction = "s3:GetObjectAcl"
	PolicyActionS3PutObjectAcl               PolicyAction = "s3:PutObjectAcl"
	PolicyActionS3GetObjectTagging           PolicyAction = "s3:GetObjectTagging"
	PolicyActionS3PutObjectTagging           PolicyAction = "s3:PutObjectTagging"
	PolicyActionS3DeleteObjectTagging        PolicyAction = "s3:DeleteObjectTagging"
	PolicyActionS3GetObjectVersion           PolicyAction = "s3:GetObjectVersion"
	PolicyActionS3GetObjectVersionAcl        PolicyAction = "s3:GetObjectVersionAcl"
	PolicyActionS3PutObjectVersionAcl        PolicyAction = "s3:PutObjectVersionAcl"
	PolicyActionS3GetObjectVersionTagging    PolicyAction = "s3:GetObjectVersionTagging"
	PolicyActionS3PutObjectVersionTagging    PolicyAction = "s3:PutObjectVersionTagging"
	PolicyActionS3DeleteObjectVersion        PolicyAction = "s3:DeleteObjectVersion"
	PolicyActionS3DeleteObjectVersionTagging PolicyAction = "s3:DeleteObjectVersionTagging"
	PolicyActionS3RestoreObject              PolicyAction = "s3:RestoreObject"
	PolicyActionS3SelectObjectContent        PolicyAction = "s3:SelectObjectContent"
	PolicyActionS3GetObjectTorrent           PolicyAction = "s3:GetObjectTorrent"

	// Multipart upload operations
	PolicyActionS3CreateMultipartUpload      PolicyAction = "s3:CreateMultipartUpload"
	PolicyActionS3AbortMultipartUpload       PolicyAction = "s3:AbortMultipartUpload"
	PolicyActionS3ListMultipartUploadParts   PolicyAction = "s3:ListMultipartUploadParts"
	PolicyActionS3CompleteMultipartUpload    PolicyAction = "s3:CompleteMultipartUpload"
	PolicyActionS3ListBucketMultipartUploads PolicyAction = "s3:ListBucketMultipartUploads"

	// Jobs, tagging and related job operations
	PolicyActionS3CreateJob        PolicyAction = "s3:CreateJob"
	PolicyActionS3ListJobs         PolicyAction = "s3:ListJobs"
	PolicyActionS3DescribeJob      PolicyAction = "s3:DescribeJob"
	PolicyActionS3UpdateJob        PolicyAction = "s3:UpdateJob"
	PolicyActionS3DeleteJobTagging PolicyAction = "s3:DeleteJobTagging"

	// Metrics, Analytics & Storage Lens
	PolicyActionS3GetBucketMetricsConfiguration         PolicyAction = "s3:GetBucketMetricsConfiguration"
	PolicyActionS3PutBucketMetricsConfiguration         PolicyAction = "s3:PutBucketMetricsConfiguration"
	PolicyActionS3DeleteBucketMetricsConfiguration      PolicyAction = "s3:DeleteBucketMetricsConfiguration"
	PolicyActionS3ListStorageLensConfigurations         PolicyAction = "s3:ListStorageLensConfigurations"
	PolicyActionS3GetStorageLensConfiguration           PolicyAction = "s3:GetStorageLensConfiguration"
	PolicyActionS3PutStorageLensConfiguration           PolicyAction = "s3:PutStorageLensConfiguration"
	PolicyActionS3DeleteStorageLensConfiguration        PolicyAction = "s3:DeleteStorageLensConfiguration"
	PolicyActionS3DeleteStorageLensConfigurationTagging PolicyAction = "s3:DeleteStorageLensConfigurationTagging"
	PolicyActionS3CreateStorageLensGroup                PolicyAction = "s3:CreateStorageLensGroup"
	PolicyActionS3DeleteStorageLensGroup                PolicyAction = "s3:DeleteStorageLensGroup"

	// Acceleration and other misc features
	PolicyActionS3GetAccelerateConfiguration PolicyAction = "s3:GetAccelerateConfiguration"
	PolicyActionS3PutAccelerateConfiguration PolicyAction = "s3:PutAccelerateConfiguration"

	// IAM - https://docs.aws.amazon.com/service-authorization/latest/reference/list_awsidentityandaccessmanagementiam.html#awsidentityandaccessmanagementiam-actions-as-permissions

	// Users
	PolicyActionIAMCreateUser               PolicyAction = "iam:CreateUser"
	PolicyActionIAMDeleteUser               PolicyAction = "iam:DeleteUser"
	PolicyActionIAMGetUser                  PolicyAction = "iam:GetUser"
	PolicyActionIAMUpdateUser               PolicyAction = "iam:UpdateUser"
	PolicyActionIAMListUsers                PolicyAction = "iam:ListUsers"
	PolicyActionIAMCreateLoginProfile       PolicyAction = "iam:CreateLoginProfile"
	PolicyActionIAMDeleteLoginProfile       PolicyAction = "iam:DeleteLoginProfile"
	PolicyActionIAMGetLoginProfile          PolicyAction = "iam:GetLoginProfile"
	PolicyActionIAMUpdateLoginProfile       PolicyAction = "iam:UpdateLoginProfile"
	PolicyActionIAMChangePassword           PolicyAction = "iam:ChangePassword"
	PolicyActionIAMGetUserPolicy            PolicyAction = "iam:GetUserPolicy"
	PolicyActionIAMListUserPolicies         PolicyAction = "iam:ListUserPolicies"
	PolicyActionIAMPutUserPolicy            PolicyAction = "iam:PutUserPolicy"
	PolicyActionIAMDeleteUserPolicy         PolicyAction = "iam:DeleteUserPolicy"
	PolicyActionIAMListAttachedUserPolicies PolicyAction = "iam:ListAttachedUserPolicies"
	PolicyActionIAMTagUser                  PolicyAction = "iam:TagUser"
	PolicyActionIAMUntagUser                PolicyAction = "iam:UntagUser"
	PolicyActionIAMListUserTags             PolicyAction = "iam:ListUserTags"

	// Access Keys & Credentials
	PolicyActionIAMCreateAccessKey          PolicyAction = "iam:CreateAccessKey"
	PolicyActionIAMDeleteAccessKey          PolicyAction = "iam:DeleteAccessKey"
	PolicyActionIAMUpdateAccessKey          PolicyAction = "iam:UpdateAccessKey"
	PolicyActionIAMListAccessKeys           PolicyAction = "iam:ListAccessKeys"
	PolicyActionIAMGetAccessKeyLastUsed     PolicyAction = "iam:GetAccessKeyLastUsed"
	PolicyActionIAMGenerateCredentialReport PolicyAction = "iam:GenerateCredentialReport"
	PolicyActionIAMGetCredentialReport      PolicyAction = "iam:GetCredentialReport"

	// Groups
	PolicyActionIAMCreateGroup               PolicyAction = "iam:CreateGroup"
	PolicyActionIAMUpdateGroup               PolicyAction = "iam:UpdateGroup"
	PolicyActionIAMDeleteGroup               PolicyAction = "iam:DeleteGroup"
	PolicyActionIAMGetGroup                  PolicyAction = "iam:GetGroup"
	PolicyActionIAMListGroups                PolicyAction = "iam:ListGroups"
	PolicyActionIAMAddUserToGroup            PolicyAction = "iam:AddUserToGroup"
	PolicyActionIAMRemoveUserFromGroup       PolicyAction = "iam:RemoveUserFromGroup"
	PolicyActionIAMPutGroupPolicy            PolicyAction = "iam:PutGroupPolicy"
	PolicyActionIAMDeleteGroupPolicy         PolicyAction = "iam:DeleteGroupPolicy"
	PolicyActionIAMGetGroupPolicy            PolicyAction = "iam:GetGroupPolicy"
	PolicyActionIAMListGroupPolicies         PolicyAction = "iam:ListGroupPolicies"
	PolicyActionIAMAttachGroupPolicy         PolicyAction = "iam:AttachGroupPolicy"
	PolicyActionIAMDetachGroupPolicy         PolicyAction = "iam:DetachGroupPolicy"
	PolicyActionIAMTagGroup                  PolicyAction = "iam:TagGroup"
	PolicyActionIAMUntagGroup                PolicyAction = "iam:UntagGroup"
	PolicyActionIAMListAttachedGroupPolicies PolicyAction = "iam:ListAttachedGroupPolicies"
	PolicyActionIAMListGroupsForUser         PolicyAction = "iam:ListGroupsForUser"

	// Roles & Instance Profiles
	PolicyActionIAMCreateRole                    PolicyAction = "iam:CreateRole"
	PolicyActionIAMDeleteRole                    PolicyAction = "iam:DeleteRole"
	PolicyActionIAMGetRole                       PolicyAction = "iam:GetRole"
	PolicyActionIAMUpdateRole                    PolicyAction = "iam:UpdateRole"
	PolicyActionIAMUpdateAssumeRolePolicy        PolicyAction = "iam:UpdateAssumeRolePolicy"
	PolicyActionIAMAttachRolePolicy              PolicyAction = "iam:AttachRolePolicy"
	PolicyActionIAMDetachRolePolicy              PolicyAction = "iam:DetachRolePolicy"
	PolicyActionIAMPutRolePolicy                 PolicyAction = "iam:PutRolePolicy"
	PolicyActionIAMDeleteRolePolicy              PolicyAction = "iam:DeleteRolePolicy"
	PolicyActionIAMGetRolePolicy                 PolicyAction = "iam:GetRolePolicy"
	PolicyActionIAMListRolePolicies              PolicyAction = "iam:ListRolePolicies"
	PolicyActionIAMPassRole                      PolicyAction = "iam:PassRole"
	PolicyActionIAMCreateInstanceProfile         PolicyAction = "iam:CreateInstanceProfile"
	PolicyActionIAMDeleteInstanceProfile         PolicyAction = "iam:DeleteInstanceProfile"
	PolicyActionIAMGetInstanceProfile            PolicyAction = "iam:GetInstanceProfile"
	PolicyActionIAMAddRoleToInstanceProfile      PolicyAction = "iam:AddRoleToInstanceProfile"
	PolicyActionIAMRemoveRoleFromInstanceProfile PolicyAction = "iam:RemoveRoleFromInstanceProfile"
	PolicyActionIAMListInstanceProfiles          PolicyAction = "iam:ListInstanceProfiles"
	PolicyActionIAMListInstanceProfilesForRole   PolicyAction = "iam:ListInstanceProfilesForRole"
	PolicyActionIAMTagRole                       PolicyAction = "iam:TagRole"
	PolicyActionIAMUntagRole                     PolicyAction = "iam:UntagRole"

	// Policies & Policy Versions
	PolicyActionIAMCreatePolicy                      PolicyAction = "iam:CreatePolicy"
	PolicyActionIAMDeletePolicy                      PolicyAction = "iam:DeletePolicy"
	PolicyActionIAMGetPolicy                         PolicyAction = "iam:GetPolicy"
	PolicyActionIAMListPolicies                      PolicyAction = "iam:ListPolicies"
	PolicyActionIAMListPoliciesGrantingServiceAccess PolicyAction = "iam:ListPoliciesGrantingServiceAccess"
	PolicyActionIAMCreatePolicyVersion               PolicyAction = "iam:CreatePolicyVersion"
	PolicyActionIAMDeletePolicyVersion               PolicyAction = "iam:DeletePolicyVersion"
	PolicyActionIAMGetPolicyVersion                  PolicyAction = "iam:GetPolicyVersion"
	PolicyActionIAMListPolicyVersions                PolicyAction = "iam:ListPolicyVersions"
	PolicyActionIAMSetDefaultPolicyVersion           PolicyAction = "iam:SetDefaultPolicyVersion"
	PolicyActionIAMAttachUserPolicy                  PolicyAction = "iam:AttachUserPolicy"
	PolicyActionIAMDetachUserPolicy                  PolicyAction = "iam:DetachUserPolicy"
	PolicyActionIAMTagPolicy                         PolicyAction = "iam:TagPolicy"
	PolicyActionIAMUntagPolicy                       PolicyAction = "iam:UntagPolicy"
	PolicyActionIAMListEntitiesForPolicy             PolicyAction = "iam:ListEntitiesForPolicy"
	PolicyActionIAMListPolicyTags                    PolicyAction = "iam:ListPolicyTags"

	// Permissions Boundaries
	PolicyActionIAMPutUserPermissionsBoundary    PolicyAction = "iam:PutUserPermissionsBoundary"
	PolicyActionIAMDeleteUserPermissionsBoundary PolicyAction = "iam:DeleteUserPermissionsBoundary"
	PolicyActionIAMPutRolePermissionsBoundary    PolicyAction = "iam:PutRolePermissionsBoundary"
	PolicyActionIAMDeleteRolePermissionsBoundary PolicyAction = "iam:DeleteRolePermissionsBoundary"

	// MFA & Virtual MFA
	PolicyActionIAMCreateVirtualMFADevice PolicyAction = "iam:CreateVirtualMFADevice"
	PolicyActionIAMDeleteVirtualMFADevice PolicyAction = "iam:DeleteVirtualMFADevice"
	PolicyActionIAMEnableMFADevice        PolicyAction = "iam:EnableMFADevice"
	PolicyActionIAMDisableMFADevice       PolicyAction = "iam:DeactivateMFADevice"
	PolicyActionIAMResyncMFADevice        PolicyAction = "iam:ResyncMFADevice"
	PolicyActionIAMListMFADevices         PolicyAction = "iam:ListMFADevices"
	PolicyActionIAMListVirtualMFADevices  PolicyAction = "iam:ListVirtualMFADevices"

	// OpenID Connect (OIDC)
	PolicyActionIAMCreateOpenIDConnectProvider             PolicyAction = "iam:CreateOpenIDConnectProvider"
	PolicyActionIAMDeleteOpenIDConnectProvider             PolicyAction = "iam:DeleteOpenIDConnectProvider"
	PolicyActionIAMGetOpenIDConnectProvider                PolicyAction = "iam:GetOpenIDConnectProvider"
	PolicyActionIAMListOpenIDConnectProviders              PolicyAction = "iam:ListOpenIDConnectProviders"
	PolicyActionIAMAddClientIDToOpenIDConnectProvider      PolicyAction = "iam:AddClientIDToOpenIDConnectProvider"
	PolicyActionIAMRemoveClientIDFromOpenIDConnectProvider PolicyAction = "iam:RemoveClientIDFromOpenIDConnectProvider"
	PolicyActionIAMUpdateOpenIDConnectProviderThumbprint   PolicyAction = "iam:UpdateOpenIDConnectProviderThumbprint"

	// SAML
	PolicyActionIAMCreateSAMLProvider PolicyAction = "iam:CreateSAMLProvider"
	PolicyActionIAMDeleteSAMLProvider PolicyAction = "iam:DeleteSAMLProvider"
	PolicyActionIAMGetSAMLProvider    PolicyAction = "iam:GetSAMLProvider"
	PolicyActionIAMListSAMLProviders  PolicyAction = "iam:ListSAMLProviders"
	PolicyActionIAMUpdateSAMLProvider PolicyAction = "iam:UpdateSAMLProvider"

	// Server Certificates & Signing Certificates
	PolicyActionIAMUploadServerCertificate PolicyAction = "iam:UploadServerCertificate"
	PolicyActionIAMDeleteServerCertificate PolicyAction = "iam:DeleteServerCertificate"
	PolicyActionIAMGetServerCertificate    PolicyAction = "iam:GetServerCertificate"
	PolicyActionIAMListServerCertificates  PolicyAction = "iam:ListServerCertificates"
	PolicyActionIAMUpdateServerCertificate PolicyAction = "iam:UpdateServerCertificate"

	PolicyActionIAMUploadSigningCertificate PolicyAction = "iam:UploadSigningCertificate"
	PolicyActionIAMDeleteSigningCertificate PolicyAction = "iam:DeleteSigningCertificate"
	PolicyActionIAMListSigningCertificates  PolicyAction = "iam:ListSigningCertificates"
	PolicyActionIAMUpdateSigningCertificate PolicyAction = "iam:UpdateSigningCertificate"

	// SSH Public Keys
	PolicyActionIAMUploadSSHPublicKey PolicyAction = "iam:UploadSSHPublicKey"
	PolicyActionIAMDeleteSSHPublicKey PolicyAction = "iam:DeleteSSHPublicKey"
	PolicyActionIAMGetSSHPublicKey    PolicyAction = "iam:GetSSHPublicKey"
	PolicyActionIAMListSSHPublicKeys  PolicyAction = "iam:ListSSHPublicKeys"
	PolicyActionIAMUpdateSSHPublicKey PolicyAction = "iam:UpdateSSHPublicKey"

	// Service Specific Credentials
	PolicyActionIAMCreateServiceSpecificCredential PolicyAction = "iam:CreateServiceSpecificCredential"
	PolicyActionIAMDeleteServiceSpecificCredential PolicyAction = "iam:DeleteServiceSpecificCredential"
	PolicyActionIAMResetServiceSpecificCredential  PolicyAction = "iam:ResetServiceSpecificCredential"
	PolicyActionIAMUpdateServiceSpecificCredential PolicyAction = "iam:UpdateServiceSpecificCredential"
	PolicyActionIAMListServiceSpecificCredentials  PolicyAction = "iam:ListServiceSpecificCredentials"

	// Miscellaneous / Account & Reports
	PolicyActionIAMCreateAccountAlias                 PolicyAction = "iam:CreateAccountAlias"
	PolicyActionIAMDeleteAccountAlias                 PolicyAction = "iam:DeleteAccountAlias"
	PolicyActionIAMListAccountAliases                 PolicyAction = "iam:ListAccountAliases"
	PolicyActionIAMGetAccountPasswordPolicy           PolicyAction = "iam:GetAccountPasswordPolicy"
	PolicyActionIAMUpdateAccountPasswordPolicy        PolicyAction = "iam:UpdateAccountPasswordPolicy"
	PolicyActionIAMDeleteAccountPasswordPolicy        PolicyAction = "iam:DeleteAccountPasswordPolicy"
	PolicyActionIAMGetAccountSummary                  PolicyAction = "iam:GetAccountSummary"
	PolicyActionIAMGetAccountAuthorizationDetails     PolicyAction = "iam:GetAccountAuthorizationDetails"
	PolicyActionIAMGetContextKeysForCustomPolicy      PolicyAction = "iam:GetContextKeysForCustomPolicy"
	PolicyActionIAMGetOrganizationsAccessReport       PolicyAction = "iam:GetOrganizationsAccessReport"
	PolicyActionIAMGetServiceLastAccessedDetails      PolicyAction = "iam:GetServiceLastAccessedDetails"
	PolicyActionIAMGetServiceLinkedRoleDeletionStatus PolicyAction = "iam:GetServiceLinkedRoleDeletionStatus"
	PolicyActionIAMCreateServiceLinkedRole            PolicyAction = "iam:CreateServiceLinkedRole"
	PolicyActionIAMDeleteServiceLinkedRole            PolicyAction = "iam:DeleteServiceLinkedRole"
)

type PolicyCondOp string

func (v PolicyCondOp) String() string {
	return string(v)
}

const (
	// Multivalued string condition operators prefix
	CondOpForAllValuesPrefix = "ForAllValues:" // Condition is true only if every request value matches. (AND logic)
	CondOpForAnyValuePrefix  = "ForAnyValue:"  // Condition is true if at least one request value matches. (OR logic)

	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_IfExists
	CondOpIfExistsSuffix PolicyCondOp = "IfExists"

	// String condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_String
	CondOpStringEquals              PolicyCondOp = "StringEquals"
	CondOpStringNotEquals           PolicyCondOp = "StringNotEquals"
	CondOpStringEqualsIgnoreCase    PolicyCondOp = "StringEqualsIgnoreCase"
	CondOpStringNotEqualsIgnoreCase PolicyCondOp = "StringNotEqualsIgnoreCase"
	CondOpStringLike                PolicyCondOp = "StringLike"
	CondOpStringNotLike             PolicyCondOp = "StringNotLike"

	// Numeric condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_Numeric
	CondOpNumericEquals            PolicyCondOp = "NumericEquals"
	CondOpNumericNotEquals         PolicyCondOp = "NumericNotEquals"
	CondOpNumericLessThan          PolicyCondOp = "NumericLessThan"
	CondOpNumericLessThanEquals    PolicyCondOp = "NumericLessThanEquals"
	CondOpNumericGreaterThan       PolicyCondOp = "NumericGreaterThan"
	CondOpNumericGreaterThanEquals PolicyCondOp = "NumericGreaterThanEquals"

	// Date condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_Date
	CondOpDateEquals            PolicyCondOp = "DateEquals"
	CondOpDateNotEquals         PolicyCondOp = "DateNotEquals"
	CondOpDateLessThan          PolicyCondOp = "DateLessThan"
	CondOpDateLessThanEquals    PolicyCondOp = "DateLessThanEquals"
	CondOpDateGreaterThan       PolicyCondOp = "DateGreaterThan"
	CondOpDateGreaterThanEquals PolicyCondOp = "DateGreaterThanEquals"

	// Boolean condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_Boolean
	CondOpBool PolicyCondOp = "Bool"

	// Binary condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_BinaryEquals
	CondOpBinaryEquals PolicyCondOp = "BinaryEquals"

	// IP address condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_IPAddress
	CondOpIpAddress    PolicyCondOp = "IpAddress"
	CondOpNotIpAddress PolicyCondOp = "NotIpAddress"

	// ARN condition operators
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_ARN
	CondOpArnEquals    PolicyCondOp = "ArnEquals"
	CondOpArnLike      PolicyCondOp = "ArnLike"
	CondOpArnNotEquals PolicyCondOp = "ArnNotEquals"
	CondOpArnNotLike   PolicyCondOp = "ArnNotLike"

	// Condition operator to check existence of condition keys
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_condition_operators.html#Conditions_Null
	CondOpNull PolicyCondOp = "Null"
)

var (
	// All IAM condtion operators
	AllSupportedConditions = map[PolicyCondOp]struct{}{}

	// All actions from supported services. i.e s3, iam
	AllSupportedActions = []PolicyAction{}

	// All S3 actions that are supported by the tool
	AllS3SupportedActions = []PolicyAction{
		// PolicyActionS3CreateAccessGrant,
		// PolicyActionS3GetAccessGrant,
		// PolicyActionS3DeleteAccessGrant,
		// PolicyActionS3CreateAccessGrantsInstance,
		// PolicyActionS3GetAccessGrantsInstance,
		// PolicyActionS3GetAccessGrantsInstanceForPrefix,
		// PolicyActionS3GetAccessGrantsInstanceResourcePolicy,
		// PolicyActionS3DeleteAccessGrantsInstance,
		// PolicyActionS3DeleteAccessGrantsInstanceResourcePolicy,
		// PolicyActionS3CreateAccessGrantsLocation,
		// PolicyActionS3GetAccessGrantsLocation,
		// PolicyActionS3DeleteAccessGrantsLocation,
		// PolicyActionS3AssociateAccessGrantsIdentityCenter,
		// PolicyActionS3DissociateAccessGrantsIdentityCenter,

		// PolicyActionS3CreateAccessPoint,
		// PolicyActionS3GetAccessPoint,
		// PolicyActionS3DeleteAccessPoint,
		// PolicyActionS3PutAccessPointPolicy,
		// PolicyActionS3CreateAccessPointForObjectLambda,
		// PolicyActionS3GetAccessPointConfigurationForObjectLambda,
		// PolicyActionS3DeleteAccessPointForObjectLambda,
		// PolicyActionS3DeleteAccessPointPolicy,
		// PolicyActionS3DeleteAccessPointPolicyForObjectLambda,
		// PolicyActionS3CreateMultiRegionAccessPoint,
		// PolicyActionS3DeleteMultiRegionAccessPoint,
		// PolicyActionS3DescribeMultiRegionAccessPointOperation,

		PolicyActionS3CreateBucket,
		PolicyActionS3DeleteBucket,
		// PolicyActionS3CreateBucketMetadataTableConfiguration,
		// PolicyActionS3DeleteBucketMetadataTableConfiguration,
		// PolicyActionS3GetBucketLocation,
		// PolicyActionS3ListAllMyBuckets,
		PolicyActionS3ListBucket,
		PolicyActionS3ListBucketVersions,
		// PolicyActionS3GetBucketAcl,
		// PolicyActionS3PutBucketAcl,
		// PolicyActionS3GetBucketCors,
		// PolicyActionS3PutBucketCors,
		// PolicyActionS3DeleteBucketCors,
		// PolicyActionS3GetBucketLogging,
		// PolicyActionS3PutBucketLogging,
		// PolicyActionS3GetBucketNotification,
		// PolicyActionS3PutBucketNotification,
		// PolicyActionS3GetBucketNotificationConfiguration,
		// PolicyActionS3PutBucketNotificationConfiguration,
		// PolicyActionS3PutBucketNotificationFilter,
		// PolicyActionS3GetBucketPolicy,
		// PolicyActionS3PutBucketPolicy,
		// PolicyActionS3GetBucketPolicyStatus,
		// PolicyActionS3DeleteBucketPolicy,
		// PolicyActionS3GetBucketRequestPayment,
		// PolicyActionS3PutBucketRequestPayment,
		PolicyActionS3GetBucketTagging,
		PolicyActionS3PutBucketTagging,
		PolicyActionS3DeleteBucketTagging,
		PolicyActionS3GetBucketVersioning,
		PolicyActionS3PutBucketVersioning,
		// PolicyActionS3GetBucketWebsite,
		// PolicyActionS3PutBucketWebsite,
		// PolicyActionS3DeleteBucketWebsite,

		PolicyActionS3GetLifecycleConfiguration,
		PolicyActionS3PutLifecycleConfiguration,
		PolicyActionS3DeleteLifecycleConfiguration,
		PolicyActionS3GetBucketLifecycle,
		PolicyActionS3PutBucketLifecycle,

		// PolicyActionS3GetBucketEncryption,
		// PolicyActionS3PutBucketEncryption,
		// PolicyActionS3GetReplicationConfiguration,
		// PolicyActionS3PutReplicationConfiguration,
		// PolicyActionS3DeleteReplicationConfiguration,
		// PolicyActionS3ReplicateObject,
		// PolicyActionS3ReplicateDelete,
		// PolicyActionS3ReplicateTags,
		PolicyActionS3GetObjectLockConfiguration,
		PolicyActionS3PutObjectLockConfiguration,
		PolicyActionS3GetBucketObjectLockConfiguration,
		PolicyActionS3GetObjectRetention,
		PolicyActionS3PutObjectRetention,
		PolicyActionS3GetObjectLegalHold,
		PolicyActionS3PutObjectLegalHold,
		PolicyActionS3BypassGovernanceRetention,

		PolicyActionS3GetObject,
		PolicyActionS3PutObject,
		PolicyActionS3DeleteObject,
		// PolicyActionS3GetObjectAcl,
		// PolicyActionS3PutObjectAcl,
		PolicyActionS3GetObjectTagging,
		PolicyActionS3PutObjectTagging,
		PolicyActionS3DeleteObjectTagging,
		PolicyActionS3GetObjectVersion,
		// PolicyActionS3GetObjectVersionAcl,
		// PolicyActionS3PutObjectVersionAcl,
		PolicyActionS3GetObjectVersionTagging,
		PolicyActionS3PutObjectVersionTagging,
		PolicyActionS3DeleteObjectVersion,
		PolicyActionS3DeleteObjectVersionTagging,
		// PolicyActionS3RestoreObject,
		// PolicyActionS3SelectObjectContent,
		// PolicyActionS3GetObjectTorrent,

		PolicyActionS3CreateMultipartUpload,
		PolicyActionS3AbortMultipartUpload,
		PolicyActionS3ListMultipartUploadParts,
		PolicyActionS3CompleteMultipartUpload,
		PolicyActionS3ListBucketMultipartUploads,

		// PolicyActionS3CreateJob,
		// PolicyActionS3ListJobs,
		// PolicyActionS3DescribeJob,
		// PolicyActionS3UpdateJob,
		// PolicyActionS3DeleteJobTagging,

		// PolicyActionS3GetBucketMetricsConfiguration,
		// PolicyActionS3PutBucketMetricsConfiguration,
		// PolicyActionS3DeleteBucketMetricsConfiguration,
		// PolicyActionS3ListStorageLensConfigurations,
		// PolicyActionS3GetStorageLensConfiguration,
		// PolicyActionS3PutStorageLensConfiguration,
		// PolicyActionS3DeleteStorageLensConfiguration,
		// PolicyActionS3DeleteStorageLensConfigurationTagging,
		// PolicyActionS3CreateStorageLensGroup,
		// PolicyActionS3DeleteStorageLensGroup,

		// PolicyActionS3GetAccelerateConfiguration,
		// PolicyActionS3PutAccelerateConfiguration,
	}

	// All IAM actions that are supported by the tool
	AllIAMSupportedActions = []PolicyAction{
		PolicyActionIAMCreateUser,
		PolicyActionIAMDeleteUser,
		PolicyActionIAMGetUser,
		PolicyActionIAMUpdateUser,
		PolicyActionIAMListUsers,
		PolicyActionIAMChangePassword,
		PolicyActionIAMGetUserPolicy,
		PolicyActionIAMListUserPolicies,
		PolicyActionIAMPutUserPolicy,
		PolicyActionIAMDeleteUserPolicy,
		PolicyActionIAMListAttachedUserPolicies,
		PolicyActionIAMTagUser,
		PolicyActionIAMUntagUser,
		PolicyActionIAMListUserTags,

		PolicyActionIAMCreateAccessKey,
		PolicyActionIAMDeleteAccessKey,
		PolicyActionIAMUpdateAccessKey,
		PolicyActionIAMListAccessKeys,
		PolicyActionIAMGetAccessKeyLastUsed,

		PolicyActionIAMCreateGroup,
		PolicyActionIAMUpdateGroup,
		PolicyActionIAMDeleteGroup,
		PolicyActionIAMGetGroup,
		PolicyActionIAMListGroups,
		PolicyActionIAMAddUserToGroup,
		PolicyActionIAMRemoveUserFromGroup,
		PolicyActionIAMPutGroupPolicy,
		PolicyActionIAMDeleteGroupPolicy,
		PolicyActionIAMGetGroupPolicy,
		PolicyActionIAMListGroupPolicies,
		PolicyActionIAMAttachGroupPolicy,
		PolicyActionIAMDetachGroupPolicy,
		PolicyActionIAMTagGroup,
		PolicyActionIAMUntagGroup,
		PolicyActionIAMListAttachedGroupPolicies,
		PolicyActionIAMListGroupsForUser,

		PolicyActionIAMCreatePolicy,
		PolicyActionIAMDeletePolicy,
		PolicyActionIAMGetPolicy,
		PolicyActionIAMListPolicies,
		// PolicyActionIAMListPoliciesGrantingServiceAccess,
		PolicyActionIAMCreatePolicyVersion,
		PolicyActionIAMDeletePolicyVersion,
		PolicyActionIAMGetPolicyVersion,
		PolicyActionIAMListPolicyVersions,
		PolicyActionIAMSetDefaultPolicyVersion,
		PolicyActionIAMAttachUserPolicy,
		PolicyActionIAMDetachUserPolicy,
		PolicyActionIAMTagPolicy,
		PolicyActionIAMUntagPolicy,
		PolicyActionIAMListPolicyTags,
		PolicyActionIAMListEntitiesForPolicy,

		// PolicyActionIAMPutUserPermissionsBoundary,
		// PolicyActionIAMDeleteUserPermissionsBoundary,
		// PolicyActionIAMPutRolePermissionsBoundary,
		// PolicyActionIAMDeleteRolePermissionsBoundary,

		// PolicyActionIAMCreateAccountAlias,
		// PolicyActionIAMDeleteAccountAlias,
		// PolicyActionIAMListAccountAliases,
		// PolicyActionIAMGetAccountPasswordPolicy,
		// PolicyActionIAMUpdateAccountPasswordPolicy,
		// PolicyActionIAMDeleteAccountPasswordPolicy,
		// PolicyActionIAMGetAccountSummary,
		// PolicyActionIAMGetAccountAuthorizationDetails,
		// PolicyActionIAMGetContextKeysForCustomPolicy,
		// PolicyActionIAMGetOrganizationsAccessReport,
		// PolicyActionIAMGetServiceLastAccessedDetails,
		// PolicyActionIAMGetServiceLinkedRoleDeletionStatus,
		// PolicyActionIAMCreateServiceLinkedRole,
		// PolicyActionIAMDeleteServiceLinkedRole,
	}
)

func init() {
	setAllSupportedConditions()
	setAllSupportedActions()
}

func IsValidIAMCondition(op PolicyCondOp) bool {
	_, ok := AllSupportedConditions[op]
	return ok
}

func setAllSupportedActions() {
	AllSupportedActions = append(AllSupportedActions, AllS3SupportedActions...)
	AllSupportedActions = append(AllSupportedActions, AllIAMSupportedActions...)
}

func setAllSupportedConditions() {
	temp := make(map[PolicyCondOp]struct{}, 0)

	prefixes := []PolicyCondOp{
		"",
		CondOpForAllValuesPrefix,
		CondOpForAnyValuePrefix,
	}

	// String Conditions
	stringConditions := []PolicyCondOp{
		CondOpStringEquals,
		CondOpStringNotEquals,
		CondOpStringEqualsIgnoreCase,
		CondOpStringNotEqualsIgnoreCase,
		CondOpStringLike,
		CondOpStringNotLike,
	}

	for i := range stringConditions {
		for _, prefix := range prefixes {
			temp[prefix+stringConditions[i]] = struct{}{}
		}
	}

	// Numeric Conditions
	numericConditions := []PolicyCondOp{
		CondOpNumericEquals,
		CondOpNumericNotEquals,
		CondOpNumericLessThan,
		CondOpNumericLessThanEquals,
		CondOpNumericGreaterThan,
		CondOpNumericGreaterThanEquals,
	}
	for i := range numericConditions {
		temp[numericConditions[i]] = struct{}{}
	}

	// Date Conditions
	dateConditions := []PolicyCondOp{
		CondOpDateEquals,
		CondOpDateNotEquals,
		CondOpDateLessThan,
		CondOpDateLessThanEquals,
		CondOpDateGreaterThan,
		CondOpDateGreaterThanEquals,
	}
	for i := range dateConditions {
		temp[dateConditions[i]] = struct{}{}
	}

	// Boolean Conditions
	booleanConditions := []PolicyCondOp{CondOpBool}
	for i := range booleanConditions {
		for _, prefix := range prefixes {
			temp[prefix+booleanConditions[i]] = struct{}{}
		}
	}

	// Binary Conditions
	temp[CondOpBinaryEquals] = struct{}{}

	// IP Address Conditions
	temp[CondOpIpAddress] = struct{}{}
	temp[CondOpNotIpAddress] = struct{}{}

	// ARN Conditions
	arnConditions := []PolicyCondOp{
		CondOpArnEquals,
		CondOpArnLike,
		CondOpArnNotEquals,
		CondOpArnNotLike,
	}
	for i := range arnConditions {
		for _, prefix := range prefixes {
			temp[prefix+arnConditions[i]] = struct{}{}
		}
	}

	// IfExists Suffix Conditions
	for k := range temp {
		AllSupportedConditions[k] = struct{}{}
		AllSupportedConditions[k+CondOpIfExistsSuffix] = struct{}{}
	}

	// Null Condition
	AllSupportedConditions[CondOpNull] = struct{}{}
}
