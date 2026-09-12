Yes — your observation is right: you wrote the raw framed stream to disk, so the stored file includes the chunk framing and per-chunk signatures. To store the *actual object bytes* and to *verify integrity*, you must **decode the SigV4 streaming chunk format** and verify each chunk signature.

Below I give you a complete, practical Go implementation you can drop into your PUT handler. It:

* Detects SigV4 streaming uploads (by headers like `x-amz-content-sha256: STREAMING-AWS4-HMAC-SHA256-PAYLOAD` or presence of `X-Amz-Decoded-Content-Length`).
* Parses the chunked framing.
* Verifies each chunk signature using the derived AWS SigV4 signing key and the previous signature (chained signing).
* Writes only the decoded chunk *data* to the destination file.
* Returns the final computed total written bytes and final chunk signature (useful to compute final ETag).

This implementation assumes you already:

* Resolved the user's secret key (via your `UserManager`) and derived the signing key, or at least can derive signing key from the secret.
* Extracted the initial signature (the "seed" previous signature) from the request: either from the `Authorization` header (for header-signed requests) or from `X-Amz-Signature` query param (for presigned URLs).
* Retrieved the `X-Amz-Date` and credential scope (date/region/service/aws4\_request) used to sign the request — these are needed to build the per-chunk string-to-sign.

---

### How chunk-signing works (short)

For each chunk the client sends, the chunk header contains a signature. The server must compute:

```
chunkStringToSign =
    "AWS4-HMAC-SHA256-PAYLOAD\n" +
    <amz-date> + "\n" +
    <credential-scope> + "\n" +
    <previous-signature> + "\n" +
    hex(SHA256(chunkData))
```

Then `chunkSignature = hex(hmac_sha256(signingKey, chunkStringToSign))`. Compare with provided signature. Update `previous-signature = chunkSignature` and continue.

---

### Drop-in code

Create a new file (e.g. `server/sigv4_streaming.go`) and paste:

```go
package server

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrChunkSignatureMismatch indicates signature verification failed for a chunk.
var ErrChunkSignatureMismatch = errors.New("chunk signature mismatch")

// decodeAndVerifySigV4Chunked reads the SigV4 chunked stream from r and writes only decoded chunk-data to out.
// Parameters:
// - r: raw request body
// - out: destination writer (file)
// - signingKey: derived signing key (kSigning) bytes (derived from secret/key/date/region/service)
// - prevSignature: initial previous signature (hex string from Authorization header or X-Amz-Signature for presigned URL)
// - amzDate: the X-Amz-Date value used for signing (format "YYYYMMDDThhmmssZ")
// - credScope: the credential scope part "YYYYMMDD/region/service/aws4_request"
//
// Returns: number of payload bytes written, lastChunkSignature (hex string), or error.
func decodeAndVerifySigV4Chunked(r io.Reader, out io.Writer, signingKey []byte, prevSignature, amzDate, credScope string) (int64, string, error) {
	br := bufio.NewReader(r)
	var totalWritten int64
	prevSig := prevSignature

	for {
		// Read chunk header line
		headerLine, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return totalWritten, prevSig, fmt.Errorf("unexpected EOF reading chunk header")
			}
			return totalWritten, prevSig, fmt.Errorf("reading chunk header: %w", err)
		}
		headerLine = strings.TrimRight(headerLine, "\r\n")

		// Header format examples:
		// "<chunk-size-in-hex>;chunk-signature=<hexsig>"
		// or "<chunk-size-in-hex>;chunk-signature=<hexsig>;other=..."
		// Some clients may also include other extensions.
		parts := strings.Split(headerLine, ";")
		if len(parts) < 1 {
			return totalWritten, prevSig, fmt.Errorf("malformed chunk header")
		}

		// parse chunk size (hex)
		sizeHex := strings.TrimSpace(parts[0])
		chunkSize64, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			return totalWritten, prevSig, fmt.Errorf("invalid chunk size hex %q: %w", sizeHex, err)
		}

		// Terminal chunk: size == 0
		if chunkSize64 == 0 {
			// after a zero chunk, there may be trailer headers and a final CRLF.
			// Consume any trailing CRLF (the protocol usually has a "\r\n" after the terminating "0" and then optional trailers + final CRLF).
			// Read the next line; if empty, OK. Otherwise, you may want to consume headers until an empty line.
			// We'll attempt to consume a single CRLF or a set of trailer lines until an empty line.
			for {
				line, err := br.ReadString('\n')
				if err != nil && err != io.EOF {
					return totalWritten, prevSig, fmt.Errorf("reading post-terminator: %w", err)
				}
				lineTrim := strings.TrimRight(line, "\r\n")
				if lineTrim == "" || err == io.EOF {
					break
				}
				// else treat as a trailer header and continue (ignore for now)
			}
			return totalWritten, prevSig, nil
		}

		// find chunk-signature among extensions
		var providedChunkSig string
		for _, ext := range parts[1:] {
			ext = strings.TrimSpace(ext)
			// extensions can be key=value pairs
			if strings.HasPrefix(ext, "chunk-signature=") {
				providedChunkSig = strings.TrimPrefix(ext, "chunk-signature=")
				providedChunkSig = strings.Trim(providedChunkSig, `"`) // remove quotes if present
				break
			}
			// also accept "chunk-signature" with = after spaces
			if strings.Contains(ext, "chunk-signature=") {
				p := strings.SplitN(ext, "chunk-signature=", 2)
				providedChunkSig = strings.Trim(p[1], `"`)
				break
			}
		}

		if providedChunkSig == "" {
			return totalWritten, prevSig, fmt.Errorf("missing chunk-signature in chunk header")
		}

		// Read chunk-data exactly chunkSize bytes
		remaining := chunkSize64
		h := sha256.New()

		// stream from br to out while hashing
		buf := make([]byte, 64*1024)
		for remaining > 0 {
			toRead := int64(len(buf))
			if remaining < toRead {
				toRead = remaining
			}
			n, err := io.ReadFull(br, buf[:toRead])
			if err != nil {
				return totalWritten, prevSig, fmt.Errorf("reading chunk-data: %w", err)
			}
			// write to file/out
			if _, err := out.Write(buf[:n]); err != nil {
				return totalWritten, prevSig, fmt.Errorf("writing decoded chunk: %w", err)
			}
			// hash for chunk
			h.Write(buf[:n])
			remaining -= int64(n)
			totalWritten += int64(n)
		}

		// after chunk-data there's a CRLF to consume
		crlf := make([]byte, 2)
		if _, err := io.ReadFull(br, crlf); err != nil {
			return totalWritten, prevSig, fmt.Errorf("reading chunk CRLF: %w", err)
		}
		if !(crlf[0] == '\r' && crlf[1] == '\n') {
			return totalWritten, prevSig, fmt.Errorf("expected CRLF after chunk-data")
		}

		// Compute chunk hash hex
		chunkHashHex := hex.EncodeToString(h.Sum(nil))

		// Build chunk string to sign
		// See AWS docs: "AWS4-HMAC-SHA256-PAYLOAD\n<time>\n<scope>\n<prevSig>\n<hex(SHA256(chunk-data))>"
		stringToSign := strings.Join([]string{
			"AWS4-HMAC-SHA256-PAYLOAD",
			amzDate,
			credScope,
			prevSig,
			chunkHashHex,
		}, "\n")

		// Compute expected chunk signature using signingKey
		expectedChunkSig := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

		// Compare signatures (case-insensitive)
		if !strings.EqualFold(expectedChunkSig, providedChunkSig) {
			return totalWritten, prevSig, ErrChunkSignatureMismatch
		}

		// advance prevSig for next chunk
		prevSig = expectedChunkSig
	}
}

