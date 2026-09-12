def validate_response_status(response, expected_status_code) -> dict:
    """
    This function validates the HTTP status code in the response metadata.
    Args:
        response (any): S3 Response object
        expected_status_code (int): HTTP status code expected in the response
    Returns:
        dict: Response metadata if validation is successful
    Raises:
        AssertionError: If validation fails
    """

    assert response is not None, "Response should not be None"
    assert "ResponseMetadata" in response, "Response should contain ResponseMetadata"

    metadata = response["ResponseMetadata"]
    assert (
        metadata["HTTPStatusCode"] == expected_status_code
    ), f"HTTP status code should be {expected_status_code}, got {metadata['HTTPStatusCode']}"

    assert "RequestId" in metadata, "ResponseMetadata should contain RequestId"
    assert metadata["RequestId"] != "", "RequestId should not be empty"

    # Verify HostId in metadata (if present)
    assert "HostId" in metadata, "ResponseMetadata should contain HostId"

    assert "HTTPHeaders" in metadata, "ResponseMetadata should contain HTTPHeaders"
    return metadata


def validate_error_response(error, expected_status_code, *args):
    """
    This function validates the HTTP status code in the error response metadata.
    Args:
        error (any): S3 Error object
        expected_status (int): HTTP status code expected in the error response
    Raises:
        AssertionError: If validation fails
    """
    print(f"Validating error response: {error}")
    assert error is not None, "Error should not be None"
    assert hasattr(error, "response"), "Error should have a response attribute"

    response = error.response
    metadata = response["ResponseMetadata"]
    assert (
        metadata["HTTPStatusCode"] == expected_status_code
    ), f"HTTP status code should be {expected_status_code}, got {metadata['HTTPStatusCode']}"

    metadata = response.get("Error", {})
    error_code = metadata.get("Code", "")
    assert error_code in args, f"Error code should be {args}, got: {error_code}"


def validate_object_lock_configuration(manager, bucket_name, expected_status):
    """
    This function validates the object lock configuration of a bucket.
    Args:
        manager (bucket.S3BucketManager): S3BucketManager instance
        bucket_name (str): Name of the bucket
        expected_status (str): Expected Object Lock status ('Enabled' or 'Disabled')
    Raises:
        AssertionError: If validation fails
    """
    response = manager.get_object_lock_configuration(bucket_name)
    status = response.get("ObjectLockConfiguration", {}).get(
        "ObjectLockEnabled", "Disabled"
    )

    assert (
        status == expected_status
    ), f"Object Lock status should be {expected_status}, got: {status}"


def validate_bucket_versioning(manager, bucket_name, expected_status):
    """
    This function validates the versioning status of a bucket.
    Args:
        manager (bucket.S3BucketManager): S3BucketManager instance
        bucket_name (str): Name of the bucket
        expected_status (str): Expected Versioning status ('Enabled', 'Suspended', or 'Disabled')
    Raises:
        AssertionError: If validation fails
    """
    response = manager.get_bucket_versioning(bucket_name)
    status = response.get("Status", "Disabled")

    assert (
        status == expected_status
    ), f"Bucket Versioning status should be {expected_status}, got: {status}"


def validate_bucket_tags(manager, bucket_name, expected_tags):
    """
    This function validates the tags of a bucket.
    Args:
        manager (bucket.S3BucketManager): S3BucketManager instance
        bucket_name (str): Name of the bucket
        expected_tags (dict): Expected tags as a dictionary
    Raises:
        AssertionError: If validation fails
    """
    tags = manager.get_bucket_tagging(bucket_name)

    assert len(tags) == len(
        expected_tags
    ), f"Number of tags should be {len(expected_tags)}, got: {len(tags)}"

    for key, value in expected_tags.items():
        assert (
            tags.get(key) == value
        ), f"Tag {key} should have value {value}, got: {tags.get(key)}"


def upload_object_versions(manager, bucket_name, object_key, count=2):
    """
    This function uploads multiple versions of an object to a versioned bucket.
    Args:
        manager (object.S3ObjectManager): S3ObjectManager instance
        bucket_name (str): Name of the bucket
        object_key (str): Key of the object
        count (int): Number of versions to upload (default is 2)
    """
    validate_bucket_versioning(manager.bucket_manager, bucket_name, "Enabled")

    for c in range(1, count + 1):
        s = f"version_{c}"
        manager.put_object(bucket_name, object_key, s.encode("utf-8"))

    versions_response = manager.list_object_versions(bucket_name)
    assert (
        len(versions_response.get("Versions", [])) == count
    ), f"Should have {count} versions"
