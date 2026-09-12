package iam

// // Basic Cases
// // Empty string
// getCanonicalURI("") 
// // Output: "/"

// // Root path
// getCanonicalURI("/") 
// // Output: "/"

// // Simple path
// getCanonicalURI("/bucket") 
// // Output: "/bucket"

// // Path without leading slash
// getCanonicalURI("bucket") 
// // Output: "/bucket"


// // Paths with Special Characters (Requiring Escaping)
// // Space in path
// getCanonicalURI("/my bucket/file.txt") 
// // Output: "/my%20bucket/file.txt"

// // Multiple special characters
// getCanonicalURI("/bucket/file name with spaces.txt") 
// // Output: "/bucket/file%20name%20with%20spaces.txt"

// // Plus sign (gets escaped)
// getCanonicalURI("/bucket/query+string") 
// // Output: "/bucket/query%2Bstring"

// // Percent sign (gets escaped)
// getCanonicalURI("/bucket/100%complete") 
// // Output: "/bucket/100%25complete"

// // Unicode characters
// getCanonicalURI("/bucket/файл.txt") 
// // Output: "/bucket/%D1%84%D0%B0%D0%B9%D0%BB.txt"


// // Complex S3-like Paths
// // Object key with folder structure
// getCanonicalURI("/my-bucket/folder/subfolder/file.txt") 
// // Output: "/my-bucket/folder/subfolder/file.txt"

// // Object key with special characters
// getCanonicalURI("/bucket/path/to/my file (copy).txt") 
// // Output: "/bucket/path/to/my%20file%20%28copy%29.txt"

// // Already encoded path (gets double-encoded if needed)
// getCanonicalURI("/bucket/already%20encoded") 
// // Output: "/bucket/already%2520encoded"

// // Path with query-like characters
// getCanonicalURI("/bucket/file?version=1&type=json") 
// // Output: "/bucket/file%3Fversion%3D1%26type%3Djson"


// // Edge Cases
// // Multiple slashes (preserved in segments)
// getCanonicalURI("/bucket//double//slash") 
// // Output: "/bucket//double//slash"

// // Path with only special characters
// getCanonicalURI("/100% complete!") 
// // Output: "/100%25%20complete%21"

// // Empty segments
// getCanonicalURI("/bucket//file") 
// // Output: "/bucket//file"

// // Trailing slash
// getCanonicalURI("/bucket/folder/") 
// // Output: "/bucket/folder/"


// // AWS S3 Real-world Examples
// // Typical S3 object key
// getCanonicalURI("/my-s3-bucket/logs/2023/12/01/app.log") 
// // Output: "/my-s3-bucket/logs/2023/12/01/app.log"

// // S3 object with spaces (common issue)
// getCanonicalURI("/bucket/user uploads/profile pic.jpg") 
// // Output: "/bucket/user%20uploads/profile%20pic.jpg"

// // S3 multipart upload key
// getCanonicalURI("/bucket/large-file.zip") 
// // Output: "/bucket/large-file.zip"

// // S3 object with metadata-like naming
// getCanonicalURI("/bucket/data_2023-12-01_v1.2.json") 
// // Output: "/bucket/data_2023-12-01_v1.2.json"


// // Performance Examples (Fast Path)
// // These examples show cases where the optimized function takes the fast path (no escaping needed):
// // Simple alphanumeric paths (fast path)
// getCanonicalURI("/bucket123/file456.txt") 
// // Output: "/bucket123/file456.txt" (no processing needed)

// // Paths with allowed characters (fast path)
// getCanonicalURI("/bucket/path-with_underscores.txt") 
// // Output: "/bucket/path-with_underscores.txt" (no processing needed)

// // Already canonical paths (fast path)
// getCanonicalURI("/simple/path/file.json") 
// // Output: "/simple/path/file.json" (no processing needed)