// hmacSHA256 helper
func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}
```

---

### How to call this in your PUT handler

1. From your SigV4 verifier step you should already have:

   * `creds.SecretKey` for the AccessKey used.
   * `amzDate` (string) and `credScope` (string) used in initial request signing.
   * `initialSignature` (the signature in Authorization header or `X-Amz-Signature`) — this is the `prevSignature` seed.

2. Derive the signing key once (same as other SigV4 steps):

```go
shortDate := amzDate[:8] // YYYYMMDD
signingKey := deriveSigningKey(creds.SecretKey, shortDate, region, "s3")
// deriveSigningKey same as in your sigv4 package:
// kDate := HMAC("AWS4"+secret, shortDate) ...
```

3. Detect streaming upload:

```go
if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-AWS4-HMAC-SHA256") ||
   r.Header.Get("X-Amz-Decoded-Content-Length") != "" {
    // streaming
    written, lastSig, err := decodeAndVerifySigV4Chunked(r.Body, outFile, signingKey, initialSignature, amzDate, credScope)
    if err != nil {
        // return SignatureDoesNotMatch or InvalidRequest style S3 XML error
    }
    // use `written` as object size; lastSig useful if you want to store
}
```

4. If not streaming, fallback to `io.Copy(outFile, r.Body)` as before.

---

### Notes, gotchas & testing

* **Initial signature (prevSignature)**:

  * For header-signed requests, it is the `Signature` value from the `Authorization` header.
  * For presigned URLs, it is the `X-Amz-Signature` query param.
  * Your VerifyRequest earlier will have validated the initial signature — you can reuse its computed `expected` signature and use that as `prevSignature` to continue the chain. That is the safest option.

* **amzDate and credScope** must be exactly the same values used for the initial request signing. `amzDate` is the same `X-Amz-Date` header (or query param). `credScope` is the part after the `/` in the credential (e.g., `20250912/us-east-1/s3/aws4_request`) — pass it in exactly.

* **Performance**:

  * The decoder reads/writes in chunks (64KB buffer). Hashing and HMAC are fast.
  * You verify per chunk — this is required for security (detect tampering early).

* **ETag**:

  * For normal uploads ETag = MD5(file). For multipart uploads and streaming uploads ETag semantics differ. If you want to produce an MD5-like ETag for streamed uploads, compute MD5 of concatenated payload bytes (you can maintain a separate md5 hash while writing). But note: AWS S3 multipart ETags are special; match them only if you need exact behavior.

* **Error handling**:

  * If signature mismatch, return S3-style `SignatureDoesNotMatch` (403) response.
  * If chunk framing malformed, return `BadRequest`.

* **Testing**:

  * Use aws-cli to perform a large `aws s3 cp bigfile s3://bucket/` (CLI will choose streaming chunked upload for large files).
  * Compare decoded size to `X-Amz-Decoded-Content-Length`.
  * Verify CLI reports success; verify downloaded object equals original.

---

If you want, I can:

* Hook this into your existing `VerifyRequest` flow (so VerifyRequest returns `signingKey` and `initialSignature` ready for streaming decode).
* Provide a concrete `PUT` handler file showing the full flow `VerifyRequest -> create file -> decodeAndVerifySigV4Chunked -> finish and respond`.
* Add unit tests that reproduce the exact chunk frames from AWS `aws-cli` and validate decoding & signature verification.

Which would you like next?
