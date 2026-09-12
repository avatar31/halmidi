package iam

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

// Notes:
// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html
// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html

const (
	SigV4Algorithm     = "AWS4-HMAC-SHA256"
	iso8601BasicLayout = "20060102T150405Z"
	shortDateLayout    = "20060102"
	defaultAllowedSkew = 15 * time.Minute

	// x-amz-content-sha256 header values
	UnsignedPayloadToken                 = "UNSIGNED-PAYLOAD"
	StreamingUnsignedPayloadTrailerToken = "STREAMING-UNSIGNED-PAYLOAD-TRAILER"
	StreamingHMACPayloadToken            = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	StreamingHMACPayloadTrailerToken     = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD-TRAILER"
	StreamingECDSAPayloadToken           = "STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD"
	StreamingECDSAPayloadTrailerToken    = "STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD-TRAILER"
)

// Example Header:
//
// GET /object-store/v1 HTTP/1.1
// Host: localhost:9051
// Accept-Encoding: identity
// Authorization: AWS4-HMAC-SHA256 Credential=LZQIV7UP69UXLCC8L3TTC4BN2YO30MCP/20250912/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=e3cb9c2e8f1d55818d56221df34867df314bc2bc88451cfc58d01f136a088df4
// User-Agent: aws-cli/2.27.56 md/awscrt#0.26.1 ua/2.1 os/linux#5.15.0-151-generic md/arch#x86_64 lang/python#3.13.4 md/pyimpl#CPython m/C,E,b,N,Z cfg/retry-mode#standard md/installer#exe md/distrib#ubuntu.22 md/prompt#off md/command#s3.ls
// X-Amz-Content-Sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
// X-Amz-Date: 20250912T123423Z
func VerifyRequest(r *http.Request) (string, error) {
	var accessKey, credentialScope, signature string
	var signedHeaders []string
	var isPresign bool
	var utcTime *time.Time
	var expirySeconds int
	authHeader := r.Header.Get("Authorization")
	q := r.URL.Query()
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	if authHeader != "" { // Signed request
		var err error
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 && parts[0] != SigV4Algorithm {
			log.Errorf("Invalid sigV4 algorithm specified: %s", authHeader)
			return "", s3common.GetSignatureDoesNotMatchS3Error("", "Invalid sigV4 algorithm specified")
		}

		params := parseAuthHeaderParams(parts[1])
		credential, ok := params["Credential"]
		if !ok || credential == "" {
			return "", s3common.GetAccessDeniedS3Error("", "Credential is not specified in Authorization header")
		}

		credParts := strings.SplitN(credential, "/", 2)
		accessKey = credParts[0]
		if len(credParts) == 2 {
			credentialScope = credParts[1]
		}

		if sh, ok := params["SignedHeaders"]; ok {
			signedHeaders = strings.Split(sh, ";")
		}
		if s, ok := params["Signature"]; ok {
			signature = s
		}

		utcTime, err = parseTimeStamp(r.Header.Get("X-Amz-Date"))
		if err != nil {
			return "", err
		}
	} else if q.Get("X-Amz-Algorithm") == SigV4Algorithm { // presigned URL
		var err error
		isPresign = true

		credential := q.Get("X-Amz-Credential")
		if credential == "" {
			return "", s3common.GetAccessDeniedS3Error("", "Credential is not specified in X-Amz-Credential query param")
		}

		credParts := strings.SplitN(credential, "/", 2)
		accessKey = credParts[0]
		if len(credParts) == 2 {
			credentialScope = credParts[1]
		}

		sh := q.Get("X-Amz-SignedHeaders")
		if sh != "" {
			signedHeaders = strings.Split(sh, ";")
		}

		signature = q.Get("X-Amz-Signature")
		utcTime, err = parseTimeStamp(q.Get("X-Amz-Date"))
		if err != nil {
			return "", err
		}

		if exp := q.Get("X-Amz-Expires"); exp != "" {
			i := utils.AtoiDefault(exp, 0)
			if i == 0 {
				return "", s3common.GetInvalidRequestS3Error("", "Invalid X-Amz-Expires query param.")
			}
			expirySeconds = i
		}
	} else {
		return "", s3common.GetUnauthorizedAccessS3Error("")
	}

	// TODO: Understand this properly
	now := time.Now().UTC()
	if expirySeconds > 0 && isPresign {
		// presigned URL: timestamp + expiry must be >= now
		expiryTime := utcTime.Add(time.Duration(expirySeconds) * time.Second)
		if now.Before(utcTime.Add(-defaultAllowedSkew)) || now.After(expiryTime.Add(defaultAllowedSkew)) {
			return "", s3common.GetRequestTimeTooSkewedS3Error("")
		}
	} else {
		// normal signed request: ensure skew within allowed
		if now.Sub(*utcTime) > defaultAllowedSkew || utcTime.Sub(now) > defaultAllowedSkew {
			return "", s3common.GetRequestTimeTooSkewedS3Error("")
		}
	}

	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = hexSha256([]byte(""))
		if r.ContentLength > 0 {
			var err error
			payloadHash, err = computePayloadHash(r)
			if err != nil {
				return "", err
			}
		}
	} else {
		// TODO: How to authenticate streaming data hash
		if payloadHash != UnsignedPayloadToken &&
			payloadHash != StreamingUnsignedPayloadTrailerToken &&
			payloadHash != StreamingHMACPayloadToken &&
			payloadHash != StreamingHMACPayloadTrailerToken &&
			payloadHash != StreamingECDSAPayloadToken &&
			payloadHash != StreamingECDSAPayloadTrailerToken {
			computedPayloadHash, err := computePayloadHash(r)
			if err != nil {
				return "", err
			}

			if payloadHash != computedPayloadHash {
				log.Error("X-Amz-Content-Sha256 content is not matching with SHA256 hash of request body content")
				return "", s3common.GetInvalidRequestS3Error("", "The SHA256 content did not match what was expected.")
			}
		}

		switch payloadHash {
		case UnsignedPayloadToken, StreamingUnsignedPayloadTrailerToken, StreamingHMACPayloadTrailerToken,
			StreamingECDSAPayloadToken, StreamingECDSAPayloadTrailerToken:
			break
		case StreamingHMACPayloadToken:
			// Clean up chunk headers from the body
			cleanedBody, err := cleanupChunkHeaders(r.Body)
			if err != nil {
				log.WithError(err).Error("Failed to cleanup chunk headers")
				return "", s3common.GetInvalidRequestS3Error("", "Invalid chunk format")
			}

			r.Body = cleanedBody
			log.Debug("Streaming HMAC payload chunk headers cleaned")
		default:
			computedPayloadHash, err := computePayloadHash(r)
			if err != nil {
				return "", err
			}

			if payloadHash != computedPayloadHash {
				log.Error("X-Amz-Content-Sha256 content is not matching with SHA256 hash of request body content")
				return "", s3common.GetInvalidRequestS3Error("", "The SHA256 content did not match what was expected.")
			}
		}
	}

	entity, err := NewAccessKeyService(ctx).GetUserAccessKeyEntity(ctx, accessKey)
	if err != nil {
		return "", err
	}

	userAccessKey := entity.GetMetadata(ctx)
	if userAccessKey.Status != s3common.AccessKeyStatusActive.String() {
		return "", s3common.GetInvalidAccessKeyIdS3Error("", "The specified access key ID is not active.")
	}
	err = entity.UpdateAccessKeyLastUsed(ctx)
	if err != nil {
		log.WithError(err).Errorf("Failed to update last used time for access key: %s", accessKey)
	}

	err = verifySignature(r, *utcTime, signedHeaders, signature, payloadHash, credentialScope,
		userAccessKey.SecretAccessKey)
	if err != nil {
		return "", err
	}

	return userAccessKey.UserName, nil
}

