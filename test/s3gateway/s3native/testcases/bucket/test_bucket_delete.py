import pytest
from http import HTTPStatus

from s3native.lib import bucket, object
from s3native.utils import get_unique_id
from s3native.testcases.helpers import (
    validate_response_status,
    validate_error_response,
    validate_bucket_tags,
    validate_object_lock_configuration,
    validate_bucket_versioning,
    upload_object_versions,
)


class TestBucketDelete:
    bucket_name: str = ""

    def setup_method(self, method):
        """
        Setup method for testcase
        """
        self.bucket_manager = bucket.S3BucketManager(self.s3_client)
        self.object_manager = object.S3ObjectManager(
            self.s3_client, self.bucket_manager
        )

    def teardown_method(self, method):
        """
        Tear down method for test cleanup
        """
        self.__cleanup_bucket()
        self.bucket_name = ""

    @pytest.mark.s3
    def test_delete_empty_bucket(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete an empty S3 bucket
            Priority: High
            Components: [bucket]
            Steps:
                - Create a new S3 bucket
                - Verify bucket is created successfully
                - Delete the empty bucket
                - Verify bucket is deleted successfully with 204 status
        """
        self.bucket_name = f"test-bucket-delete-empty-{get_unique_id()}"
        self.__create_bucket(upload_objects=False)

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)

        metadata = validate_response_status(delete_response, HTTPStatus.NO_CONTENT)
        self.__validate_delete_bucket_headers(metadata["HTTPHeaders"])

        # Verify bucket no longer exists
        self.__verify_bucket_deleted()
        self.bucket_name = ""  # Mark as cleaned up

    @pytest.mark.s3
    def test_delete_bucket_after_deleting_all_objects(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete bucket after removing all objects
            Priority: High
            Components: [bucket, object]
            Steps:
                - Create a new S3 bucket
                - Upload multiple objects to the bucket
                - Delete all objects from the bucket
                - Delete the empty bucket
                - Verify bucket is deleted successfully
        """
        self.bucket_name = f"test-bucket-delete-after-cleanup-{get_unique_id()}"
        self.__create_bucket(upload_objects=True)
        self.__delete_objects_in_bucket()

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        # Verify bucket no longer exists
        self.__verify_bucket_deleted()
        self.bucket_name = ""

    @pytest.mark.s3
    @pytest.mark.tags
    def test_delete_bucket_with_tags(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete bucket that has tags
            Priority: Medium
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket
                - Add tags to the bucket
                - Delete the bucket
                - Verify bucket is deleted successfully
        """
        self.bucket_name = f"test-bucket-delete-with-tags-{get_unique_id()}"
        self.__create_bucket(upload_objects=False)

        # Add tags
        tag_set = {"Environment": "Test", "Purpose": "Deletion"}
        self.bucket_manager.put_bucket_tagging(self.bucket_name, tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        self.__verify_bucket_deleted()
        self.bucket_name = ""

    @pytest.mark.s3
    @pytest.mark.object_lock
    def test_delete_bucket_with_object_lock_enabled(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete empty bucket with object lock enabled
            Priority: Medium
            Components: [bucket, object_lock]
            Steps:
                - Create a new S3 bucket with object lock enabled
                - Verify object lock is enabled
                - Delete the empty bucket
                - Verify bucket is deleted successfully
        """
        self.bucket_name = f"test-bucket-delete-objlock-{get_unique_id()}"
        self.__create_bucket(enable_object_lock=True)

        validate_object_lock_configuration(
            self.bucket_manager, self.bucket_name, "Enabled"
        )

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        self.__verify_bucket_deleted()
        self.bucket_name = ""

    @pytest.mark.s3
    @pytest.mark.versioning
    def test_delete_bucket_with_versioning_enabled(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete empty bucket with versioning enabled
            Priority: Medium
            Components: [bucket, versioning]
            Steps:
                - Create a new S3 bucket
                - Enable versioning on the bucket
                - Delete the empty bucket
                - Verify bucket is deleted successfully
        """
        self.bucket_name = f"test-bucket-delete-versioning-{get_unique_id()}"
        self.__create_bucket()

        # Enable versioning
        status = "Enabled"
        self.bucket_manager.put_bucket_versioning(self.bucket_name, status)
        validate_bucket_versioning(self.bucket_manager, self.bucket_name, status)

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        self.__verify_bucket_deleted()
        self.bucket_name = ""

    @pytest.mark.s3
    @pytest.mark.lifecycle
    @pytest.mark.skip(
        reason="An error occurred (MissingContentMD5) when calling the PutBucketLifecycleConfiguration operation"
    )
    def test_delete_bucket_with_lifecycle_configuration(self):
        """
        Args:
        Metadata:
            Summary: Test case to delete bucket with lifecycle configuration
            Priority: Low
            Components: [bucket, lifecycle]
            Steps:
                - Create a new S3 bucket
                - Add lifecycle configuration
                - Delete the bucket
                - Verify bucket is deleted successfully
        """
        self.bucket_name = f"test-bucket-delete-lifecycle-{get_unique_id()}"
        self.__create_bucket()

        self.bucket_manager.put_bucket_lifecycle_configuration(
            self.bucket_name, self.bucket_manager.get_dummy_lifecycle_configuration()
        )

        # Delete bucket
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        self.__verify_bucket_deleted()
        self.bucket_name = ""

    @pytest.mark.s3
    def test_delete_bucket_multiple_times(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify idempotent delete behavior
            Priority: Medium
            Components: [bucket]
            Steps:
                - Create a new S3 bucket
                - Delete the bucket successfully
                - Attempt to delete the same bucket again
                - Verify appropriate error (NoSuchBucket) is returned
        """
        self.bucket_name = f"test-bucket-delete-multiple-{get_unique_id()}"
        self.__create_bucket()

        # Delete bucket first time
        delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
        validate_response_status(delete_response, HTTPStatus.NO_CONTENT)

        # Attempt to delete again
        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised NoSuchBucket error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.NOT_FOUND, "NoSuchBucket")

        self.bucket_name = ""

    @pytest.mark.s3
    def test_delete_non_existent_bucket(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when deleting a bucket that doesn't exist
            Priority: High
            Components: [bucket]
            Steps:
                - Attempt to delete a bucket that doesn't exist
                - Verify NoSuchBucket error is returned with 404 status
        """
        self.bucket_name = f"test-bucket-non-existent-{get_unique_id()}"

        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised NoSuchBucket error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.NOT_FOUND, "NoSuchBucket")

        self.bucket_name = ""

    @pytest.mark.s3
    def test_delete_bucket_with_objects(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when deleting bucket with objects
            Priority: High
            Components: [bucket, object]
            Steps:
                - Create a new S3 bucket
                - Upload objects to the bucket
                - Attempt to delete the bucket without removing objects
                - Verify BucketNotEmpty error is returned
        """
        self.bucket_name = f"test-bucket-delete-not-empty-{get_unique_id()}"
        self.__create_bucket(upload_objects=True)

        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised BucketNotEmpty error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.CONFLICT, "BucketNotEmpty")

    @pytest.mark.s3
    @pytest.mark.versioning
    def test_delete_bucket_with_versioned_objects(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when deleting bucket with versioned objects
            Priority: High
            Components: [bucket, versioning]
            Steps:
                - Create a bucket with versioning enabled
                - Upload objects (creating versions)
                - Attempt to delete the bucket
                - Verify BucketNotEmpty error is returned
        """
        self.bucket_name = f"test-bucket-delete-versioned-{get_unique_id()}"
        self.__create_bucket(enable_object_lock=True)
        upload_object_versions(
            self.object_manager,
            self.bucket_name,
            "versioned-file.txt",
        )

        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised BucketNotEmpty error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.CONFLICT, "BucketNotEmpty")

    @pytest.mark.s3
    @pytest.mark.versioning
    def test_delete_bucket_with_delete_markers(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when deleting bucket with delete markers
            Priority: Medium
            Components: [bucket, versioning, object]
            Steps:
                - Create a bucket with versioning enabled
                - Upload and delete objects (creating delete markers)
                - Attempt to delete the bucket
                - Verify BucketNotEmpty error is returned
        """
        self.bucket_name = f"test-bucket-delete-markers-{get_unique_id()}"
        self.__create_bucket(enable_object_lock=True)
        object_key = "file-to-delete.txt"
        upload_object_versions(
            self.object_manager,
            self.bucket_name,
            object_key,
            count=1,
        )

        # Delete object (creates delete marker)
        self.object_manager.delete_object(self.bucket_name, object_key)

        # Verify delete marker exists
        versions_response = self.object_manager.list_object_versions(self.bucket_name)
        assert (
            len(versions_response.get("DeleteMarkers", [])) > 0
        ), "Should have delete markers"

        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised BucketNotEmpty error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.CONFLICT, "BucketNotEmpty")

    @pytest.mark.s3
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_delete_bucket_with_empty_name(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when bucket name is empty
            Priority: Medium
            Components: [bucket]
            Steps:
                - Attempt to delete a bucket with empty name
                - Verify appropriate error is returned
        """
        self.bucket_name = ""

        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised error for empty bucket name")
        except Exception as e:
            validate_error_response(
                e, HTTPStatus.METHOD_NOT_ALLOWED, "MethodNotAllowed"
            )

    @pytest.mark.s3
    @pytest.mark.multipartupload
    @pytest.mark.skip(reason="TC is failing. Needs rework.")
    def test_delete_bucket_with_multipart_uploads(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when bucket has incomplete multipart uploads
            Priority: Medium
            Components: [bucket, multipartupload]
            Steps:
                - Create a bucket
                - Initiate a multipart upload without completing it
                - Attempt to delete the bucket
                - Verify BucketNotEmpty error is returned
        """
        self.bucket_name = f"test-bucket-delete-multipart-{get_unique_id()}"
        self.__create_bucket()

        # Initiate multipart upload
        object_key = "multipart-file.bin"
        multipart_response = self.object_manager.create_multipart_upload(
            self.bucket_name, object_key
        )
        validate_response_status(multipart_response, HTTPStatus.OK)

        upload_id = multipart_response["UploadId"]
        print("Initiated multipart upload with UploadId:", upload_id)
        uploadpart_response = self.object_manager.upload_part(
            self.bucket_name, object_key, upload_id, 1, b"part1 data"
        )
        validate_response_status(uploadpart_response, HTTPStatus.OK)
        print("Uploaded part ETag:", uploadpart_response["ETag"])

        # Verify multipart upload exists
        uploads_response = self.object_manager.list_multipart_uploads(self.bucket_name)
        print(f"Multipart uploads: {uploads_response}")
        assert (
            len(uploads_response.get("Uploads", [])) == 1
        ), "Should have 1 multipart upload"

        # Attempt to delete bucket
        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail("Should have raised BucketNotEmpty error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.CONFLICT, "BucketNotEmpty")
        finally:
            # Cleanup: abort multipart upload
            try:
                self.object_manager.abort_multipart_upload(
                    self.bucket_name, object_key, upload_id
                )
            except:
                pass

    @pytest.mark.s3
    def test_delete_bucket_name_case_sensitivity(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify bucket names are case-sensitive
            Priority: Low
            Components: [bucket]
            Steps:
                - Create a bucket with lowercase name
                - Attempt to delete with different case
                - Verify NoSuchBucket error is returned
        """
        self.bucket_name = f"test-bucket-case-{get_unique_id()}"

        # Create bucket
        self.__create_bucket()

        # Attempt to delete with uppercase (should fail)
        uppercase_name = self.bucket_name.upper()
        try:
            self.bucket_manager.delete_bucket(uppercase_name)
            pytest.fail("Should have raised NoSuchBucket or InvalidBucketName error")
        except Exception as e:
            # Bucket names must be lowercase, so uppercase will fail validation
            validate_error_response(e, HTTPStatus.NOT_FOUND, "NoSuchBucket")

    @pytest.mark.s3
    @pytest.mark.skip(reason="Feature not implemented yet")
    def test_delete_bucket_without_permission(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when user lacks delete permission
            Priority: Medium
            Components: [bucket, permissions]
            Steps:
                - Create a bucket
                - Use credentials without delete permission
                - Attempt to delete the bucket
                - Verify AccessDenied error is returned
        """
        self.bucket_name = f"test-bucket-no-delete-perm-{get_unique_id()}"

        # This test requires a separate client with restricted permissions
        # Implementation depends on your IAM setup
        try:
            # restricted_client.delete_bucket(Bucket=self.bucket_name)
            pytest.skip("Requires IAM permissions setup")
        except Exception as e:
            validate_error_response(e, HTTPStatus.FORBIDDEN, "AccessDenied")

    # ========== HELPER METHODS ==========

    def __create_bucket(self, enable_object_lock=False, upload_objects=False):
        """
        Helper function to create a bucket for testing
        Args:
            enable_object_lock(bool): Whether to enable object lock
            upload_objects(bool): Whether to upload test objects
        """
        # Create bucket
        create_response = self.bucket_manager.create_bucket(
            self.bucket_name,
            ObjectLockEnabledForBucket=enable_object_lock,
        )
        validate_response_status(create_response, HTTPStatus.OK)

        if upload_objects:
            object_keys = ["file1.txt", "file2.txt", "folder/file3.txt"]
            for key in object_keys:
                self.object_manager.put_object(self.bucket_name, key, b"test content")

            # Verify objects exist
            list_response = self.object_manager.list_objects(self.bucket_name)
            assert list_response.get("KeyCount", 0) == len(
                object_keys
            ), f"Should have {len(object_keys)} objects"

    def __delete_objects_in_bucket(self):
        """
        Helper function to delete all objects in the bucket
        """
        self.object_manager.delete_all_objects(self.bucket_name)

        # Verify bucket is empty
        list_response = self.object_manager.list_objects(self.bucket_name)
        assert list_response.get("KeyCount", 0) == 0, "Bucket should be empty"

    def __validate_delete_bucket_headers(self, headers):
        """
        This function validates the headers in the delete bucket response.
        Args:
            headers(dict): Response headers
        Raises:
            AssertionError: If validation fails
        """
        # Delete bucket response should have minimal headers
        assert "date" in headers, "Response headers should contain 'date'"
        assert (
            "server" in headers or "x-amz-request-id" in headers
        ), "Response should contain server or request-id header"

    def __verify_bucket_deleted(self):
        """
        Verify that the bucket no longer exists
        Raises:
            AssertionError: If bucket still exists
        """
        try:
            self.bucket_manager.delete_bucket(self.bucket_name)
            pytest.fail(f"Bucket {self.bucket_name} should not exist after deletion")
        except Exception as e:
            validate_error_response(e, HTTPStatus.NOT_FOUND, "NoSuchBucket")

    def __cleanup_bucket(self):
        """
        Cleanup function to delete the created bucket and its contents
        """
        if self.bucket_name == "":
            return

        try:
            # First, delete all objects and versions
            try:
                # Delete all object versions
                versions_response = self.object_manager.list_object_versions(
                    self.bucket_name
                )

                # Delete versions
                for version in versions_response.get("Versions", []):
                    self.object_manager.delete_object(
                        self.bucket_name,
                        version["Key"],
                        VersionId=version["VersionId"],
                    )

                # Delete delete markers
                for marker in versions_response.get("DeleteMarkers", []):
                    self.object_manager.delete_object(
                        self.bucket_name,
                        marker["Key"],
                        VersionId=marker["VersionId"],
                    )
            except:
                # If versioning not enabled, just delete objects
                try:
                    objects_response = self.object_manager.list_object_versions(
                        self.bucket_name
                    )
                    for obj in objects_response.get("Contents", []):
                        self.object_manager.delete_object(self.bucket_name, obj["Key"])
                except:
                    pass

            # Abort any incomplete multipart uploads
            try:
                uploads_response = self.object_manager.list_multipart_uploads(
                    self.bucket_name
                )
                for upload in uploads_response.get("Uploads", []):
                    self.object_manager.abort_multipart_upload(
                        Bucket=self.bucket_name,
                        Key=upload["Key"],
                        UploadId=upload["UploadId"],
                    )
            except:
                pass

            # Finally, delete the bucket
            delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
            assert (
                delete_response["ResponseMetadata"]["HTTPStatusCode"] == 204
            ), "Delete bucket should return 204 No Content"
        except Exception as e:
            print(f"Warning: Failed to clean up bucket {self.bucket_name}: {str(e)}")
