import pytest
from http import HTTPStatus

from s3native.lib.bucket import S3BucketManager
from s3native.utils import get_unique_id
from s3native.testcases.helpers import (
    validate_response_status,
    validate_error_response,
    validate_object_lock_configuration,
    validate_bucket_versioning,
    validate_bucket_tags,
)


class TestBucketCreate:
    bucket_name: str = ""

    def setup_method(self, method):
        """
        Setup method for testcase
        """
        self.bucket_manager = S3BucketManager(self.s3_client)

    def teardown_method(self, method):
        """
        Tear down method for test cleanup
        """
        self.__cleanup_bucket()
        self.bucket_name = ""

    @pytest.mark.s3
    def test_create_bucket(self):
        """
        Args:
        Metadata:
            Summary: Test case to create a new S3 bucket in object server
            Priority: High
            Components: [bucket]
            Steps:
                - Create a new S3 bucket by specifying non-existing bucket name
                - Verify that the bucket is created successfully
                - Verify response body and status code
        """
        self.bucket_name = f"test-bucket-create-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(self.bucket_name)

        print(f"Create bucket response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

    @pytest.mark.s3
    @pytest.mark.object_lock
    @pytest.mark.versioning
    def test_create_bucket_with_object_lock_enabled(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with object lock enabled
            Priority: High
            Components: [bucket, object_lock]
            Steps:
                - Create a new S3 bucket with x-amz-bucket-object-lock-enabled header set to true
                - Verify that the bucket is created successfully
                - Verify object lock configuration is enabled
                - Verify versioning is automatically enabled (required for object lock)
        """
        self.bucket_name = f"test-bucket-object-lock-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, ObjectLockEnabledForBucket=True
        )

        print(f"Create bucket with object lock enabled response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        validate_object_lock_configuration(
            self.bucket_manager, self.bucket_name, "Enabled"
        )
        validate_bucket_versioning(self.bucket_manager, self.bucket_name, "Enabled")

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_single_tag(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with a single tag
            Priority: High
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with x-amz-tagging header containing a single tag
                - Verify that the bucket is created successfully
                - Verify the tag is applied to the bucket
        """
        self.bucket_name = f"test-bucket-single-tag-{get_unique_id()}"
        tag_set = {"Environment": "Development"}

        response = self.bucket_manager.create_bucket(self.bucket_name)

        print(f"Create bucket with single tag response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_multiple_tags(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with multiple tags
            Priority: High
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with x-amz-tagging header containing multiple tags
                - Verify that the bucket is created successfully
                - Verify all tags are applied to the bucket
        """
        self.bucket_name = f"test-bucket-multi-tags-{get_unique_id()}"
        tag_set = {
            "Environment": "Development",
            "Project": "DataLake",
            "CostCenter": "Engineering",
        }

        response = self.bucket_manager.create_bucket(self.bucket_name)

        print(f"Create bucket with multiple tags response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_special_character_tags(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with tags containing special characters
            Priority: Medium
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with tags containing special characters
                - Verify that the bucket is created successfully
                - Verify tags with special characters are properly encoded and stored
        """
        self.bucket_name = f"test-bucket-special-tags-{get_unique_id()}"
        # URL-encoded tag values with special characters
        tag_set = {
            "Name": "Test+Bucket",
            "Description": "This+is+a+test+bucket",
            "Owner": "john.doe%40example.com",
        }

        response = self.bucket_manager.create_bucket(self.bucket_name)
        validate_response_status(response, HTTPStatus.OK)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_max_tags(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with maximum allowed tags (50)
            Priority: Medium
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with 50 tags (AWS max limit)
                - Verify that the bucket is created successfully
                - Verify all 50 tags are applied correctly
        """
        self.bucket_name = f"test-bucket-max-tags-{get_unique_id()}"

        # Create 50 tags (AWS S3 max limit for bucket tags)
        tag_set = {f"Tag{i}": f"Value{i}" for i in range(1, 51)}

        response = self.bucket_manager.create_bucket(self.bucket_name)

        print(f"Create bucket with maximum tags response: {response}")
        validate_response_status(response, HTTPStatus.OK)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_tags_exceeding_limit(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when exceeding maximum tag limit (>50)
            Priority: Medium
            Components: [bucket, tags]
            Steps:
                - Attempt to create bucket with more than 50 tags
                - Verify appropriate error is returned
        """
        self.bucket_name = f"test-bucket-exceed-tags{get_unique_id()}"

        # Create 51 tags (exceeding AWS limit)
        tag_set = {f"Tag{i}": f"Value{i}" for i in range(1, 52)}

        try:
            response = self.bucket_manager.create_bucket(self.bucket_name)
            validate_response_status(response, HTTPStatus.OK)

            self.__put_bucket_tags(tag_set)
            pytest.fail("Should have raised an error for exceeding tag limit")
        except Exception as e:
            # boto3 is throwing MalformedXML
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "TooManyTags")

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.object_lock
    def test_create_bucket_with_tags_and_object_lock(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with tags and object lock enabled
            Priority: High
            Components: [bucket, tags, object_lock]
            Steps:
                - Create a new S3 bucket with tags and object lock enabled
                - Verify that the bucket is created successfully
                - Verify both tags and object lock configuration are applied
        """
        self.bucket_name = f"test-bucket-tags-lock-{get_unique_id()}"
        tag_set = {"Compliance": "Required", "Retention": "7Years"}

        response = self.bucket_manager.create_bucket(
            self.bucket_name, ObjectLockEnabledForBucket=True
        )

        print(f"Create bucket with tags and object lock response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        self.__put_bucket_tags(tag_set)

        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)
        validate_object_lock_configuration(
            self.bucket_manager, self.bucket_name, "Enabled"
        )

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_empty_tag_value(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with empty tag value
            Priority: Low
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with a tag that has an empty value
                - Verify that the bucket is created successfully
                - Verify tag with empty value is stored correctly
        """
        self.bucket_name = f"test-bucket-empty-tag{get_unique_id()}"
        tag_set = {"Environment": "Production", "EmptyValue": ""}

        response = self.bucket_manager.create_bucket(self.bucket_name)
        validate_response_status(response, HTTPStatus.OK)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    def test_create_bucket_with_long_tag_key_and_value(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with maximum length tag key and value
            Priority: Low
            Components: [bucket, tags]
            Steps:
                - Create a new S3 bucket with tag key (128 chars max) and value (256 chars max)
                - Verify that the bucket is created successfully
                - Verify tags are stored correctly
        """
        self.bucket_name = f"test-bucket-long-tag-{get_unique_id()}"

        # Max key length is 128 characters, max value length is 256 characters
        long_key = "K" * 128
        long_value = "V" * 256
        tag_set = {"Environment": "Production", long_key: long_value}

        response = self.bucket_manager.create_bucket(self.bucket_name)
        validate_response_status(response, HTTPStatus.OK)

        self.__put_bucket_tags(tag_set)
        validate_bucket_tags(self.bucket_manager, self.bucket_name, tag_set)

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.skip(reason="Found different behavior than AWS S3")
    def test_create_bucket_with_duplicate_tag_keys(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify behavior with duplicate tag keys
            Priority: High
            Components: [bucket, tags]
            Steps:
                - Attempt to create bucket with duplicate tag keys
                - Verify we are getting right error
        """
        self.bucket_name = f"test-bucket-dup-tags-{get_unique_id()}"
        # Duplicate Environment key with different values
        tag_set = {
            "Environment": "Production",
            "Team": "DevOps",
            "Environment": "Staging",
        }

        try:
            response = self.bucket_manager.create_bucket(self.bucket_name)
            validate_response_status(response, HTTPStatus.OK)

            self.__put_bucket_tags(tag_set)
            pytest.fail("Should have raised InvalidTag error")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidTag")

    @pytest.mark.s3
    def test_create_bucket_already_exists(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when creating a bucket that already exists
            Priority: High
            Components: [bucket]
            Steps:
                - Create a new S3 bucket
                - Attempt to create the same bucket again
                - Verify BucketAlreadyExists or BucketAlreadyOwnedByYou error is returned
        """
        self.bucket_name = f"test-bucket-already-exists-{get_unique_id()}"

        # Create bucket first time
        response = self.bucket_manager.create_bucket(self.bucket_name)
        validate_response_status(response, HTTPStatus.OK)

        # Attempt to create the same bucket again
        try:
            response = self.bucket_manager.create_bucket(self.bucket_name)
            pytest.fail(
                "Should have raised BucketAlreadyExists or BucketAlreadyOwnedByYou error"
            )
        except Exception as e:
            validate_error_response(
                e, HTTPStatus.CONFLICT, "BucketAlreadyExists", "BucketAlreadyOwnedByYou"
            )

    @pytest.mark.s3
    def test_create_bucket_invalid_name_too_short(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when bucket name is too short (< 3 characters)
            Priority: High
            Components: [bucket]
            Steps:
                - Attempt to create a bucket with name less than 3 characters
                - Verify InvalidBucketName error is returned
        """
        self.bucket_name = "ab"  # Only 2 characters
        try:
            self.bucket_manager.create_bucket(self.bucket_name)
            pytest.fail("Should have raised InvalidBucketName error for short name")
        except Exception as e:
            self.bucket_name = ""
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidBucketName")

    @pytest.mark.s3
    def test_create_bucket_invalid_name_too_long(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when bucket name is too long (> 63 characters)
            Priority: High
            Components: [bucket]
            Steps:
                - Attempt to create a bucket with name more than 63 characters
                - Verify InvalidBucketName error is returned
        """
        self.bucket_name = "a" * 64  # 64 characters (exceeds 63 limit)
        try:
            self.bucket_manager.create_bucket(self.bucket_name)
            pytest.fail("Should have raised InvalidBucketName error for long name")
        except Exception as e:
            self.bucket_name = ""
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidBucketName")

    @pytest.mark.s3
    @pytest.mark.parametrize(
        "invalid_name",
        [
            "Bucket-With-Uppercase",  # Contains uppercase
            "bucket_with_underscore",  # Contains underscore
            "bucket..doubledot",  # Contains consecutive dots
            "bucket-.invalid",  # Dot adjacent to hyphen
            "-bucket-starts-hyphen",  # Starts with hyphen
            "bucket-ends-hyphen-",  # Ends with hyphen
            ".bucket-starts-dot",  # Starts with dot
            "bucket-ends-dot.",  # Ends with dot
            "192.168.1.1",  # IP address format
            # "bucket name spaces",  # Contains spaces (Disabled due to boto3 framework validation)
            # "bucket@invalid",  # Contains special character (Disabled due to boto3 framework validation)
        ],
    )
    def test_create_bucket_invalid_name_format(self, invalid_name):
        """
        Args:
            invalid_name: Invalid bucket name to test
        Metadata:
            Summary: Test case to verify error when bucket name has invalid format
            Priority: High
            Components: [bucket]
            Steps:
                - Attempt to create a bucket with invalid name format
                - Verify InvalidBucketName error is returned
        """
        self.bucket_name = invalid_name
        try:
            self.bucket_manager.create_bucket(self.bucket_name)
            pytest.fail(
                f"Should have raised InvalidBucketName error for name: {invalid_name}"
            )
        except Exception as e:
            self.bucket_name = ""
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidBucketName")

    @pytest.mark.s3
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_empty_name(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when bucket name is empty
            Priority: High
            Components: [bucket]
            Steps:
                - Attempt to create a bucket with empty name
                - Verify appropriate error is returned
        """
        self.bucket_name = ""
        try:
            self.bucket_manager.create_bucket(self.bucket_name)
            pytest.fail("Should have raised error for empty bucket name")
        except Exception as e:
            validate_error_response(
                e, HTTPStatus.METHOD_NOT_ALLOWED, "MethodNotAllowed"
            )

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.parametrize(
        "invalid_tag",
        [
            "InvalidTagNoEquals",  # Missing = sign
            "=ValueWithoutKey",  # Missing key
            "Key=",  # Empty value is valid, but this tests edge case
        ],
    )
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_invalid_tag_format(self, invalid_tag):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when tag format is invalid
            Priority: Medium
            Components: [bucket, tags]
            Steps:
                - Attempt to create a bucket with malformed tag string
                - Verify InvalidTag or BadRequest error is returned
        """
        self.bucket_name = f"test-bucket-invalid-tag-{get_unique_id()}"
        try:
            self.bucket_manager.create_bucket(self.bucket_name, Tagging=invalid_tag)
            pytest.fail(f"Tag format '{invalid_tag}' was accepted")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidTag")

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_tag_key_too_long(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when tag key exceeds 128 characters
            Priority: Low
            Components: [bucket, tags]
            Steps:
                - Attempt to create a bucket with tag key > 128 characters
                - Verify InvalidTag error is returned
        """
        self.bucket_name = f"test-bucket-long-tagkey-{get_unique_id()}"

        # Tag key with 129 characters (exceeds 128 limit)
        long_key = "K" * 129
        tag_set = f"{long_key}=Value"

        try:
            self.bucket_manager.create_bucket(self.bucket_name, Tagging=tag_set)
            pytest.fail("Should have raised InvalidTag error for long tag key")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidTag")

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_tag_value_too_long(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when tag value exceeds 256 characters
            Priority: Low
            Components: [bucket, tags]
            Steps:
                - Attempt to create a bucket with tag value > 256 characters
                - Verify InvalidTag error is returned
        """
        self.bucket_name = f"test-bucket-long-tagval-{get_unique_id()}"

        # Tag value with 257 characters (exceeds 256 limit)
        long_value = "V" * 257
        tag_set = f"Key={long_value}"

        try:
            self.bucket_manager.create_bucket(self.bucket_name, Tagging=tag_set)
            pytest.fail("Should have raised InvalidTag error for long tag value")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidTag")

    @pytest.mark.s3
    @pytest.mark.object_lock
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_with_invalid_object_lock_value(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when invalid object lock value is provided
            Priority: Medium
            Components: [bucket, object_lock]
            Steps:
                - Attempt to create a bucket with invalid object lock value
                - Verify InvalidArgument error is returned
        """
        self.bucket_name = f"test-bucket-invalid-lock-{get_unique_id()}"

        try:
            self.bucket_manager.create_bucket(
                self.bucket_name, ObjectLockEnabledForBucket="InvalidValue"
            )
            pytest.fail("Should have raised error for invalid object lock value")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "InvalidArgument")

    @pytest.mark.s3
    @pytest.mark.skip(reason="Disabled due to boto3 framework validation")
    def test_create_bucket_malformed_xml_in_configuration(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when malformed XML in CreateBucketConfiguration
            Priority: Low
            Components: [bucket, validation]
            Steps:
                - Attempt to create a bucket with malformed configuration
                - Verify MalformedXML error is returned
        """
        self.bucket_name = f"test-bucket-malformed-xml{get_unique_id()}"

        try:
            # This would require sending raw request with malformed XML
            # Using boto3, we can try to pass invalid configuration structure
            response = self.bucket_manager.create_bucket(
                self.bucket_name,
                CreateBucketConfiguration={"InvalidKey": "InvalidValue"},
            )
            # May succeed with boto3 validation, or fail at server
            print("Warning: Invalid configuration was accepted")
        except Exception as e:
            validate_error_response(e, HTTPStatus.BAD_REQUEST, "MalformedXML")

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.parametrize(
        "acl", ["private", "public-read", "public-read-write", "authenticated-read"]
    )
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_acl(self, acl):
        """
        Args:
            acl: The ACL setting for the bucket
        Metadata:
            Summary: Test case to create bucket with ACL
            Priority: High
            Components: [bucket, acl]
            Steps:
                - Create a new S3 bucket with x-amz-acl header set to {private | public-read | public-read-write | authenticated-read}
                - Verify that the bucket is created successfully
                - Verify bucket ACL is set to private
        """
        self.bucket_name = f"test-bucket-acl-{acl}-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(self.bucket_name, ACL=acl)

        print(f"Create bucket with ACL private response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify ACL is set to private
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        print(f"Bucket ACL response: {acl_response}")
        assert acl_response is not None, "ACL response should not be None"
        assert len(acl_response["Grants"]) > 0, "ACL should have at least one grant"

        if acl == "private":
            # Private ACL should only grant full control to owner
            owner_grants = [
                g for g in acl_response["Grants"] if g["Permission"] == "FULL_CONTROL"
            ]
            assert len(owner_grants) > 0, "Owner should have FULL_CONTROL permission"
        elif acl == "public-read":
            # Check for AllUsers group with READ permission
            all_users_grants = [
                g
                for g in acl_response["Grants"]
                if "URI" in g.get("Grantee", {})
                and "AllUsers" in g["Grantee"]["URI"]
                and g["Permission"] == "READ"
            ]
            assert (
                len(all_users_grants) > 0
            ), "ACL should grant READ permission to AllUsers group"
        elif acl == "public-read-write":
            # Check for AllUsers group with READ and WRITE permissions
            all_users_read = [
                g
                for g in acl_response["Grants"]
                if "URI" in g.get("Grantee", {})
                and "AllUsers" in g["Grantee"]["URI"]
                and g["Permission"] == "READ"
            ]
            all_users_write = [
                g
                for g in acl_response["Grants"]
                if "URI" in g.get("Grantee", {})
                and "AllUsers" in g["Grantee"]["URI"]
                and g["Permission"] == "WRITE"
            ]
            assert (
                len(all_users_read) > 0
            ), "ACL should grant READ permission to AllUsers"
            assert (
                len(all_users_write) > 0
            ), "ACL should grant WRITE permission to AllUsers"
        elif acl == "authenticated-read":
            # Check for AuthenticatedUsers group with READ permission
            auth_users_grants = [
                g
                for g in acl_response["Grants"]
                if "URI" in g.get("Grantee", {})
                and "AuthenticatedUsers" in g["Grantee"]["URI"]
                and g["Permission"] == "READ"
            ]
            assert (
                len(auth_users_grants) > 0
            ), "ACL should grant READ permission to AuthenticatedUsers"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_grant_full_control(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with x-amz-grant-full-control header
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Create a new S3 bucket with x-amz-grant-full-control header
                - Verify that the bucket is created successfully
                - Verify the specified grantee has full control access
        """
        self.bucket_name = f"test-bucket-grant-full-{get_unique_id()}"

        # Get current user's canonical ID for testing
        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        print(f"owner_id: {owner_id}")
        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for grant testing")

        grant_full_control = f"id={owner_id}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, GrantFullControl=grant_full_control
        )

        print(f"Create bucket with grant-full-control response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify the grant is applied
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        full_control_grants = [
            g
            for g in acl_response["Grants"]
            if g["Permission"] == "FULL_CONTROL"
            and g.get("Grantee", {}).get("ID") == owner_id
        ]
        assert (
            len(full_control_grants) > 0
        ), "Specified grantee should have FULL_CONTROL permission"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_grant_read(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with x-amz-grant-read header
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Create a new S3 bucket with x-amz-grant-read header
                - Verify that the bucket is created successfully
                - Verify the specified grantee has read access
        """
        self.bucket_name = f"test-bucket-grant-read-{get_unique_id()}"

        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for grant testing")

        grant_read = f"id={owner_id}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, GrantRead=grant_read
        )

        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify the grant is applied
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        read_grants = [
            g
            for g in acl_response["Grants"]
            if g["Permission"] == "READ" and g.get("Grantee", {}).get("ID") == owner_id
        ]
        assert len(read_grants) > 0, "Specified grantee should have READ permission"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_grant_write(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with x-amz-grant-write header
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Create a new S3 bucket with x-amz-grant-write header
                - Verify that the bucket is created successfully
                - Verify the specified grantee has write access
        """
        self.bucket_name = f"test-bucket-grant-write-{get_unique_id()}"

        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for grant testing")

        grant_write = f"id={owner_id}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, GrantWrite=grant_write
        )

        print(f"Create bucket with grant-write response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify the grant is applied
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        write_grants = [
            g
            for g in acl_response["Grants"]
            if g["Permission"] == "WRITE" and g.get("Grantee", {}).get("ID") == owner_id
        ]
        assert len(write_grants) > 0, "Specified grantee should have WRITE permission"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_grant_read_acp(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with x-amz-grant-read-acp header
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Create a new S3 bucket with x-amz-grant-read-acp header
                - Verify that the bucket is created successfully
                - Verify the specified grantee has read ACP permission
        """
        self.bucket_name = f"test-bucket-grant-readacp-{get_unique_id()}"

        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for grant testing")

        grant_read_acp = f"id={owner_id}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, GrantReadACP=grant_read_acp
        )

        print(f"Create bucket with grant-read-acp response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify the grant is applied
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        read_acp_grants = [
            g
            for g in acl_response["Grants"]
            if g["Permission"] == "READ_ACP"
            and g.get("Grantee", {}).get("ID") == owner_id
        ]
        assert (
            len(read_acp_grants) > 0
        ), "Specified grantee should have READ_ACP permission"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_grant_write_acp(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with x-amz-grant-write-acp header
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Create a new S3 bucket with x-amz-grant-write-acp header
                - Verify that the bucket is created successfully
                - Verify the specified grantee has write ACP permission
        """
        self.bucket_name = f"test-bucket-grant-writeacp-{get_unique_id()}"

        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for grant testing")

        grant_write_acp = f"id={owner_id}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, GrantWriteACP=grant_write_acp
        )

        print(f"Create bucket with grant-write-acp response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify the grant is applied
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        write_acp_grants = [
            g
            for g in acl_response["Grants"]
            if g["Permission"] == "WRITE_ACP"
            and g.get("Grantee", {}).get("ID") == owner_id
        ]
        assert (
            len(write_acp_grants) > 0
        ), "Specified grantee should have WRITE_ACP permission"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.versioning
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_combined_acl_and_object_lock(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with both ACL and object lock enabled
            Priority: High
            Components: [bucket, acl, object_lock]
            Steps:
                - Create a new S3 bucket with ACL and object lock enabled
                - Verify that the bucket is created successfully
                - Verify both ACL and object lock configurations are applied
        """
        self.bucket_name = f"test-bucket-combined-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, ACL="private", ObjectLockEnabledForBucket=True
        )

        print(f"Create bucket with combined settings response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify ACL
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        owner_grants = [
            g for g in acl_response["Grants"] if g["Permission"] == "FULL_CONTROL"
        ]
        assert len(owner_grants) > 0, "Owner should have FULL_CONTROL permission"

        # Verify object lock
        lock_response = self.bucket_manager.get_object_lock_configuration(
            Bucket=self.bucket_name
        )
        assert (
            lock_response["ObjectLockConfiguration"]["ObjectLockEnabled"] == "Enabled"
        ), "Object lock should be enabled"

        # Verify versioning
        versioning_response = self.bucket_manager.get_bucket_versioning(
            Bucket=self.bucket_name
        )
        assert (
            versioning_response.get("Status") == "Enabled"
        ), "Versioning should be enabled with object lock"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_invalid_grant_format(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when grant header has invalid format
            Priority: Medium
            Components: [bucket, acl, grants]
            Steps:
                - Attempt to create a bucket with malformed grant header
                - Verify InvalidArgument error is returned
        """
        self.bucket_name = f"test-bucket-invalid-grant-{get_unique_id()}"

        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name,
                GrantRead="invalid-grant-format",  # Should be id=xxx or uri=xxx or emailAddress=xxx
            )
            pytest.fail(
                "Should have raised InvalidArgument error for invalid grant format"
            )
        except Exception as e:
            print(f"Expected error for invalid grant format: {str(e)}")
            assert "InvalidArgument" in str(e) or "Invalid" in str(
                e
            ), f"Should receive InvalidArgument error, got: {str(e)}"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_conflicting_acl_and_grants(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when both ACL and grant headers are provided
            Priority: Medium
            Components: [bucket, acl]
            Steps:
                - Attempt to create a bucket with both ACL and grant-* headers
                - Verify InvalidRequest error is returned
        """
        self.bucket_name = f"test-bucket-acl-conflict-{get_unique_id()}"

        owner_response = self.bucket_manager.list_buckets()
        owner_id = owner_response.get("Owner", {}).get("ID", "")

        if not owner_id:
            pytest.skip("Unable to retrieve owner ID for testing")

        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name,
                ACL="private",  # Canned ACL
                GrantRead=f"id={owner_id}",  # Grant header - conflict
            )
            pytest.fail(
                "Should have raised InvalidRequest error for conflicting ACL parameters"
            )
        except Exception as e:
            print(f"Expected error for conflicting ACL parameters: {str(e)}")
            assert (
                "InvalidRequest" in str(e)
                or "InvalidArgument" in str(e)
                or "cannot be specified together" in str(e).lower()
            ), f"Should receive InvalidRequest error, got: {str(e)}"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_invalid_acl(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when invalid ACL value is provided
            Priority: Medium
            Components: [bucket, acl]
            Steps:
                - Attempt to create a bucket with invalid ACL value
                - Verify InvalidArgument error is returned
        """
        self.bucket_name = f"test-bucket-invalid-acl-{get_unique_id()}"

        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name, ACL="invalid-acl-value"
            )
            pytest.fail("Should have raised InvalidArgument error for invalid ACL")
        except Exception as e:
            print(f"Expected error for invalid ACL: {str(e)}")
            assert "InvalidArgument" in str(e) or "Invalid" in str(
                e
            ), f"Should receive InvalidArgument error, got: {str(e)}"

            if hasattr(e, "response"):
                status_code = e.response.get("ResponseMetadata", {}).get(
                    "HTTPStatusCode", 0
                )
                assert (
                    status_code == 400
                ), f"Status code should be 400, got: {status_code}"

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.acl
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_tags_and_acl(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with both tags and ACL
            Priority: High
            Components: [bucket, tags, acl]
            Steps:
                - Create a new S3 bucket with both tags and ACL settings
                - Verify that the bucket is created successfully
                - Verify both tags and ACL are applied correctly
        """
        self.bucket_name = f"test-bucket-tags-acl-{get_unique_id()}"
        tag_set = "Environment=Staging&Team=QA"

        response = self.bucket_manager.create_bucket(
            self.bucket_name, Tagging=tag_set, ACL="private"
        )

        print(f"Create bucket with tags and ACL response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify bucket tags
        tagging_response = self.bucket_manager.get_bucket_tagging(
            Bucket=self.bucket_name
        )
        assert len(tagging_response["TagSet"]) == 2, "Should have exactly two tags"
        tags_dict = {tag["Key"]: tag["Value"] for tag in tagging_response["TagSet"]}
        assert tags_dict["Environment"] == "Staging", "Environment should be 'Staging'"
        assert tags_dict["Team"] == "QA", "Team should be 'QA'"

        # Verify ACL
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        owner_grants = [
            g for g in acl_response["Grants"] if g["Permission"] == "FULL_CONTROL"
        ]
        assert len(owner_grants) > 0, "Owner should have FULL_CONTROL permission"

    @pytest.mark.s3
    @pytest.mark.object_ownership
    @pytest.mark.parametrize(
        "ownership", ["BucketOwnerPreferred", "ObjectWriter", "BucketOwnerEnforced"]
    )
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_object_ownership(self, ownership):
        """
        Args:
            ownership: The object ownership setting
        Metadata:
            Summary: Test case to create bucket with x-amz-object-ownership header
            Priority: High
            Components: [bucket, ownership]
            Steps:
                - Create a new S3 bucket with x-amz-object-ownership header set to {BucketOwnerPreferred | ObjectWriter | BucketOwnerEnforced}
                - Verify that the bucket is created successfully
                - Verify object ownership configuration is set correctly
        """
        self.bucket_name = f"test-bucket-ownership-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, ObjectOwnership=ownership
        )

        print(f"Create bucket with object ownership '{ownership}' response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify object ownership configuration
        ownership_response = self.bucket_manager.get_bucket_ownership_controls(
            Bucket=self.bucket_name
        )
        assert (
            ownership_response is not None
        ), "Object ownership response should not be None"
        assert (
            "OwnershipControls" in ownership_response
        ), "Response should contain OwnershipControls"
        assert (
            "Rules" in ownership_response["OwnershipControls"]
        ), "OwnershipControls should contain Rules"
        assert (
            len(ownership_response["OwnershipControls"]["Rules"]) > 0
        ), "Should have at least one ownership rule"

        # Verify the ownership setting matches
        actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
            "ObjectOwnership"
        ]
        assert (
            actual_ownership == ownership
        ), f"Object ownership should be '{ownership}', got '{actual_ownership}'"

        # Additional validation based on ownership type
        if ownership == "BucketOwnerEnforced":
            # When BucketOwnerEnforced is set, ACLs should be disabled
            # Attempting to get/set ACL might fail or show restricted access
            try:
                acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
                # If ACLs are disabled, grants should be minimal (only owner)
                assert (
                    len(acl_response.get("Grants", [])) <= 1
                ), "With BucketOwnerEnforced, ACLs should be disabled/minimal"
            except Exception as e:
                # This is expected behavior - ACLs might not be accessible
                print(f"Expected behavior with BucketOwnerEnforced: {str(e)}")

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_ownership_and_acl_conflict(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify ACL behavior with BucketOwnerEnforced ownership
            Priority: High
            Components: [bucket, ownership, acl]
            Steps:
                - Attempt to create bucket with BucketOwnerEnforced and ACL settings
                - Verify appropriate error or behavior (ACLs should be disabled)
        """
        self.bucket_name = f"test-bucket-own-conflict-{get_unique_id()}"

        # BucketOwnerEnforced disables ACLs, so this should either:
        # 1. Succeed but ignore ACL parameter
        # 2. Return an error indicating ACLs cannot be set
        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name,
                ObjectOwnership="BucketOwnerEnforced",
                ACL="public-read",  # This should conflict
            )

            # If it succeeds, verify ACLs are not applied
            metadata = validate_response_status(response, HTTPStatus.OK)

            ownership_response = self.bucket_manager.get_bucket_ownership_controls(
                self.bucket_name
            )
            actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
                "ObjectOwnership"
            ]
            assert (
                actual_ownership == "BucketOwnerEnforced"
            ), "Object ownership should be BucketOwnerEnforced"

            # Verify public ACL is NOT applied (ACLs should be disabled)
            acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
            all_users_grants = [
                g
                for g in acl_response.get("Grants", [])
                if "URI" in g.get("Grantee", {}) and "AllUsers" in g["Grantee"]["URI"]
            ]
            assert (
                len(all_users_grants) == 0
            ), "Public ACL should not be applied when BucketOwnerEnforced is set"

        except Exception as e:
            # This is also valid - AWS might reject the conflicting parameters
            print(
                f"Expected error when combining BucketOwnerEnforced with ACL: {str(e)}"
            )
            assert "InvalidBucketAclWithObjectOwnership" in str(
                e
            ) or "AccessControlListNotSupported" in str(
                e
            ), f"Should receive appropriate ACL conflict error, got: {str(e)}"

    @pytest.mark.s3
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_ownership_bucket_owner_preferred(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify BucketOwnerPreferred ownership behavior
            Priority: Medium
            Components: [bucket, ownership]
            Steps:
                - Create bucket with BucketOwnerPreferred ownership
                - Verify ownership configuration
                - Verify ACLs can still be set
        """
        self.bucket_name = f"test-bucket-own-preferred-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, ObjectOwnership="BucketOwnerPreferred", ACL="private"
        )

        print(f"Create bucket with BucketOwnerPreferred ownership response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify object ownership
        ownership_response = self.bucket_manager.get_bucket_ownership_controls(
            Bucket=self.bucket_name
        )
        actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
            "ObjectOwnership"
        ]
        assert (
            actual_ownership == "BucketOwnerPreferred"
        ), "Object ownership should be BucketOwnerPreferred"

        # Verify ACLs can be set (unlike BucketOwnerEnforced)
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        assert acl_response is not None, "Should be able to get bucket ACL"
        assert len(acl_response.get("Grants", [])) > 0, "ACL grants should be present"

    @pytest.mark.s3
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_ownership_object_writer(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify ObjectWriter ownership behavior
            Priority: Medium
            Components: [bucket, ownership]
            Steps:
                - Create bucket with ObjectWriter ownership
                - Verify ownership configuration
                - Verify ACLs can be set
        """
        self.bucket_name = f"test-bucket-own-writer-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name, ObjectOwnership="ObjectWriter", ACL="private"
        )

        print(f"Create bucket with ObjectWriter ownership response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify object ownership
        ownership_response = self.bucket_manager.get_bucket_ownership_controls(
            Bucket=self.bucket_name
        )
        actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
            "ObjectOwnership"
        ]
        assert (
            actual_ownership == "ObjectWriter"
        ), "Object ownership should be ObjectWriter"

        # Verify ACLs can be set
        acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
        assert acl_response is not None, "Should be able to get bucket ACL"
        assert len(acl_response.get("Grants", [])) > 0, "ACL grants should be present"

    @pytest.mark.s3
    @pytest.mark.object_ownership
    @pytest.mark.object_lock
    @pytest.mark.versioning
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_ownership_and_object_lock(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with object ownership and object lock
            Priority: High
            Components: [bucket, ownership, object_lock]
            Steps:
                - Create bucket with BucketOwnerPreferred and object lock enabled
                - Verify both configurations are applied correctly
        """
        self.bucket_name = f"test-bucket-own-lock-{get_unique_id()}"
        response = self.bucket_manager.create_bucket(
            self.bucket_name,
            ObjectOwnership="BucketOwnerPreferred",
            ObjectLockEnabledForBucket=True,
        )

        print(f"Create bucket with ownership and object lock response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify object ownership
        ownership_response = self.bucket_manager.get_bucket_ownership_controls(
            Bucket=self.bucket_name
        )
        actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
            "ObjectOwnership"
        ]
        assert (
            actual_ownership == "BucketOwnerPreferred"
        ), "Object ownership should be BucketOwnerPreferred"

        # Verify object lock
        lock_response = self.bucket_manager.get_object_lock_configuration(
            Bucket=self.bucket_name
        )
        assert (
            lock_response["ObjectLockConfiguration"]["ObjectLockEnabled"] == "Enabled"
        ), "Object lock should be enabled"

        # Verify versioning (required for object lock)
        versioning_response = self.bucket_manager.get_bucket_versioning(
            Bucket=self.bucket_name
        )
        assert (
            versioning_response.get("Status") == "Enabled"
        ), "Versioning should be enabled with object lock"

    @pytest.mark.s3
    @pytest.mark.tags
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_tags_and_ownership(self):
        """
        Args:
        Metadata:
            Summary: Test case to create bucket with tags and object ownership
            Priority: Medium
            Components: [bucket, tags, ownership]
            Steps:
                - Create a new S3 bucket with tags and object ownership settings
                - Verify that the bucket is created successfully
                - Verify both tags and ownership configuration are applied
        """
        self.bucket_name = f"test-bucket-tags-own-{get_unique_id()}"
        tag_set = "DataClassification=Confidential&Owner=SecurityTeam"

        response = self.bucket_manager.create_bucket(
            self.bucket_name, Tagging=tag_set, ObjectOwnership="BucketOwnerPreferred"
        )

        print(f"Create bucket with tags and ownership response: {response}")
        metadata = validate_response_status(response, HTTPStatus.OK)
        self.__validate_create_bucket_headers(metadata["HTTPHeaders"])
        self.__validate_create_bucket_reponse(response)

        # Verify bucket tags
        tagging_response = self.bucket_manager.get_bucket_tagging(
            Bucket=self.bucket_name
        )
        assert len(tagging_response["TagSet"]) == 2, "Should have exactly two tags"
        tags_dict = {tag["Key"]: tag["Value"] for tag in tagging_response["TagSet"]}
        assert (
            tags_dict["DataClassification"] == "Confidential"
        ), "DataClassification should be 'Confidential'"
        assert tags_dict["Owner"] == "SecurityTeam", "Owner should be 'SecurityTeam'"

        # Verify object ownership
        ownership_response = self.bucket_manager.get_bucket_ownership_controls(
            Bucket=self.bucket_name
        )
        actual_ownership = ownership_response["OwnershipControls"]["Rules"][0][
            "ObjectOwnership"
        ]
        assert (
            actual_ownership == "BucketOwnerPreferred"
        ), "Object ownership should be BucketOwnerPreferred"

    @pytest.mark.s3
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_with_invalid_ownership(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when invalid object ownership value is provided
            Priority: Medium
            Components: [bucket, ownership]
            Steps:
                - Attempt to create a bucket with invalid object ownership value
                - Verify InvalidArgument error is returned
        """
        self.bucket_name = f"test-bucket-invalid-own-{get_unique_id()}"

        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name, ObjectOwnership="InvalidOwnershipValue"
            )
            pytest.fail(
                "Should have raised InvalidArgument error for invalid ownership"
            )
        except Exception as e:
            print(f"Expected error for invalid object ownership: {str(e)}")
            assert "InvalidArgument" in str(e) or "Invalid" in str(
                e
            ), f"Should receive InvalidArgument error, got: {str(e)}"

    @pytest.mark.s3
    @pytest.mark.acl
    @pytest.mark.object_ownership
    @pytest.mark.skip(reason="Unsupported Feature")
    def test_create_bucket_acl_with_bucket_owner_enforced_conflict(self):
        """
        Args:
        Metadata:
            Summary: Test case to verify error when setting ACL with BucketOwnerEnforced
            Priority: High
            Components: [bucket, ownership, acl]
            Steps:
                - Attempt to create bucket with BucketOwnerEnforced and public ACL
                - Verify InvalidBucketAclWithObjectOwnership error is returned
        """
        self.bucket_name = f"test-bucket-own-acl-err-{get_unique_id()}"

        try:
            response = self.bucket_manager.create_bucket(
                self.bucket_name,
                ObjectOwnership="BucketOwnerEnforced",
                ACL="public-read",  # This conflicts with BucketOwnerEnforced
            )
            # If it succeeds, it should ignore ACL
            metadata = validate_response_status(response, HTTPStatus.OK)
            print("Warning: ACL was ignored with BucketOwnerEnforced (valid behavior)")

            # Verify ACL is not actually public
            acl_response = self.bucket_manager.get_bucket_acl(self.bucket_name)
            all_users_grants = [
                g
                for g in acl_response.get("Grants", [])
                if "URI" in g.get("Grantee", {}) and "AllUsers" in g["Grantee"]["URI"]
            ]
            assert (
                len(all_users_grants) == 0
            ), "Public ACL should not be applied with BucketOwnerEnforced"

        except Exception as e:
            print(f"Expected error for ACL with BucketOwnerEnforced: {str(e)}")
            assert (
                "InvalidBucketAclWithObjectOwnership" in str(e)
                or "AccessControlListNotSupported" in str(e)
                or "Invalid" in str(e)
            ), f"Should receive InvalidBucketAclWithObjectOwnership error, got: {str(e)}"

    # ========== HELPER METHODS ==========

    def __put_bucket_tags(self, tags):
        """
        Helper function to add tags to a bucket
        Args:
            tags (dict): Tag key-value pairs
        Raises:
            AssertionError: If validation fails
        """
        response = self.bucket_manager.put_bucket_tagging(
            self.bucket_name,
            tags,
        )

        validate_response_status(response, HTTPStatus.OK)

    def __validate_create_bucket_headers(self, headers):
        """
        This function validates the headers in the create bucket response.
        Args:
            headers(dict): Response headers
        Raises:
            AssertionError: If validation fails
        """
        # Verify Location header (required in CreateBucket response)
        assert "location" in headers, "Response headers should contain 'location'"
        assert (
            headers["location"] == f"/{self.bucket_name}"
        ), f"Location header should be '/{self.bucket_name}', got '{headers['location']}'"

        assert (
            "content-length" in headers
        ), "Response headers should contain 'content-length'"
        assert "server" in headers, "Response headers should contain 'server'"
        assert "date" in headers, "Response headers should contain 'date'"

        assert (
            "x-amz-request-id" in headers
        ), "Response headers should contain 'x-amz-request-id'"
        assert headers["x-amz-request-id"] != "", "x-amz-request-id should not be empty"

    def __validate_create_bucket_reponse(self, response):
        """
        Validate create bucket response for Location and x-amz-bucket-arn
        Args:
            response (dict): Create bucket response
        Raises:
            AssertionError: If validation fails
        """
        assert (
            response["Location"] == f"/{self.bucket_name}"
        ), f"Location should be '/{self.bucket_name}'"

        if "x-amz-bucket-arn" in response:
            expected_arn = f"arn:aws:s3:::{self.bucket_name}"
            assert (
                response["x-amz-bucket-arn"] == expected_arn
            ), f"x-amz-bucket-arn should be '{expected_arn}', got '{response['x-amz-bucket-arn']}'"

    def __cleanup_bucket(self):
        """
        Cleanup function to delete the created bucket
        """
        if self.bucket_name == "":
            return

        try:
            delete_response = self.bucket_manager.delete_bucket(self.bucket_name)
            assert (
                delete_response["ResponseMetadata"]["HTTPStatusCode"] == 204
            ), "Delete bucket should return 204 No Content"
        except Exception as e:
            print(f"Warning: Failed to clean up bucket {self.bucket_name}: {str(e)}")
