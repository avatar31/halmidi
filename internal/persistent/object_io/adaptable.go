package objectio

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/avatar31/halmidi/internal/core/namespace"
	"github.com/avatar31/halmidi/internal/fileutils"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/storage/erasure"
)

// Storage Statergy:
//
//	For non-versioning objects:
//		.../{bucket}/{objectId}/{filename}
//	For versioning objects:
//		.../{bucket}/{objectId}/{version}/{filename}
type adaptableIO struct {
	mu sync.Mutex
}

func (aio *adaptableIO) Write(ctx context.Context, input ObjectInput) (*erasure.FileInfo, error) {
	aio.mu.Lock()
	defer aio.mu.Unlock()

	// case 1: Versioning is never enabled or versioning is disabled
	// 		Add file in .../{bucket}/{objectId}/{fileName}
	// case 2: Versioning is enabled
	// 		Add file in .../{bucket}/{objectId}/{version}/{fileName}
	fileName, path := getFileNameAndDir(input)
	fileinfo, err := erasure.GetEncoder().Encode(ctx, filepath.Join(path, fileName), input.ContentLen, input.Data)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Errorf("Error writing file %s/%s", path, fileName)
		return nil, err
	}

	return fileinfo, nil
}

func (aio *adaptableIO) Copy(ctx context.Context, src, dest ObjectInput) (*erasure.FileInfo, error) {
	aio.mu.Lock()
	defer aio.mu.Unlock()

	srcFile, err := aio.read(ctx, src)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = srcFile.Close()
		_ = os.Remove(srcFile.Name())
	}()

	dest.ContentLen = src.TotalSize
	fileName, path := getFileNameAndDir(dest)
	fileinfo, err := erasure.GetEncoder().Encode(ctx, filepath.Join(path, fileName), dest.ContentLen, srcFile)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Errorf("Error writing file %s/%s", path, fileName)
		return nil, err
	}

	return fileinfo, nil
}

func (aio *adaptableIO) CreateStreamForWrite(ctx context.Context, input ObjectInput) (writeStream, saveStream, error) {
	aio.mu.Lock()
	filename, path := getFileNameAndDir(input)
	tmpFile, err := fileutils.CreateTempFile(ctx, filename)
	if err != nil {
		return nil, nil, err
	}

	log := logger.GetLogger(ctx).WithFields(map[string]any{
		"bucket":    input.Bucket,
		"objectKey": input.Key,
		"version":   input.Version,
	})

	writeStreamFunc := func(chunk io.Reader) error {
		_, err := io.Copy(tmpFile, chunk)
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
			aio.mu.Unlock() // Be careful here
			return err
		}
		return nil
	}

	saveStreamFunc := func(success bool) (*erasure.FileInfo, error) {
		defer aio.mu.Unlock() // Be careful here
		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
		}()

		// Handling error case
		if !success {
			log.Warning("Write stream is closing unexpectly and deleting all hlf uploaded data")

			return nil, nil
		}

		stat, err := tmpFile.Stat()
		if err != nil {
			log.WithError(err).Errorf("Error getting temp file info for %s/%s", path, filename)
			return nil, err
		}

		if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek file: %w", err)
		}

		fileinfo, err := erasure.GetEncoder().Encode(ctx, filepath.Join(path, filename), stat.Size(), tmpFile)
		if err != nil {
			log.WithError(err).Errorf("Error writing file %s/%s", path, filename)
			return nil, err
		}

		return fileinfo, nil
	}

	return writeStreamFunc, saveStreamFunc, nil
}

func (aio *adaptableIO) Read(ctx context.Context, input ObjectInput) (*os.File, error) {
	aio.mu.Lock()
	defer aio.mu.Unlock()

	return aio.read(ctx, input)
}

func (aio *adaptableIO) read(ctx context.Context, input ObjectInput) (*os.File, error) {
	fileInfo := &erasure.FileInfo{
		Shards:       input.Shards,
		TotalSize:    input.TotalSize,
		DataShards:   input.DataShards,
		ParityShards: input.ParityShards,
		Checksum:     input.Checksum,
	}
	fileName, path := getFileNameAndDir(input)
	file, err := erasure.GetEncoder().Decode(ctx, filepath.Join(path, fileName), fileInfo)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Errorf("Error reading file for object %s/%s", input.Bucket, input.Key)
		return nil, err
	}

	return file, nil
}

func (aio *adaptableIO) ReadBytesRange(ctx context.Context, input ObjectInput, start, end int64) (*os.File, error) {
	aio.mu.Lock()
	defer aio.mu.Unlock()

	fileInfo := &erasure.FileInfo{
		Shards:       input.Shards,
		TotalSize:    input.TotalSize,
		DataShards:   input.DataShards,
		ParityShards: input.ParityShards,
		Checksum:     input.Checksum,
	}
	fileName, path := getFileNameAndDir(input)
	file, err := erasure.GetEncoder().DecodeBytesRange(ctx, filepath.Join(path, fileName), fileInfo, start, end)
	if err != nil {
		logger.GetLogger(ctx).WithError(err).Errorf("Error reading file for object %s/%s", input.Bucket, input.Key)
		return nil, err
	}

	return file, nil
}

func (aio *adaptableIO) Remove(ctx context.Context, input ObjectInput) error {
	log := logger.GetLogger(ctx)
	aio.mu.Lock()
	defer aio.mu.Unlock()

	if input.DoCleanup {
		// TODO: Parallelize this loop
		for i := range input.Shards {
			absPath := input.Shards[i].Path

			index := strings.Index(absPath, input.Id)
			if index == -1 {
				log.Errorf("Object uuid %s not found in shard path %s", input.Id, absPath)
				return fmt.Errorf("invalid shard path %s", absPath)
			}

			bucketPath := absPath[:index]
			targetDir := filepath.Join(bucketPath, input.Id)

			// TODO: Handle error
			log.Infof("Deleting file/dir: %s", targetDir)
			_ = os.RemoveAll(targetDir)
		}

		return nil
	}

	if input.Version != "" {
		_, relativePath := getFileNameAndDir(input)

		// TODO: Parallelize this loop
		for i := range input.Shards {
			absPath := input.Shards[i].Path

			// Split absPath by the relative path to find the base
			index := strings.Index(absPath, relativePath)
			if index == -1 {
				log.Infof("Relative path %s not found in absPath %s", relativePath, absPath)
				return fmt.Errorf("invalid shard path %s", absPath)
			}

			diskPath := absPath[:index]
			targetDir := filepath.Join(diskPath, relativePath)
			// TODO: Handle error
			log.Infof("Deleting file/dir: %s", targetDir)
			_ = os.RemoveAll(targetDir)
		}

		return nil
	}

	// TODO: Parallelize this loop
	for i := range input.Shards {
		log.Infof("Deleting file/dir: %s", input.Shards[i].Path)
		// TODO: Handle error
		_ = os.Remove(input.Shards[i].Path)
	}

	return nil
}

func getFileNameAndDir(input ObjectInput) (string, string) {
	basepath := filepath.Join(namespace.DEFAULT_NAMESPACE, input.Bucket, input.Id)
	if input.Version != "" {
		basepath = filepath.Join(basepath, input.Version)
	}

	filename := filepath.Base(input.Key)
	return filename, basepath
}
