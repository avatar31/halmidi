# [S3 Features](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)

## 🧩 1. Amazon S3

### 👉 Core service

* **What it is:** AWS’s primary **object storage service**
* **Used for:** Storing files (objects) in buckets
* **Key features:**

  * Upload/download objects
  * Lifecycle policies
  * Versioning, encryption, replication
* **Scope:** Data storage itself

✅ Think: *“Where my data lives”*

---

## 🛠️ 2. Amazon S3 Control

### 👉 Management/control plane for S3 at scale

* **What it is:** A **separate API layer** to manage S3 resources across accounts and organizations
* **Used for:**

  * Managing **access points**
  * Bulk operations (batch jobs)
  * Account-level settings
* **Does NOT store data**

✅ Think: *“Admin panel for S3 across accounts”*

---

## 🏢 3. Amazon S3 on Outposts

### 👉 S3 in your data center

* **What it is:** S3 running on **AWS Outposts hardware on-premises**
* **Used for:**

  * Low-latency local storage
  * Data residency requirements
* **Still managed by AWS**

✅ Think: *“S3, but physically inside your data center”*

---

## 📊 4. Amazon S3 Tables

### 👉 Structured/tabular data in S3

* **What it is:** A newer feature for **storing and querying tabular datasets**
* **Used for:**

  * Analytics workloads
  * Data lakes
  * Integration with engines like Athena
* **Adds structure on top of S3 objects**

✅ Think: *“S3 optimized for tables and analytics”*

---

## 🧠 5. Amazon S3 Vectors

### 👉 Vector data storage (AI/ML)

* **What it is:** A feature for storing **vector embeddings** (used in AI/semantic search)
* **Used for:**

  * Similarity search
  * AI applications (RAG, semantic retrieval)
* Works with vector indexes and queries

✅ Think: *“S3 for AI embeddings and vector search”*

---

## ⚖️ Side-by-side comparison

| Feature                   | Purpose           | Stores Data? | Typical Use Case           |
| ------------------------- | ----------------- | ------------ | -------------------------- |
| **Amazon S3**             | Object storage    | ✅ Yes        | Files, backups, media      |
| **Amazon S3 Control**     | Management API    | ❌ No         | Admin, governance          |
| **Amazon S3 on Outposts** | On-prem S3        | ✅ Yes        | Hybrid/cloud-local storage |
| **Amazon S3 Tables**      | Tabular datasets  | ✅ Yes        | Analytics, data lakes      |
| **Amazon S3 Vectors**     | Vector embeddings | ✅ Yes        | AI/ML, semantic search     |

---

## 🧠 Key mental model

* **S3** → Storage foundation
* **S3 Control** → Management layer
* **S3 on Outposts** → Deployment option (where it runs)
* **S3 Tables** → Structured analytics layer
* **S3 Vectors** → AI/ML vector layer

---


# [IAM Features](https://docs.aws.amazon.com/IAM/latest/UserGuide/introduction.html)
AWS Identity and Access Management (IAM) is a web service for securely controlling access to AWS services. With IAM, you can centrally manage users, security credentials such as access keys, and permissions that control which AWS resources users and applications can access.



## AWS S3 vs MinIO vs Halmidi

### [AWS S3 API Reference](https://docs.aws.amazon.com/AmazonS3/latest/API/API_Operations_Amazon_Simple_Storage_Service.html)

