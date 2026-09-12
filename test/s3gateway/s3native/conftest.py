import pytest
import boto3
from botocore.client import Config


@pytest.fixture(scope='session', autouse=True)
def s3_client():
    """Create a boto3 S3 client connected to object server."""
    return boto3.client(
        "s3",
        endpoint_url="http://localhost:9000",
        aws_access_key_id="minioadmin",
        aws_secret_access_key="minioadmin",
        config=Config(signature_version="s3v4"),
    )


@pytest.fixture(scope="class", autouse=True)
def initialize_s3_client(request, s3_client):
    """Initialize S3 client for test classes."""
    request.cls.s3_client = s3_client
