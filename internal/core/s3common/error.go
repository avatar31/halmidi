package s3common

import "net/http"

// https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html

type S3ErrorCode string

const (
	InternalError                S3ErrorCode = "InternalError"
	MalformedXML                 S3ErrorCode = "MalformedXML"
	MalformedPOSTRequest         S3ErrorCode = "MalformedPOSTRequest"
	InvalidArgument              S3ErrorCode = "InvalidArgument"
	InvalidKey                   S3ErrorCode = "InvalidKey"
	InvalidRequest               S3ErrorCode = "InvalidRequest"
	MalformedPolicyDocument      S3ErrorCode = "MalformedPolicyDocument"
	InsufficientCapacity         S3ErrorCode = "InsufficientCapacity"
	AccessDenied                 S3ErrorCode = "AccessDenied"
	NoSuchBucket                 S3ErrorCode = "NoSuchBucket"
	InvalidBucketName            S3ErrorCode = "InvalidBucketName"
	BucketAlreadyExists          S3ErrorCode = "BucketAlreadyExists"
	EntityAlreadyExists          S3ErrorCode = "EntityAlreadyExists"
	BucketNotEmpty               S3ErrorCode = "BucketNotEmpty"
	NoSuchKey                    S3ErrorCode = "NoSuchKey"
	NoSuchVersion                S3ErrorCode = "NoSuchVersion"
	NoSuchTagSet                 S3ErrorCode = "NoSuchTagSet"
	NoSuchUpload                 S3ErrorCode = "NoSuchUpload"
	NoSuchLifecycleConfiguration S3ErrorCode = "NoSuchLifecycleConfiguration"
	NoSuchEntity                 S3ErrorCode = "NoSuchEntity"
	InvalidPart                  S3ErrorCode = "InvalidPart"
	TooManyTags                  S3ErrorCode = "TooManyTags"
	InvalidTag                   S3ErrorCode = "InvalidTag"
	MethodNotAllowed             S3ErrorCode = "MethodNotAllowed"
	MissingRequiredParameter     S3ErrorCode = "MissingRequiredParameter"
	LimitExceeded                S3ErrorCode = "LimitExceeded"
	InvalidAccessKeyId           S3ErrorCode = "InvalidAccessKeyId"
	UnauthorizedAccess           S3ErrorCode = "UnauthorizedAccess"
	SignatureDoesNotMatch        S3ErrorCode = "SignatureDoesNotMatch"
	AuthorizationHeaderMalformed S3ErrorCode = "AuthorizationHeaderMalformed"
	RequestTimeTooSkewed         S3ErrorCode = "RequestTimeTooSkewed"
	InvalidRange                 S3ErrorCode = "InvalidRange"
	DeleteConflict               S3ErrorCode = "DeleteConflict"
)

type S3Error struct {
	Bucket         string
	Message        string
	S3ErrorCode    S3ErrorCode
	HttpStatusCode int
}

func (e S3Error) Error() string {
	return e.Message
}

func (e *S3Error) SetHttpStatusCode(code int) {
	e.HttpStatusCode = code
}

func GetNewS3Error(bucket, message string, s3ErrCode S3ErrorCode) S3Error {
	return S3Error{
		Bucket:      bucket,
		Message:     message,
		S3ErrorCode: s3ErrCode,
	}
}

func IsS3Error(err error) bool {
	_, ok := err.(S3Error)
	return ok
}

