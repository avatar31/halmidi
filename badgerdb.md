Awesome — let’s dive into some **micro-optimizations for BadgerDB** that are especially useful under **high load**, with a focus on:

1. 🚮 **GC tuning**
2. 📦 **Batching reads/writes**
3. 🔁 **Efficient iteration**
4. 🧠 **Memory & LSM tuning**
5. 🛠️ **General performance flags**

---

## 1. 🚮 **Value Log GC Tuning**

Badger uses a **log-structured merge (LSM) tree**, and writes go to a **value log**. Over time, deleted or overwritten values accumulate as garbage.

### ✅ What you should do:

Run GC periodically in a background goroutine:

```go
func runGC(db *badger.DB) {
	for {
		err := db.RunValueLogGC(0.5) // Try to reclaim space if reclaimable space >= 50%
		if err == nil {
			continue // Keep running if GC was successful
		}
		time.Sleep(10 * time.Minute) // Sleep before next GC attempt
	}
}
```

### 🔧 Tuning Tips:

* Run GC every 10–30 minutes
* Tune the discard ratio (`0.5` means "collect if 50% or more of the log is garbage")

---

## 2. 📦 **Batching Reads/Writes**

### ✅ Why:

* Individual transactions are expensive
* Batching improves write throughput dramatically

### ✍️ Batched Write Example:

```go
func batchedWrite(db *badger.DB, kvs map[string][]byte) error {
	return db.Update(func(txn *badger.Txn) error {
		for k, v := range kvs {
			if err := txn.Set([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}
```

### 🔁 Batched Read Example:

```go
func batchedRead(db *badger.DB, keys [][]byte) (map[string][]byte, error) {
	results := make(map[string][]byte)
	err := db.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			item, err := txn.Get(key)
			if err == badger.ErrKeyNotFound {
				continue
			} else if err != nil {
				return err
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			results[string(key)] = val
		}
		return nil
	})
	return results, err
}
```

---

## 3. 🔁 **Iterator Best Practices**

Use iterators for:

* Scanning ranges
* Prefix lookups
* Streaming-like reads

### ⚡ Prefix Scan Example:

```go
func prefixScan(db *badger.DB, prefix string) error {
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			k := item.Key()
			v, _ := item.ValueCopy(nil)
			fmt.Printf("Key=%s Value=%s\n", k, v)
		}
		return nil
	})
}
```

---

## 4. 🧠 **Memory, LSM, and Table Tuning**

### Key Settings to Tweak:

```go
opts := badger.DefaultOptions(path).
    WithNumMemtables(5).                  // More memtables = better write performance
    WithNumLevelZeroTables(5).           // Avoid stalls at level 0
    WithNumLevelZeroTablesStall(10).     // Raise stall threshold
    WithMaxTableSize(64 << 20).          // 64MB per table (default is 64MB, tweak if needed)
    WithValueThreshold(128).             // Inline small values, large values go to value log
```

### Guidelines:

| Option               | Use Case                                              |
| -------------------- | ----------------------------------------------------- |
| `WithNumMemtables`   | Increase if you're write-heavy                        |
| `WithMaxTableSize`   | Bump to 128MB+ on SSDs for better compaction          |
| `WithValueThreshold` | Lower to inline small values; higher for large values |
| `WithCompression`    | Turn off compression for hot read paths               |
| `WithBlockCacheSize` | Control memory usage for block cache                  |

---

## 5. 🛠️ **General Tips**

### ✅ Parallel Reads:

Badger supports **concurrent reads and writes** out of the box — use goroutines.

### ✅ Avoid Long Transactions:

Keep each transaction short and focused. Use batching, but don’t hold TXNs too long.

### ✅ Use Read-Only DB Mode:

If you're only reading:

```go
opts := opts.WithReadOnly(true)
```

Improves performance, especially for analytics or large scans.

### ✅ Disable Sync for Write Caching (use cautiously):

```go
opts := opts.WithSyncWrites(false)
```

Gives faster writes but risks data loss on crash. Best used with:

* Replication
* External durability (like WAL elsewhere)

---

## 💡 Summary of Micro-Optimizations

| Area       | Optimization                          |
| ---------- | ------------------------------------- |
| GC         | Run `RunValueLogGC()` periodically    |
| Writes     | Use batched `txn.Set()`               |
| Reads      | Batch key lookups, use prefetch       |
| Iteration  | Use iterators + prefix efficiently    |
| LSM tuning | Adjust `NumMemtables`, `MaxTableSize` |
| Value size | Use `ValueThreshold` smartly          |
| Caching    | Only if reverse mapping or large data |

