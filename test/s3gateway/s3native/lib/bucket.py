from .client import S3BaseClient


class S3BucketManager(S3BaseClient):
    def head_bucket(self, bucket_name):
        """Head a bucket to check its existence."""
        return self.client.head_bucket(Bucket=bucket_name)

    def list_buckets(self):
        """List all buckets in the S3 account."""
        return self.client.list_buckets()

    def create_bucket(self, bucket_name, **kwargs):
        """Create a new bucket with the given name."""
        return self.client.create_bucket(Bucket=bucket_name, **kwargs)

    def delete_bucket(self, bucket_name):
        """Delete a bucket by name."""
        return self.client.delete_bucket(Bucket=bucket_name)

    def bucket_exists(self, bucket_name):
        """Check if a bucket exists by attempting to head the bucket."""
        return self.client.head_bucket(Bucket=bucket_name)

    def put_bucket_tagging(self, bucket_name, tag_set):
        """Put tagging on a bucket."""
        return self.client.put_bucket_tagging(
            Bucket=bucket_name,
            Tagging={"TagSet": [{"Key": k, "Value": v} for k, v in tag_set.items()]},
        )

    def get_bucket_tagging(self, bucket_name):
        """Get tagging of a bucket."""
        tagging_response = self.client.get_bucket_tagging(Bucket=bucket_name)
        return {tag["Key"]: tag["Value"] for tag in tagging_response.get("TagSet", [])}

    def get_bucket_versioning(self, bucket_name):
        """Get versioning status of a bucket."""
        return self.client.get_bucket_versioning(Bucket=bucket_name)

    def get_object_lock_configuration(self, bucket_name):
        """Get object lock configuration of a bucket."""
        return self.client.get_object_lock_configuration(Bucket=bucket_name)

    def get_bucket_acl(self, bucket_name):
        """Get ACL of a bucket."""
        return self.client.get_bucket_acl(Bucket=bucket_name)

    def get_bucket_ownership_controls(self, bucket_name):
        """Get ownership controls of a bucket."""
        return self.client.get_bucket_ownership_controls(Bucket=bucket_name)

    def put_bucket_tagging(self, bucket_name, tag_set):
        """Put tagging on a bucket."""
        return self.client.put_bucket_tagging(
            Bucket=bucket_name,
            Tagging={"TagSet": [{"Key": k, "Value": v} for k, v in tag_set.items()]},
        )

    def put_bucket_versioning(self, bucket_name, status):
        """Put versioning status on a bucket."""
        return self.client.put_bucket_versioning(
            Bucket=bucket_name,
            VersioningConfiguration={"Status": status},
        )

    def put_bucket_lifecycle_configuration(self, bucket_name, lifecycle_configuration):
        """Put lifecycle configuration on a bucket."""
        return self.client.put_bucket_lifecycle_configuration(
            Bucket=bucket_name,
            LifecycleConfiguration=lifecycle_configuration,
        )

    def delete_bucket(self, bucket_name):
        """Delete a bucket by name."""
        return self.client.delete_bucket(Bucket=bucket_name)

    def get_dummy_lifecycle_configuration(self):
        """Get a dummy lifecycle configuration for testing."""
        return {
            "Rules": [
                {
                    "ID": "Expire old objects",
                    "Status": "Enabled",
                    "Filter": {"Prefix": ""},
                    "Expiration": {"Days": 30},
                }
            ]
        }