// 400 Bad Request
func GetInvalidArgumentS3Error(bucket string, message ...string) error {
	msg := getMessageString("Invalid argument", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidArgument)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetMalformedXMLS3Error(bucket string, message ...string) error {
	msg := getMessageString("The XML you provided was not well-formed", message...)
	s3err := GetNewS3Error(bucket, msg, MalformedXML)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetMalformedPOSTRequestS3Error(bucket string, message ...string) error {
	msg := getMessageString("The body of your POST request is not well-formed multipart/form-data.", message...)
	s3err := GetNewS3Error(bucket, msg, MalformedPOSTRequest)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetInvalidKeyS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified key is not valid.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidKey)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetInvalidRequestS3Error(bucket string, message ...string) error {
	msg := getMessageString("Invalid Request.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidRequest)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetMalformedPolicyDocumentS3Error(bucket string, message ...string) error {
	msg := getMessageString("The content of the form does not meet the conditions specified in the policy document.", message...)
	s3err := GetNewS3Error(bucket, msg, MalformedPolicyDocument)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetInvalidPartS3Error(bucket string, message ...string) error {
	msg := getMessageString("One or more of the specified parts could not be found.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidPart)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetTooManyTagsS3Error(bucket string, message ...string) error {
	msg := getMessageString("The number of tags exceeds the limit.", message...)
	s3err := GetNewS3Error(bucket, msg, TooManyTags)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetInvalidTagS3Error(bucket string, message ...string) error {
	msg := getMessageString("This request contains a tag key or value that isn't valid.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidTag)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetMissingRequiredParameterS3Error(bucket string, message ...string) error {
	msg := getMessageString("Missing Required Parameter.", message...)
	s3err := GetNewS3Error(bucket, msg, MissingRequiredParameter)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 400 Bad Request
func GetAuthorizationHeaderMalformedS3Error(bucket string, message ...string) error {
	msg := getMessageString("The authorization header that you provided is not valid.", message...)
	s3err := GetNewS3Error(bucket, msg, AuthorizationHeaderMalformed)
	s3err.SetHttpStatusCode(http.StatusBadRequest)
	return s3err
}

// 401 Unauthorized
func GetUnauthorizedAccessS3Error(bucket string, message ...string) error {
	msg := getMessageString("You are not authorized to perform this operation.", message...)
	s3err := GetNewS3Error(bucket, msg, UnauthorizedAccess)
	s3err.SetHttpStatusCode(http.StatusUnauthorized)
	return s3err
}

// 403 Forbidden
func GetRequestTimeTooSkewedS3Error(bucket string, message ...string) error {
	msg := getMessageString("The difference between the request time and the server's time is too large.", message...)
	s3err := GetNewS3Error(bucket, msg, RequestTimeTooSkewed)
	s3err.SetHttpStatusCode(http.StatusForbidden)
	return s3err
}

// 403 Forbidden
func GetSignatureDoesNotMatchS3Error(bucket string, message ...string) error {
	msg := getMessageString("The request signature that the server calculated does not match the signature that you provided.", message...)
	s3err := GetNewS3Error(bucket, msg, SignatureDoesNotMatch)
	s3err.SetHttpStatusCode(http.StatusForbidden)
	return s3err
}

// 403 Forbidden
func GetInvalidAccessKeyIdS3Error(bucket string, message ...string) error {
	msg := getMessageString("The access key ID that you provided does not exist in our records.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidAccessKeyId)
	s3err.SetHttpStatusCode(http.StatusForbidden)
	return s3err
}

// 403 Forbidden
func GetAccessDeniedS3Error(bucket string, message ...string) error {
	msg := getMessageString("Operation is not allowed.", message...)
	s3err := GetNewS3Error(bucket, msg, AccessDenied)
	s3err.SetHttpStatusCode(http.StatusForbidden)
	return s3err
}

// 404 Not Found
func GetNoSuchKeyS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified key does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchKey)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchVersionS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified version id does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchVersion)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchBucketS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified bucket does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchBucket)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchUploadS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified multipart upload does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchUpload)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchTagSetS3Error(bucket string, message ...string) error {
	msg := getMessageString("The TagSet does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchTagSet)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchLifecycleConfigurationS3Error(bucket string, message ...string) error {
	msg := getMessageString("The specified lifecycle configuration does not exist.", message...)
	s3err := GetNewS3Error(bucket, msg, NoSuchLifecycleConfiguration)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 404 Not Found
func GetNoSuchEntityS3Error(entity string, message ...string) error {
	msg := getMessageString("The referenced resource entity that does not exist.", message...)
	s3err := GetNewS3Error(entity, msg, NoSuchEntity)
	s3err.SetHttpStatusCode(http.StatusNotFound)
	return s3err
}

// 405 Not Found
func GetMethodNotAllowedS3Error(entity string, message ...string) error {
	msg := getMessageString("The specified method is not allowed against this resource.", message...)
	s3err := GetNewS3Error(entity, msg, MethodNotAllowed)
	s3err.SetHttpStatusCode(http.StatusMethodNotAllowed)
	return s3err
}

// 409 Conflict
func GetBucketAlreadyExistsS3Error(bucket string, message ...string) error {
	msg := getMessageString("The requested bucket name is not available.", message...)
	s3err := GetNewS3Error(bucket, msg, BucketAlreadyExists)
	s3err.SetHttpStatusCode(http.StatusConflict)
	return s3err
}

// 409 Conflict
func GetEntityAlreadyExistsS3Error(entity string, message ...string) error {
	msg := getMessageString("Entity with given name already exist.", message...)
	s3err := GetNewS3Error(entity, msg, EntityAlreadyExists)
	s3err.SetHttpStatusCode(http.StatusConflict)
	return s3err
}

// 409 Conflict
func GetBucketNotEmptyS3Error(bucket string, message ...string) error {
	msg := getMessageString("The bucket that you tried to delete is not empty.", message...)
	s3err := GetNewS3Error(bucket, msg, BucketNotEmpty)
	s3err.SetHttpStatusCode(http.StatusConflict)
	return s3err
}

// 409 Conflict
func GetLimitExceededS3Error(username string, message ...string) error {
	msg := getMessageString("Attempted to create resources beyond the current limits.", message...)
	s3err := GetNewS3Error(username, msg, LimitExceeded)
	s3err.SetHttpStatusCode(http.StatusConflict)
	return s3err
}

// 409 Conflict
func GetDeleteConflictS3Error(username string, message ...string) error {
	msg := getMessageString("The request was rejected because it attempted to delete a resource that has attached subordinate entities.", message...)
	s3err := GetNewS3Error(username, msg, DeleteConflict)
	s3err.SetHttpStatusCode(http.StatusConflict)
	return s3err
}

// 416 Invalid Range Header
func GetInvalidRangeS3Error(bucket string, message ...string) error {
	msg := getMessageString("The requested range cannot be satisfied.", message...)
	s3err := GetNewS3Error(bucket, msg, InvalidRange)
	s3err.SetHttpStatusCode(http.StatusRequestedRangeNotSatisfiable)
	return s3err
}

// 500 Internal Server Error
func GetInternalErrorS3Error(bucket string, message ...string) error {
	msg := getMessageString("We encountered an internal error. Please try again.", message...)
	s3err := GetNewS3Error(bucket, msg, InternalError)
	s3err.SetHttpStatusCode(http.StatusInternalServerError)
	return s3err
}

func getMessageString(def string, message ...string) string {
	if len(message) > 0 {
		return message[0]
	}
	return def
}
