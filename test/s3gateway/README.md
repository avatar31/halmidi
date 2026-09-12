# Halmidi s3 Gateway Test Framework

### How to run
- Create a virtual env with python3
- Install requirements
- Run Test cases using pytest command

```sh
halmidi $ cd test/s3gateway
halmidi/test/s3gateway $ python3 -m venv venv
halmidi/test/s3gateway $ source venv/bin/activate
halmidi/test/s3gateway $ pip3 install -r requirements.txt
halmidi/test/s3gateway $ pytest -m s3
```

- To Run Single Test
```sh
pytest s3native/testcases/bucket/test_bucket_create.py::TestBucketCreate::test_create_bucket
```
