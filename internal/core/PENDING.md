### Pending Tasks
Following tasks are need to be verified with both VAST and Minio and then implement

#### Overall
- Distributed support for Quota
- Use proto instead json to store data Badger db
- Use uuid in object keys
- Rename, Move, Copy
- Can we enable object locking after creating bucket
- The bucket is created with Object Lock ENABLED at the bucket level. No retention mode (like GOVERNANCE or COMPLIANCE) is applied yet. No retention period is applied either. No default rules are in place for new objects. So: This command only enables the capability to use Object Lock — it doesn't enforce anything yet.
- Object locking for multipart upload
- Integrate Object locking and versioning
- Add lifecycle rule simulation: auto-delete older versions
- Properly handle concurrent PUTs on same key.
- Store versions so they’re easy to find and don’t risk data corruption or race conditions.
- Ensure data and metadata are safely flushed to disk (e.g., sync writes).
- Metrics like object count, space usage
- https://opentelemetry.io/docs/languages/go/
- 2025/09/20 09:55:10 http: superfluous response.WriteHeader call from github.com/avatar31/halmidi/cmd/rest/s3_native_api_handlers.sendErrorResp (utils.go:65)
- Erasure coding and rebalancing
- https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-friendly-names
- Support `--query` in list queries



#### Hadnlers
```log
{"level":"debug","traceId":"6805ff1b-a141-424d-aa22-38e04dcf007a","method":"GET","url":"/warp-benchmark-bucket/?delimiter=&encoding-type=url&fetch-owner=true&list-type=2&prefix=","caller":"rest/middleware.go:42","time":"2025-09-19T09:56:46+05:30","message":"Request received: GET /warp-benchmark-bucket/?delimiter=&encoding-type=url&fetch-owner=true&list-type=2&prefix= HTTP/1.1\r\nHost: localhost:9051\r\nAuthorization: AWS4-HMAC-SHA256 Credential=AFARKKDMQ7JEPECMV3B/20250919/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=a8e552887b142c35ded8a09c6b185a68d4268b86cec1e7077ec8cacda5c77c42\r\nUser-Agent: MinIO (linux; amd64) minio-go/v7.0.95 warp/1.3.0\r\nX-Amz-Content-Sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\r\nX-Amz-Date: 20250919T042646Z\r\n\r\n"}
{"level":"error","traceId":"6805ff1b-a141-424d-aa22-38e04dcf007a","key":"namespace:default:bucket:warp-benchmark-bucket:objects:78fd5339-5722-5938-8314-003fcfc44979","module":"badger-db","error":"Key not found","caller":"db/badger.go:95","time":"2025-09-19T09:56:46+05:30","message":"Error while checking key exist or not"}
{"level":"info","traceId":"6805ff1b-a141-424d-aa22-38e04dcf007a","statusCode":200,"caller":"s3_native_api_handlers/utils.go:77","time":"2025-09-19T09:56:46+05:30","message":"Successfully handled api request"}
{"level":"debug","traceId":"6805ff1b-a141-424d-aa22-38e04dcf007a","caller":"s3_native_api_handlers/utils.go:82","time":"2025-09-19T09:56:46+05:30","message":"Response Headers: map[Content-Type:[application/xml] X-Trace-Id:[6805ff1b-a141-424d-aa22-38e04dcf007a]]"}
{"level":"info","traceId":"6805ff1b-a141-424d-aa22-38e04dcf007a","method":"GET","url":"/warp-benchmark-bucket/?delimiter=&encoding-type=url&fetch-owner=true&list-type=2&prefix=","elapsedTime":"0ms","caller":"rest/middleware.go:48","time":"2025-09-19T09:56:46+05:30","message":"Request completed"}
```
