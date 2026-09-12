package disk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/avatar31/halmidi/config"
	fileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
	"github.com/avatar31/halmidi/internal/logger"
)

type Disk struct {
	ID   int
	Path string
}

const (
	MinDisksRequired  = 6
	MaxDisksSupported = 16
)

// ValidateDisks checks if all disks are accessible and writable
// TODO: It should validate if disks are on separate physical devices via network
func ValidateDisks(ctx context.Context, diskList []string) ([]Disk, error) {
	total := len(diskList)
	if total < MinDisksRequired {
		return nil, fmt.Errorf("minimum %d disks required", MinDisksRequired)
	}

	if total > MaxDisksSupported {
		return nil, fmt.Errorf("maximum %d disks supported", MaxDisksSupported)
	}

	errChan := make(chan error, total)
	var wg sync.WaitGroup

	temp := map[string]struct{}{}
	for _, disk := range diskList {
		// Check for duplicate disk paths
		if _, exists := temp[disk]; exists {
			errChan <- fmt.Errorf("duplicate disk path detected: %s", disk)
			break
		}
		temp[disk] = struct{}{}

		wg.Add(1)
		go func(d string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			// Check if disk exists
			if info, err := os.Stat(d); os.IsNotExist(err) {
				errChan <- fmt.Errorf("disk %s: not found: %w", d, err)
				return
			} else if !info.IsDir() {
				errChan <- fmt.Errorf("disk %s: not a directory", d)
				return
			}

			// Test write
			testFile := filepath.Join(d, ".write_test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				errChan <- fmt.Errorf("disk %s: not writable: %w", d, err)
				return
			}
			_ = os.Remove(testFile)
		}(disk)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("disk validation failed: %v", errs)
	}

	disks := make([]Disk, total)
	for i, diskPath := range diskList {
		fullPath := filepath.Join(diskPath, config.APP_NAME)
		if err := fileutils.CreateDirIfNotExists(fullPath); err != nil {
			logger.GetLogger(ctx).WithError(err).Errorf("Failed to create basepath directory: %s", fullPath)
			return nil, err
		}
		disks[i] = Disk{ID: i, Path: fullPath}
	}

	return disks, nil
}
