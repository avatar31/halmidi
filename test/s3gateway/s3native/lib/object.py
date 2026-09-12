import boto3
from botocore.client import Config

from s3native.lib.client import S3BaseClient


class S3ObjectManager(S3BaseClient):
    def __init__(self, s3_client, bucket_manager):
        super().__init__(s3_client)
        self.bucket_manager = bucket_manager

    def put_object(self, bucket_name, object_key, body, **kwargs):
        return self.client.put_object(
            Bucket=bucket_name, Key=object_key, Body=body, **kwargs
        )

    def create_multipart_upload(self, bucket_name, object_key, **kwargs):
        return self.client.create_multipart_upload(
            Bucket=bucket_name, Key=object_key, **kwargs
        )

    def upload_part(self, bucket_name, object_key, upload_id, part_number, body):
        return self.client.upload_part(
            Bucket=bucket_name,
            Key=object_key,
            UploadId=upload_id,
            PartNumber=part_number,
            Body=body,
        )

    def abort_multipart_upload(self, bucket_name, object_key, upload_id):
        return self.client.abort_multipart_upload(
            Bucket=bucket_name, Key=object_key, UploadId=upload_id
        )

    def list_objects(self, bucket_name, **kwargs):
        return self.client.list_objects_v2(Bucket=bucket_name, **kwargs)

    def list_object_versions(self, bucket_name, **kwargs):
        return self.client.list_object_versions(Bucket=bucket_name, **kwargs)

    def list_multipart_uploads(self, bucket_name, **kwargs):
        return self.client.list_multipart_uploads(Bucket=bucket_name, **kwargs)

    def delete_object(self, bucket_name, object_key, **kwargs):
        return self.client.delete_object(Bucket=bucket_name, Key=object_key, **kwargs)

    def delete_all_objects(self, bucket_name):
        objects_to_delete = self.client.list_objects_v2(Bucket=bucket_name)

        if "Contents" in objects_to_delete:
            for obj in objects_to_delete["Contents"]:
                self.delete_object(bucket_name, obj["Key"])
