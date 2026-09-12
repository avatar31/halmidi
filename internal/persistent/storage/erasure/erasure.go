package erasure

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/reedsolomon"
	"github.com/avatar31/halmidi/internal/fileutils"
	osfileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/storage/disk"
)

const (
	ShardTypeData   = "data"
	ShardTypeParity = "parity"
)

type Shard struct {
	Type     string `json:"type"`
	Checksum string `json:"checksum"` // TODO: Do we need checksum?
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

type Encoder struct {
	strategy Strategy
	encoders []reedsolomon.StreamEncoder
	mu       sync.RWMutex
}

type FileInfo struct {
	DataShards   int32
	ParityShards int32
	Shards       []Shard
	CreatedAt    time.Time
	TotalSize    int64
	Checksum     string
}

const (
	DefaultBlockSize = 64 * 1024 // 64 KB
)

var (
	once    sync.Once
	encoder *Encoder
)

func Init(ctx context.Context, disks []disk.Disk) error {
	var err error
	once.Do(func() {
		diskMap := make(map[int]disk.Disk)
		for i := range disks {
			diskMap[disks[i].ID] = disks[i]
		}

		encoder = &Encoder{
			strategy: CalculateOptimalStrategy(diskMap),
		}

		streamEncoders := make([]reedsolomon.StreamEncoder, len(encoder.strategy.Sets))
		for i, set := range encoder.strategy.Sets {
			var enc reedsolomon.StreamEncoder

			dataShards := int(set.DataShards)
			parityShards := int(set.ParityShards)
			// TODO: Revisit encoder options for performance tuning
			enc, err = reedsolomon.NewStream(
				dataShards,
				parityShards,
				reedsolomon.WithConcurrentStreams(true),
				reedsolomon.WithStreamBlockSize(DefaultBlockSize),
				reedsolomon.WithAutoGoroutines(dataShards+parityShards),
			)
			if err != nil {
				err = fmt.Errorf("failed to create encoder for set %d: %w", i, err)
				return
			}
			streamEncoders[i] = enc
		}

		encoder.encoders = streamEncoders
		logger.GetLogger(ctx).Infof("Erasure initialized with configuration: %+v", encoder)
	})

	return err
}

func GetEncoder() *Encoder {
	return encoder
}

func (e *Encoder) Encode(ctx context.Context, outfileRelativePath string, contentLen int64,
	data io.Reader) (*FileInfo, error) {
	start := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before encoding: %w", err)
	}

	if len(e.strategy.Sets) == 0 {
		return nil, fmt.Errorf("no storage strategy configured")
	}

	// Since we are restricting disks count to 6-16, we will have only one set
	// TODO: In production, you might want to stripe data across multiple sets
	set := e.strategy.Sets[0]
	encoder := e.encoders[0]

	outFiles := make([]*os.File, set.TotalShards)
	// writers := make([]*bufio.Writer, set.TotalShards)
	hashers := make([]hash.Hash, set.TotalShards)
	shards := make([]Shard, set.TotalShards)

	log := logger.GetLogger(ctx).WithField("file", outfileRelativePath)
	log.Infof("Writing data in erasure set: %+v", set)

	// Cleanup function for error cases
	cleanup := func() {
		for j := range outFiles {
			if outFiles[j] != nil {
				_ = outFiles[j].Close()
				_ = os.RemoveAll(filepath.Dir(outFiles[j].Name()))
				outFiles[j] = nil // Mark as closed (prevents double-close)
			}
		}
	}

	for i := range set.Disks {
		shardPath := filepath.Join(set.Disks[i].Path, outfileRelativePath)
		if err := osfileutils.CreateDir(filepath.Dir(shardPath)); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to create directory for shard %d: %w", i, err)
		}

		f, err := os.Create(shardPath)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to create shard %d: %w", i, err)
		}

		outFiles[i] = f
		// writers[i] = bufio.NewWriterSize(f, DefaultBlockSize)
		hashers[i] = md5.New()
	}

	var totalSize atomic.Int64
	var overallHashMu sync.Mutex
	overallHash := md5.New()

	// Counting reader to track bytes
	countingReader := &fileHashReader{
		ctx:    ctx,
		reader: data,
		hash:   overallHash,
		hashMu: &overallHashMu,
		size:   &totalSize,
	}

	// Prepare data shard writers (with hash)
	dataWriters := make([]io.Writer, set.DataShards)
	for i := range set.DataShards {
		dataWriters[i] = io.MultiWriter(outFiles[i], hashers[i])
	}

	// Split input into data shards
	if err := encoder.Split(countingReader, dataWriters, contentLen); err != nil {
		log.WithError(err).Errorf("Error while splitting file into data shards")
		cleanup()
		return nil, fmt.Errorf("failed to split file into data shards: %w", err)
	}

	if err := ctx.Err(); err != nil {
		cleanup()
		return nil, fmt.Errorf("context cancelled after splitting: %w", err)
	}

	dataReaders := make([]io.Reader, set.DataShards)
	for i := range set.DataShards {
		// Seek to beginning
		if _, err := outFiles[i].Seek(0, io.SeekStart); err != nil {
			cleanup()
			return nil, fmt.Errorf("seek shard %d: %w", i, err)
		}
		dataReaders[i] = outFiles[i]
	}

	// Prepare parity shard writers (with hash)
	parityWriters := make([]io.Writer, set.ParityShards)
	for i := range set.ParityShards {
		idx := set.DataShards + i
		parityWriters[i] = io.MultiWriter(outFiles[idx], hashers[idx])
	}

	// Encode parity shards
	if err := encoder.Encode(dataReaders, parityWriters); err != nil {
		log.WithError(err).Errorf("Error while encoding parity shards")
		cleanup()
		return nil, fmt.Errorf("failed to encode parity shards: %w", err)
	}

	if err := ctx.Err(); err != nil {
		cleanup()
		return nil, fmt.Errorf("context cancelled after encoding: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, set.TotalShards)

	// Flush and sync shards in parallel
	for i := range set.TotalShards {
		wg.Add(1)
		go func(idx int32) {
			defer wg.Done()
			if ctx.Err() != nil {
				errChan <- fmt.Errorf("context cancelled during sync: %w", ctx.Err())
				return
			}

			// Sync to disk (this is the slow operation that benefits from parallelization)
			if err := outFiles[idx].Sync(); err != nil {
				errChan <- fmt.Errorf("sync shard %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors during flush/sync
	for err := range errChan {
		if err != nil {
			cleanup()
			return nil, err
		}
	}

	// Close shard files
	for i := range set.TotalShards {
		stat, err := outFiles[i].Stat()
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("stat failed: %w", err)
		}
		_ = outFiles[i].Close()

		shardType := ShardTypeData
		if i >= set.DataShards {
			shardType = ShardTypeParity
		}
		shards[i] = Shard{
			Path:     filepath.Join(set.Disks[i].Path, outfileRelativePath),
			Type:     shardType,
			Checksum: hex.EncodeToString(hashers[i].Sum(nil)),
			Size:     stat.Size(),
		}
	}

	log.Infof("Saved shard files successfully")

	info := &FileInfo{
		DataShards:   set.DataShards,
		ParityShards: set.ParityShards,
		Shards:       shards,
		CreatedAt:    time.Now().UTC(),
		TotalSize:    totalSize.Load(),
		Checksum:     hex.EncodeToString(overallHash.Sum(nil)),
	}

	elapsed := time.Since(start)
	log.Infof("Completed writing file %s. Time taken %v", outfileRelativePath, elapsed)
	return info, nil
}

func (e *Encoder) reconstruct(ctx context.Context, fileRelativePath string, fileInfo *FileInfo,
	enc reedsolomon.StreamEncoder) error {
	log := logger.GetLogger(ctx).WithField("file", fileRelativePath)
	shardReaders := e.openShards(ctx, fileInfo.Shards)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before reconstruction: %w", err)
	}

	out := make([]io.Writer, len(shardReaders))
	for i := range out {
		if shardReaders[i] == nil {
			var err error
			out[i], err = os.Create(fileInfo.Shards[i].Path)
			if err != nil {
				log.WithError(err).Errorf("Failed to create missing shard file %s", fileInfo.Shards[i].Path)
				e.closeShardReaders(shardReaders)
				return fmt.Errorf("failed to create missing shard file: %w", err)
			}
		}
	}

	shardReaders = e.openShards(ctx, fileInfo.Shards)
	err := enc.Reconstruct(shardReaders, out)
	e.closeShardReaders(shardReaders)
	if err != nil {
		log.WithError(err).Errorf("Failed to reconstruct missing/corrupted shards")
		e.closeShardReaders(shardReaders)
		return fmt.Errorf("failed to reconstruct missing/corrupted shards: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled after reconstruction: %w", err)
	}

	// Close output.
	for i := range out {
		if out[i] != nil {
			err := out[i].(*os.File).Close()
			if err != nil {
				// TODO: Is something missing here?
				log.WithError(err).Errorf("Failed to close reconstructed shard file %s", fileInfo.Shards[i].Path)
				e.closeShardReaders(shardReaders)
				return fmt.Errorf("failed to close reconstructed shard file: %w", err)
			}
		}
	}

	shardReaders = e.openShards(ctx, fileInfo.Shards)
	ok, err := enc.Verify(shardReaders)
	e.closeShardReaders(shardReaders)
	if !ok {
		log.WithError(err).Errorf("Reconstructed shards verification failed")
		return fmt.Errorf("reconstructed shards verification failed: %w", err)
	}

	return nil
}

// Decode reconstructs the original file from erasure-coded shards.
// Returns a temporary file that will be automatically deleted when closed.
// Caller MUST close the returned file to trigger cleanup.
// TODO: Test Reconstruction
func (e *Encoder) Decode(ctx context.Context, fileRelativePath string, fileInfo *FileInfo) (*os.File, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	log := logger.GetLogger(ctx).WithField("file", fileRelativePath)

	// Since we are restricting disks count to 6-16, we will have only one set
	encoder := e.encoders[0]

	shardReaders := e.openShards(ctx, fileInfo.Shards)
	ok, _ := encoder.Verify(shardReaders)
	e.closeShardReaders(shardReaders)
	if ok {
		log.Infof("All shards are valid, proceeding with reconstruction")
	} else {
		log.Infof("Some shards are missing or corrupted, attempting reconstruction")
		err := e.reconstruct(ctx, fileRelativePath, fileInfo, encoder)
		if err != nil {
			return nil, err
		}
	}

	shardReaders = e.openShards(ctx, fileInfo.Shards)
	f, err := e.joinShards(ctx, fileRelativePath, fileInfo.TotalSize, shardReaders, encoder)
	e.closeShardReaders(shardReaders)
	return f, err
}

func (e *Encoder) joinShards(ctx context.Context, fileRelativePath string, size int64, r []io.Reader,
	enc reedsolomon.StreamEncoder) (*os.File, error) {
	log := logger.GetLogger(ctx).WithField("file", fileRelativePath)

	// Create temporary output file
	outputFile, err := fileutils.CreateTempFile(ctx, fmt.Sprintf("reconstructed_%s", filepath.Base(fileRelativePath)))
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	err = enc.Join(outputFile, r, size)
	if err != nil {
		log.WithError(err).Errorf("Failed to reconstruct file from shards")
		_ = outputFile.Close()
		_ = os.Remove(outputFile.Name())
		return nil, fmt.Errorf("failed to reconstruct file from shards: %w", err)
	}

	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		_ = outputFile.Close()
		return nil, fmt.Errorf("seek file: %w", err)
	}

	log.Infof("File reconstructed successfully")
	return outputFile, nil
}

func (e *Encoder) closeShardReaders(shards []io.Reader) {
	for i := range shards {
		if shards[i] != nil {
			if closer, ok := shards[i].(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}
}

// TODO: Test Reconstruction
// TODO: Test Storage Footprint of temp files
func (e *Encoder) DecodeBytesRange(ctx context.Context, fileRelativePath string, fileInfo *FileInfo, start,
	end int64) (*os.File, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	log := logger.GetLogger(ctx).WithField("file", fileRelativePath)

	// Since we are restricting disks count to 6-16, we will have only one set
	encoder := e.encoders[0]

	// Verify & reconstruct if needed
	shardReaders := e.openShards(ctx, fileInfo.Shards)
	ok, _ := encoder.Verify(shardReaders)
	e.closeShardReaders(shardReaders)
	if !ok {
		log.Infof("Some shards are missing or corrupted, attempting reconstruction")
		if err := e.reconstruct(ctx, fileRelativePath, fileInfo, encoder); err != nil {
			return nil, err
		}
	}

	// Calculate which shards contain the requested range
	relevantShards, startOffset, endOffset := e.calculateRange(fileInfo.Shards[:fileInfo.DataShards], start, end)

	// Create output file
	outputFile, err := fileutils.CreateTempFile(ctx, fmt.Sprintf("range_%s", filepath.Base(fileRelativePath)))
	if err != nil {
		return nil, fmt.Errorf("failed to create range output file: %w", err)
	}

	// Read directly from relevant shards without full reconstruction
	remaining := endOffset - startOffset + 1
	for i, shard := range relevantShards {
		f, err := os.Open(shard.Path)
		if err != nil {
			_ = outputFile.Close()
			_ = os.Remove(outputFile.Name())
			return nil, fmt.Errorf("open shard %s: %w", shard.Path, err)
		}

		copyStart := int64(0)
		if i == 0 {
			// First shard: skip to startOffset
			copyStart = startOffset
		}

		if _, err := f.Seek(copyStart, io.SeekStart); err != nil {
			_ = f.Close()
			_ = outputFile.Close()
			_ = os.Remove(outputFile.Name())
			return nil, fmt.Errorf("seek shard %d: %w", i, err)
		}

		available := shard.Size - copyStart
		toCopy := available
		if toCopy > remaining {
			toCopy = remaining
		}

		_, err = io.CopyN(outputFile, f, toCopy)
		_ = f.Close()
		if err != nil && err != io.EOF {
			_ = outputFile.Close()
			_ = os.Remove(outputFile.Name())
			return nil, fmt.Errorf("copy shard %d: %w", i, err)
		}

		remaining -= toCopy
		if remaining <= 0 {
			break
		}
	}

	if remaining > 0 {
		_ = outputFile.Close()
		_ = os.Remove(outputFile.Name())
		return nil, fmt.Errorf("range read incomplete, %d bytes remaining", remaining)
	}

	// Seek to beginning for reading
	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		_ = outputFile.Close()
		_ = os.Remove(outputFile.Name())
		return nil, fmt.Errorf("seek output: %w", err)
	}

	return outputFile, nil
}

// TC1:
// FileSize: 100
// shards: 4
// shard1: 25
// shard2: 25
// shard3: 25
// shard4: 25
//
// request: start=20, end=65
// result: [shard1, shard2, shard3], startOffset=20, endOffset=65
//
// request: start=35, end=65
// result: [shard2, shard3], startOffset=10, endOffset=30
//
// request: start=0, end=10
// result: [shard1], startOffset=0, endOffset=10
//
// request: start=90, end=100
// result: [shard4], startOffset=15, endOffset=25
//
// TC2:
// FileSize: 32159518
// shards: 4
// shard1: 8039880
// shard2: 8039880
// shard3: 8039880
// shard4: 8039880
//
// request: start=0, end=8388607
// result: [shard1, shard2], startOffset=0, endOffset=8388607
//
// reqeust: start=8388608, end=16777215
// result: [shard2, shard3], startOffset=0, endOffset=8388607
//
// request: start=16777216, end=25165823
// result: [shard3, shard4], startOffset=0, endOffset=8388607
//
// request: start=25165824, end=32159517
// result: [shard4], startOffset=0, endOffset=6983709
func (e *Encoder) calculateRange(allShards []Shard, start, end int64) ([]Shard, int64, int64) {
	// Find which shard contains the start byte
	var startShardIdx int
	var startOffset int64
	var currentOffset int64
	for i, shard := range allShards {
		if currentOffset+shard.Size > start {
			startShardIdx = i
			startOffset = start - currentOffset
			break
		}
		currentOffset += shard.Size
	}

	// Find which shard contains the end byte
	var endShardIdx int
	currentOffset = 0
	for i, shard := range allShards {
		if currentOffset+shard.Size > end {
			endShardIdx = i
			break
		}
		currentOffset += shard.Size
	}

	// Calculate total size of shards before startShardIdx
	var offsetBeforeStart int64
	for i := 0; i < startShardIdx; i++ {
		offsetBeforeStart += allShards[i].Size
	}

	// Adjust endOffset to be relative to the relevant shards slice
	endOffset := end - offsetBeforeStart

	// Extract the relevant shards
	return allShards[startShardIdx : endShardIdx+1], startOffset, endOffset
}

func (e *Encoder) openShards(ctx context.Context, shards []Shard) []io.Reader {
	log := logger.GetLogger(ctx)
	shardReaders := make([]io.Reader, len(shards))
	for i, shard := range shards {
		f, err := os.Open(shard.Path)
		if err != nil {
			log.Warningf("Shard %s not found, will attempt reconstruction", shard.Path)
			shardReaders[i] = nil
			continue
		}
		stat, err := f.Stat()
		if err != nil {
			log.Warningf("Failed to stat shard %s, will attempt reconstruction", shard.Path)
			shardReaders[i] = nil
			continue
		}
		if stat.Size() == 0 {
			log.Warningf("Shard %s is empty, will attempt reconstruction", shard.Path)
			shardReaders[i] = nil
			continue
		}

		shardReaders[i] = f
	}

	return shardReaders
}

// fileHashReader wraps an io.Reader to count bytes and compute MD5 hash
type fileHashReader struct {
	ctx    context.Context
	reader io.Reader
	hash   hash.Hash
	hashMu *sync.Mutex
	size   *atomic.Int64
}

func (c *fileHashReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled
	if err := c.ctx.Err(); err != nil {
		return 0, fmt.Errorf("read cancelled: %w", err)
	}

	n, err = c.reader.Read(p)
	if n > 0 {
		c.size.Add(int64(n))
		// Protect hash writes from concurrent access
		if c.hashMu != nil {
			c.hashMu.Lock()
			c.hash.Write(p[:n])
			c.hashMu.Unlock()
		} else {
			c.hash.Write(p[:n])
		}
	}
	return n, err
}