// Credential=LZQIV7UP69UXLCC8L3TTC4BN2YO30MCP/20250912/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=e3cb9c2e8f1d55818d56221df34867df314bc2bc88451cfc58d01f136a088df4
func parseAuthHeaderParams(s string) map[string]string {
	res := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.Trim(parts[1], `" `)
		res[k] = v
	}
	return res
}

func parseTimeStamp(dateHeader string) (*time.Time, error) {
	if dateHeader != "" {
		t, err := time.Parse(iso8601BasicLayout, dateHeader)
		if err != nil {
			return nil, s3common.GetRequestTimeTooSkewedS3Error("")
		}

		utcTime := t.UTC()
		return &utcTime, nil
	}

	return nil, s3common.GetRequestTimeTooSkewedS3Error("")
}

func computePayloadHash(r *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", s3common.GetMalformedPOSTRequestS3Error("")
	}
	payloadHash := hexSha256(bodyBytes)
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return payloadHash, nil
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html#signing-request-intro
func verifySignature(r *http.Request, utcTime time.Time, signedHeaders []string, signature, payloadHash,
	credentialScope, secretAccessKey string) error {
	shortDate := utcTime.Format(shortDateLayout)
	region := s3common.PROXY_AWS_REGION
	serviceName := s3common.DEFAULT_SERVICE

	// credential scope: {date}/{region}/{service}/aws4_request
	scopeParts := strings.Split(credentialScope, "/")
	if len(scopeParts) >= 3 {
		if scopeParts[0] != "" {
			shortDate = scopeParts[0]
		}

		if scopeParts[1] != "" {
			region = scopeParts[1]
		}

		if scopeParts[2] != "" {
			serviceName = scopeParts[2]
		}
	}

	// Build canonical request
	canonicalReq := createCanonicalRequest(r, signedHeaders, payloadHash)
	stringToSign := strings.Join([]string{
		SigV4Algorithm,
		utcTime.Format(iso8601BasicLayout),
		credentialScope,
		hexSha256([]byte(canonicalReq)),
	}, "\n")

	// Derive signing key
	signKey := deriveSigningKey(secretAccessKey, shortDate, region, serviceName)

	// Compute expected signature
	computedSignature := hex.EncodeToString(hmacSHA256(signKey, []byte(stringToSign)))

	// Compare (constant time)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(computedSignature)), []byte(strings.ToLower(signature))) != 1 {
		return s3common.GetSignatureDoesNotMatchS3Error("")
	}

	return nil
}

// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_sigv-create-signed-request.html#create-canonical-request
func createCanonicalRequest(r *http.Request, signedHeaders []string, payloadHash string) string {
	canonicalURI := getCanonicalURI(r.URL.Path)
	canonicalQS := getCanonicalQueryString(r.URL)
	canonicalHeaders, signedHdrsStr := getCanonicalHeaders(r, signedHeaders)

	// Build canonical request parts
	parts := []string{
		r.Method,
		canonicalURI,
		canonicalQS,
		canonicalHeaders,
		signedHdrsStr,
		payloadHash,
	}
	return strings.Join(parts, "\n")
}

func getCanonicalURI(uri string) string {
	if uri == "" {
		return s3common.DEFAULT_IAM_RESOURCE_PATH
	}
	// Ensure leading slash
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}

	var result strings.Builder
	result.Grow(len(uri) * 2) // Pre-allocate for worst case

	segments := strings.Split(uri, "/")
	for i, seg := range segments {
		if i > 0 {
			result.WriteByte('/')
		}
		result.WriteString(url.PathEscape(seg))
	}

	return result.String()
}

func getCanonicalQueryString(u *url.URL) string {
	q := u.Query()
	if len(q) == 0 {
		return ""
	}

	keys := make([]string, 0, len(q))
	totalParams := 0

	// Collect keys and count total parameters in one pass
	for k, vals := range q {
		keys = append(keys, k)
		totalParams += len(vals)
	}

	sort.Strings(keys)

	var result strings.Builder
	// Estimate: key + value + "=" + "&" for each param
	avgParamSize := 20 // reasonable estimate for typical query params
	result.Grow(totalParams * avgParamSize)

	first := true
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)

		ek := escapeQueryComponent(k)
		for _, v := range vals {
			if !first {
				result.WriteByte('&')
			}
			first = false

			result.WriteString(ek)
			result.WriteByte('=')
			result.WriteString(escapeQueryComponent(v))
		}
	}

	return result.String()
}

func escapeQueryComponent(s string) string {
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	return escaped
}

func getCanonicalHeaders(r *http.Request, signedHeaders []string) (string, string) {
	var headersToSign []string
	if len(signedHeaders) > 0 {
		for _, h := range signedHeaders {
			headersToSign = append(headersToSign, strings.ToLower(strings.TrimSpace(h)))
		}
	} else {
		for k := range r.Header {
			headersToSign = append(headersToSign, strings.ToLower(k))
		}
		// include host
		if r.Host != "" {
			headersToSign = append(headersToSign, "host")
		}
	}

	headersToSign = dedupeAndSort(headersToSign)

	var b strings.Builder
	for _, hn := range headersToSign {
		var val string
		if hn == "host" {
			val = r.Host
		} else {
			val = strings.Join(r.Header[http.CanonicalHeaderKey(hn)], ",")
		}
		// Trim and compress spaces
		val = strings.TrimSpace(val)
		val = strings.Join(strings.Fields(val), " ")
		b.WriteString(hn)
		b.WriteString(":")
		b.WriteString(val)
		b.WriteString("\n")
	}
	return b.String(), strings.Join(headersToSign, ";")
}

func dedupeAndSort(list []string) []string {
	m := map[string]struct{}{}
	for _, v := range list {
		if v == "" {
			continue
		}
		m[v] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_sigv-create-signed-request.html#derive-signing-key
func deriveSigningKey(secret, date, region, service string) []byte {
	k := []byte("AWS4" + secret)
	kDate := hmacSHA256(k, []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key []byte, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func hexSha256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html
func cleanupChunkHeaders(body io.ReadCloser) (io.ReadCloser, error) {
	var cleanedData bytes.Buffer
	reader := bufio.NewReader(body)
	defer func() {
		_ = body.Close()
	}()

	for {
		// Read chunk size line (hex size + ";chunk-signature=..." + CRLF)
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Parse chunk size from the line (format: "hex-size;chunk-signature=sig\r\n")
		chunkSizeLine := strings.TrimSpace(string(line))
		if chunkSizeLine == "" {
			continue
		}

		// Extract hex size (before semicolon)
		sizeStr := strings.Split(chunkSizeLine, ";")[0]
		chunkSize, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			return nil, err
		}

		// If chunk size is 0, we've reached the end
		if chunkSize == 0 {
			// Read trailing headers and final CRLF
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil || len(strings.TrimSpace(string(line))) == 0 {
					break
				}
			}
			break
		}

		// Read the actual chunk data
		chunkData := make([]byte, chunkSize)
		_, err = io.ReadFull(reader, chunkData)
		if err != nil {
			return nil, err
		}

		// Write clean data (without headers)
		cleanedData.Write(chunkData)

		// Read trailing CRLF after chunk data
		_, _ = reader.ReadBytes('\n')
	}

	return io.NopCloser(&cleanedData), nil
}