---

### 6. **Missing Metrics and Observability**
No instrumentation for:
- Transaction durations
- Oracle timestamp drift
- BulkGet performance
- GC efficiency

**Recommendation:**
```go
type DBMetrics struct {
    TxnDuration     prometheus.Histogram
    OracleTimestamp prometheus.Gauge
    BulkGetSize     prometheus.Histogram
    GCRuns          prometheus.Counter
}

func (bdb *BadgerDB) NewTransaction(ctx context.Context, readOnly bool,
    performOps func(context.Context, *badger.Txn) error) error {

    start := time.Now()
    defer func() {
        metrics.TxnDuration.Observe(time.Since(start).Seconds())
    }()

    // ... rest of implementation
}
```

Would you like a full production-ready wrapper that:

* Manages batching
* Caches keys
* Handles TTLs
* Supports versioning?

Or just help integrating these into your current system?

--------------------------------------------------------------------------------------------------------

Perfect. Here’s a **production-ready BadgerDB wrapper** in Go, covering:

✅ Deterministic UUID-based keys
✅ Batching (reads/writes)
✅ Feature rename handling via alias map
✅ Optional TTL-aware GC task
✅ Efficient prefix scanning
✅ Configurable options for performance
✅ Clear, maintainable structure

---

## 🧱 Project Layout (Single-file version for simplicity)

You can split this later into packages like `db/`, `uuid/`, `models/`, etc.

---

### 📁 `main.go` (Full Code)

```go
package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

// --------------------- UUID Helpers ---------------------

var (
	NameNamespace        = uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	FeatureNameNamespace = uuid.MustParse("abcdefab-cdef-abcd-efab-cdefabcdefab")
)

func generateUUID(namespace uuid.UUID, name string) uuid.UUID {
	return uuid.NewSHA1(namespace, []byte(name))
}

// --------------------- DB Wrapper ---------------------

type DBWrapper struct {
	db  *badger.DB
	mu  sync.RWMutex
	opt Options
}

type Options struct {
	Path           string
	EnableGC       bool
	GCInterval     time.Duration
	GCDiscardRatio float64
}

// Create new DB with tuned performance options
func NewDBWrapper(opt Options) (*DBWrapper, error) {
	opts := badger.DefaultOptions(opt.Path).
		WithLoggingLevel(badger.ERROR).
		WithNumMemtables(5).
		WithNumLevelZeroTables(5).
		WithNumLevelZeroTablesStall(10).
		WithMaxTableSize(64 << 20).
		WithValueThreshold(256).
		WithCompression(badger.Snappy)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	wrapper := &DBWrapper{db: db, opt: opt}

	if opt.EnableGC {
		go wrapper.runGC()
	}

	return wrapper, nil
}

func (d *DBWrapper) Close() {
	d.db.Close()
}

// --------------------- Core Set/Get ---------------------

func (d *DBWrapper) Set(key, val []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (d *DBWrapper) Get(key []byte) ([]byte, error) {
	var val []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	return val, err
}

// --------------------- Batched Set/Get ---------------------

func (d *DBWrapper) SetBatch(data map[string][]byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		for k, v := range data {
			if err := txn.Set([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DBWrapper) GetBatch(keys []string) (map[string][]byte, error) {
	results := make(map[string][]byte)
	err := d.db.View(func(txn *badger.Txn) error {
		for _, k := range keys {
			item, err := txn.Get([]byte(k))
			if err == badger.ErrKeyNotFound {
				continue
			} else if err != nil {
				return err
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			results[k] = val
		}
		return nil
	})
	return results, err
}

// --------------------- Prefix Scan ---------------------

func (d *DBWrapper) PrefixScan(prefix string) (map[string][]byte, error) {
	results := make(map[string][]byte)
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			k := item.Key()
			v, _ := item.ValueCopy(nil)
			results[string(k)] = v
		}
		return nil
	})
	return results, err
}

// --------------------- Feature Rename (Alias Mapping) ---------------------

func (d *DBWrapper) ResolveFeatureUUID(featureName string) uuid.UUID {
	originalUUID := generateUUID(FeatureNameNamespace, featureName)
	aliasKey := []byte("alias:feature_name:" + originalUUID.String())

	// Check for alias
	if val, err := d.Get(aliasKey); err == nil {
		if newUUID, err := uuid.Parse(string(val)); err == nil {
			return newUUID
		}
	}
	return originalUUID
}

func (d *DBWrapper) RenameFeature(oldName, newName string) error {
	oldUUID := generateUUID(FeatureNameNamespace, oldName)
	newUUID := generateUUID(FeatureNameNamespace, newName)
	aliasKey := "alias:feature_name:" + oldUUID.String()
	return d.Set([]byte(aliasKey), []byte(newUUID.String()))
}

// --------------------- GC Task ---------------------

func (d *DBWrapper) runGC() {
	ticker := time.NewTicker(d.opt.GCInterval)
	defer ticker.Stop()
	for range ticker.C {
	again:
		err := d.db.RunValueLogGC(d.opt.GCDiscardRatio)
		if err == nil {
			goto again // Keep collecting if GC successful
		}
	}
}

// --------------------- Example Usage ---------------------

func main() {
	db, err := NewDBWrapper(Options{
		Path:           "./badgerdb",
		EnableGC:       true,
		GCInterval:     10 * time.Minute,
		GCDiscardRatio: 0.5,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Example input
	userName := "some-user-id"
	feature := "analytics"
	featureName := "daily_active_users"

	userUUID := generateUUID(NameNamespace, userName)
	featureUUID := db.ResolveFeatureUUID(featureName)

	key := fmt.Sprintf("user:%s:%s:%s", userUUID, feature, featureUUID)
	value := []byte("enabled")

	if err := db.Set([]byte(key), value); err != nil {
		log.Fatal(err)
	}

	// Read back
	val, err := db.Get([]byte(key))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Value:", string(val))

	// Feature rename
	db.RenameFeature("daily_active_users", "daily_users")
	newUUID := db.ResolveFeatureUUID("daily_active_users")
	fmt.Println("Resolved UUID after rename:", newUUID)
}
```