| Feature | AWS S3 | MinIO | Halmidi |
| --- | --- | --- | --- |
| Bucket APIs | | | |
| [HeadBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadBucket.html) | ✅YES | ✅YES | ✅YES |
| [CreateBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html) | ✅YES | ✅YES | ✅YES |
| [DeleteBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucket.html) | ✅YES | ✅YES | ✅YES |
| [ListBuckets](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBuckets.html) | ✅YES | ✅YES | ✅YES |
| [ListDirectoryBuckets](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListDirectoryBuckets.html) | ✅YES | ✅YES | |
| Bucket Lifecycle | | | |
| [PutBucketLifecycle](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketLifecycle.html) | ✅YES | ✅YES | ✅YES |
| [PutBucketLifecycleConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketLifecycleConfiguration.html) | ✅YES | ✅YES | |
| [GetBucketLifecycle](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLifecycle.html) | ✅YES | ✅YES | ✅YES |
| [GetBucketLifecycleConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLifecycleConfiguration.html) | ✅YES | ✅YES | |
| [DeleteBucketLifecycle](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketLifecycle.html) | ✅YES | ✅YES | ✅YES |
| Bucket Notification | | | |
| [PutBucketNotification](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketNotification.html) | ✅YES | ✅YES | |
| [PutBucketNotificationConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketNotificationConfiguration.html) | ✅YES | ✅YES | |
| [GetBucketNotification](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketNotification.html) | ✅YES | ✅YES | |
| [GetBucketNotificationConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketNotificationConfiguration.html) | ✅YES | ✅YES | |
| Bucket Policy | | | |
| [PutBucketPolicy](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketPolicy.html) | ✅YES | ✅YES | |
| [DeleteBucketPolicy](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketPolicy.html) | ✅YES | ✅YES | |
| [GetBucketPolicy](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketPolicy.html) | ✅YES | ✅YES | |
| [GetBucketPolicyStatus](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketPolicyStatus.html) | ✅YES | ✅YES | |
| Bucket Versioning | | | |
| [PutBucketVersioning](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketVersioning.html) | ✅YES | ✅YES | ✅YES |
| [GetBucketVersioning](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketVersioning.html) | ✅YES | ✅YES | ✅YES |
| Bucket Replication | | | |
| [PutBucketReplication](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketReplication.html) | ✅YES | ✅YES | |
| [DeleteBucketReplication](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketReplication.html) | ✅YES | ✅YES | |
| [GetBucketReplication](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketReplication.html) | ✅YES | ✅YES | |
| Object APIs | | | |
| [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html) | ✅YES | ✅YES | ✅YES |
| [HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html) | ✅YES | ✅YES | |
| [GetObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html) | ✅YES | ✅YES | ✅YES |
| [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html) | ✅YES | ✅YES | ✅YES |
| [DeleteObjects](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html) | ✅YES | ✅YES | ✅YES |
| [ListObjects](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjects.html) | ✅YES | ✅YES | ❌NO |
| [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html) | ✅YES | ✅YES | ✅YES |
| [ListObjectVersions](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectVersions.html) | ✅YES | ✅YES | ✅YES |
| [SelectObjectContent](https://docs.aws.amazon.com/AmazonS3/latest/API/API_SelectObjectContent.html) | ✅YES | ✅YES | |
| [CopyObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CopyObject.html) | ✅YES | ✅YES | ✅YES |
| [HeadObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html) | ✅YES | ✅YES | | ✅YES |
| [GetObjectAttributes](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html) | ✅YES | ✅YES | | ✅YES |
| [RenameObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_RenameObject.html) | ✅YES | ✅YES | |
| [UpdateObjectEncryption](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UpdateObjectEncryption.html) | ✅YES | ❌NO | |
| [WriteGetObjectResponse](https://docs.aws.amazon.com/AmazonS3/latest/API/API_WriteGetObjectResponse.html) | ✅YES | ❌NO | |
| [RestoreObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_RestoreObject.html) | ✅YES | ❌NO | |
| Object Locking | | | |
| [GetObjectLegalHold](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectLegalHold.html) | ✅YES | ✅YES | ✅YES |
| [PutObjectLegalHold](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectLegalHold.html) | ✅YES | ✅YES | ✅YES |
| [GetObjectRetention](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectRetention.html) | ✅YES | ✅YES | |
| [PutObjectRetention](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectRetention.html) | ✅YES | ✅YES | | |
| [GetObjectLockConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectLockConfiguration.html) | ✅YES | ✅YES | ✅YES |
| [PutObjectLockConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectLockConfiguration.html) | ✅YES | ✅YES | ✅YES |
| Multipart Uploads | | | |
| [CreateMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html) | ✅YES | ✅YES | ✅YES |
| [CompleteMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html) | ✅YES | ✅YES | ✅YES |
| [AbortMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_AbortMultipartUpload.html) | ✅YES | ✅YES | ✅YES |
| [ListMultipartUploads](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListMultipartUploads.html) | ✅YES | ✅YES | |
| [ListParts](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListParts.html) | ✅YES | ✅YES | ✅YES |
| [UploadPart](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPart.html) | ✅YES | ✅YES | ✅YES |
| [UploadPartCopy](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPartCopy.html) | ✅YES | ✅YES | |
| BucketEncryption | | | |
| [PutBucketEncryption](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketEncryption.html) | ✅YES | ✅YES | |
| [GetBucketEncryption](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketEncryption.html) | ✅YES | ✅YES | |
| [DeleteBucketEncryption](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketEncryption.html) | ✅YES | ✅YES | |
| Bucket/Object Tagging | | | |
| [PutBucketTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketTagging.html) | ✅YES | ✅YES | ✅YES |
| [PutObjectTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectTagging.html) | ✅YES | ✅YES | ✅YES |
| [DeleteBucketTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketTagging.html) | ✅YES | ✅YES | ✅YES |
| [DeleteObjectTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjectTagging.html) | ✅YES | ✅YES | ✅YES |
| [GetBucketTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketTagging.html) | ✅YES | ✅YES | ✅YES |
| [GetObjectTagging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectTagging.html) | ✅YES | ✅YES | ✅YES |
| BucketCors | | | |
| [PutBucketCors](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketCors.html) | ✅YES | ✅YES | |
| [GetBucketCors](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketCors.html) | ✅YES | ✅YES | |
| [DeleteBucketCors](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketCors.html) | ✅YES | ✅YES | |
| BucketIntelligentTieringConfiguration | | | |
| [PutBucketIntelligentTieringConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketIntelligentTieringConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketIntelligentTieringConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketIntelligentTieringConfiguration.html) | ✅YES| ❌NO | |
| [GetBucketIntelligentTieringConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketIntelligentTieringConfiguration.html) | ✅YES | ❌NO | |
| [ListBucketIntelligentTieringConfigurations](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBucketIntelligentTieringConfigurations.html) | ✅YES | ❌NO | |
| BucketMetadataConfiguration | | | |
| [CreateBucketMetadataConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucketMetadataConfiguration.html) | ✅YES | ❌NO | |
| [CreateBucketMetadataTableConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucketMetadataTableConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketMetadataConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketMetadataConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketMetadataTableConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketMetadataTableConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketMetadataConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketMetadataConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketMetadataTableConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketMetadataTableConfiguration.html) | ✅YES | ❌NO | |
| [UpdateBucketMetadataInventoryTableConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UpdateBucketMetadataInventoryTableConfiguration.html) | ✅YES | ❌NO | |
| [UpdateBucketMetadataJournalTableConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UpdateBucketMetadataJournalTableConfiguration.html) | ✅YES | ❌NO | |
| BucketOwnershipControls | | | |
| [PutBucketOwnershipControls](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketOwnershipControls.html) | ✅YES | ❌NO | |
| [DeleteBucketOwnershipControls](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketOwnershipControls.html) | ✅YES | ❌NO | |
| [GetBucketOwnershipControls](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketOwnershipControls.html) | ✅YES | ❌NO | |
| BucketAnalyticsConfiguration | | | |
| [PutBucketAnalyticsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketAnalyticsConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketAnalyticsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketAnalyticsConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketAnalyticsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAnalyticsConfiguration.html) | ✅YES | ❌NO | |
| [ListBucketAnalyticsConfigurations](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBucketAnalyticsConfigurations.html) | ✅YES | ❌NO | |
| BucketMetricsConfiguration | | | |
| [PutBucketMetricsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketMetricsConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketMetricsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketMetricsConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketMetricsConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketMetricsConfiguration.html) | ✅YES | ❌NO | |
| [ListBucketMetricsConfigurations](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBucketMetricsConfigurations.html) | ✅YES | ❌NO | |
| BucketInventoryConfiguration | | | |
| [PutBucketInventoryConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketInventoryConfiguration.html) | ✅YES | ❌NO | |
| [DeleteBucketInventoryConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketInventoryConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketInventoryConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketInventoryConfiguration.html) | ✅YES | ❌NO | |
| [ListBucketInventoryConfigurations](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBucketInventoryConfigurations.html) | ✅YES | ❌NO | |
| Bucket Website | | | |
| [PutBucketWebsite](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketWebsite.html) | ✅YES | ❌NO | |
| [DeleteBucketWebsite](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketWebsite.html) | ✅YES | ❌NO | |
| [GetBucketWebsite](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketWebsite.html) | ✅YES | ❌NO | |
| Bucket PublicAccessBlock | | | |
| [PutPublicAccessBlock](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutPublicAccessBlock.html) | ✅YES | ❌NO | |
| [DeletePublicAccessBlock](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeletePublicAccessBlock.html) | ✅YES | ❌NO | |
| [GetPublicAccessBlock](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetPublicAccessBlock.html) | ✅YES | ❌NO | |
| Bucket/Object Acl (Halmidi won't support these as AWS announced End of Support notice) | | | |
| [PutBucketAcl](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketAcl.html) | ✅YES | ❌NO | ❌NO |
| [PutObjectAcl](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObjectAcl.html) | ✅YES | ❌NO | ❌NO |
| [GetBucketAcl](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAcl.html) | ✅YES | ❌NO | ❌NO |
| [GetObjectAcl](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAcl.html) | ✅YES | ❌NO | ❌NO |
| Bucket Logging | | | |
| [PutBucketLogging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketLogging.html) | ✅YES | ❌NO | |
| [GetBucketLogging](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLogging.html) | ✅YES | ❌NO | |
| Bucket RequestPayment | | | |
| [PutBucketRequestPayment](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketRequestPayment.html) | ✅YES | ❌NO | |
| [GetBucketRequestPayment](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketRequestPayment.html) | ✅YES | ❌NO | |
| Bucket AccelerateConfiguration | | | |
| [PutBucketAccelerateConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketAccelerateConfiguration.html) | ✅YES | ❌NO | |
| [GetBucketAccelerateConfiguration](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAccelerateConfiguration.html) | ✅YES | ❌NO | |
| BucketAbac | | | |
| [PutBucketAbac](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutBucketAbac.html) | ✅YES | ❌NO | |
| [GetBucketAbac](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAbac.html) | ✅YES | ❌NO | |
| Miscellaneous | | | |
| [GetBucketLocation](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html) | ✅YES | ❌NO | ❌NO |
| [CreateSession](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateSession.html) | ✅YES | ❌NO | ❌NO |
| [GetObjectTorrent](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectTorrent.html) | ✅YES | ❌NO | |


### [AWS IAM API Reference](https://docs.aws.amazon.com/IAM/latest/APIReference/API_Operations.html)

| Feature | AWS IAM | Halmidi |
| --- | --- | --- |
| User APIs | | |
| [CreateUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateUser.html) | ✅YES | ✅YES |
| [DeleteUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteUser.html) | ✅YES | ✅YES |
| [GetUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUser.html) | ✅YES | ✅YES |
| [UpdateUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateUser.html) | ✅YES | ✅YES |
| [ListUsers](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUsers.html) | ✅YES | ✅YES |
| [ChangePassword](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ChangePassword.html) | ✅YES | ✅YES |
| [GetUserPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUserPolicy.html) | ✅YES | ✅YES |
| [ListUserPolicies](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserPolicies.html) | ✅YES | ✅YES |
| [PutUserPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutUserPolicy.html) | ✅YES | ✅YES |
| [DeleteUserPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteUserPolicy.html) | ✅YES | ✅YES |
| [ListAttachedUserPolicies](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedUserPolicies.html) | ✅YES | ✅YES |
| [TagUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagUser.html) | ✅YES | ✅YES |
| [UntagUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagUser.html) | ✅YES | ✅YES |
| [ListUserTags](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListUserTags.html) | ✅YES | ✅YES |
| Access Key APIs | | |
| [CreateAccessKey](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateAccessKey.html) | ✅YES | ✅YES |
| [DeleteAccessKey](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteAccessKey.html) | ✅YES | ✅YES |
| [UpdateAccessKey](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateAccessKey.html) | ✅YES | ✅YES |
| [GetAccessKey](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetAccessKey.html) | ✅YES | ✅YES |
| [ListAccessKeys](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccessKeys.html) | ✅YES | ✅YES
| [GetAccessKeyLastUsed](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetAccessKeyLastUsed.html) | ✅YES | ✅YES |
| Group APIs | | |
| [CreateGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreateGroup.html) | ✅YES | ✅YES |
| [UpdateGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UpdateGroup.html) | ✅YES | ✅YES |
| [DeleteGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroup.html) | ✅YES | ✅YES |
| [GetGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroup.html) | ✅YES | ✅YES |
| [ListGroups](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroups.html) | ✅YES | ✅YES |
| [AddUserToGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_AddUserToGroup.html) | ✅YES | ✅YES |
| [RemoveUserFromGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_RemoveUserFromGroup.html) | ✅YES | ✅YES |
| [PutGroupPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_PutGroupPolicy.html) | ✅YES | ✅YES |
| [DeleteGroupPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeleteGroupPolicy.html) | ✅YES | ✅YES |
| [GetGroupPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetGroupPolicy.html) | ✅YES | ✅YES |
| [ListGroupPolicies](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupPolicies.html) | ✅YES | ✅YES |
| [AttachGroupPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachGroupPolicy.html) | ✅YES | ✅YES |
| [DetachGroupPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachGroupPolicy.html) | ✅YES | ✅YES |
| [TagGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagGroup.html) | ✅YES | ✅YES |
| [UntagGroup](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagGroup.html) | ✅YES | ✅YES |
| [ListAttachedGroupPolicies](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAttachedGroupPolicies.html) | ✅YES | ✅YES |
| [ListGroupForUser](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListGroupsForUser.html) | ✅YES | ✅YES |
| IAM Policy APIs | | |
| [CreatePolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicy.html) | ✅YES | ✅YES |
| [DeletePolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicy.html) | ✅YES | ✅YES |
| [GetPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicy.html) | ✅YES | ✅YES |
| [ListPolicies](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicies.html) | ✅YES | ✅YES |
| [CreatePolicyVersion](https://docs.aws.amazon.com/IAM/latest/APIReference/API_CreatePolicyVersion.html) | ✅YES | ✅YES |
| [DeletePolicyVersion](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DeletePolicyVersion.html) | ✅YES | ✅YES |
| [GetPolicyVersion](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetPolicyVersion.html) | ✅YES | ✅YES |
| [ListPolicyVersions](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicyVersions.html) | ✅YES | ✅YES |
| [SetDefaultPolicyVersion](https://docs.aws.amazon.com/IAM/latest/APIReference/API_SetDefaultPolicyVersion.html) | ✅YES | ✅YES |
| [AttachUserPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_AttachUserPolicy.html) | ✅YES | ✅YES |
| [DetachUserPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_DetachUserPolicy.html) | ✅YES | ✅YES |
| [TagPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_TagPolicy.html) | ✅YES | ✅YES |
| [UntagPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_UntagPolicy.html) | ✅YES | ✅YES |
| [ListPolicyTags](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListPolicyTags.html) | ✅YES | ✅YES |
| [ListEntitiesForPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListEntitiesForPolicy.html) | ✅YES | ✅YES |


# References
- [MinIO S3 Compatibility](https://minio-docs.tf.fo/reference/s3-api-compatibility)