---

## ✅ Features Included:

| Feature                | Included? | Notes                |
| ---------------------- | --------- | -------------------- |
| Deterministic UUIDs    | ✅         | UUIDv5 via SHA-1     |
| Batching               | ✅         | Reads & writes       |
| Feature rename support | ✅         | Alias mapping        |
| Prefix scans           | ✅         | For fast lookups     |
| GC background task     | ✅         | Optional via config  |
| Performance tuning     | ✅         | LSM, memtables, etc. |

---

## 🔄 Next Steps (Optional Enhancements)

* TTL support on keys
* Namespaced DB for multitenancy
* Metrics collection (via Prometheus)
* WAL backup/restore
* Compression toggle

---

Let me know if you want this split into reusable packages or integrated into a real API (e.g., gRPC or REST).


https://dbdb.io/db/badgerdb

Entrypting data

```go
package main

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
    "log"

    "github.com/dgraph-io/badger/v4"
)

// Encrypt a plaintext string with a given key using AES-GCM
func encrypt(key []byte, plaintext string) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, aesGCM.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
    return ciphertext, nil
}

// Decrypt a ciphertext blob with a given key
func decrypt(key []byte, ciphertext []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := aesGCM.NonceSize()
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}

func main() {
    // Custom per-value key (must be 16, 24, or 32 bytes)
    encryptionKey := []byte("example-32-byte-key-for-aes-gcm!!") // 32 bytes

    opts := badger.DefaultOptions("./badger-selective-encryption")
    db, err := badger.Open(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Encrypt a specific value
    plaintext := "Sensitive user data"
    encryptedValue, err := encrypt(encryptionKey, plaintext)
    if err != nil {
        log.Fatal(err)
    }

    // Store encrypted value
    err = db.Update(func(txn *badger.Txn) error {
        return txn.Set([]byte("user:123:data"), encryptedValue)
    })
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve and decrypt value
    err = db.View(func(txn *badger.Txn) error {
        item, err := txn.Get([]byte("user:123:data"))
        if err != nil {
            return err
        }

        return item.Value(func(val []byte) error {
            decrypted, err := decrypt(encryptionKey, val)
            if err != nil {
                return err
            }
            fmt.Println("Decrypted value:", decrypted)
            return nil
        })
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Enhancements

### Long Term:
- Consider implementing distributed Oracle for multi-node scenarios
- Add circuit breaker for DB operations
- Implement query result caching layer
- Add DB operation tracing with OpenTelemetry
