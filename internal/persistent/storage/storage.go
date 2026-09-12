package storage

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/storage/disk"
	"github.com/avatar31/halmidi/internal/persistent/storage/erasure"
	"github.com/avatar31/halmidi/utils"
)

var (
	disks []disk.Disk
)

func Initialize(ctx context.Context) {
	log := logger.GetLogger(ctx)
	volumes, err := expandPathPattern(config.GetConfig().Volumes)
	if err != nil {
		log.WithError(err).Panic("Failed to expand volume paths")
	}

	disks, err = disk.ValidateDisks(ctx, volumes)
	if err != nil {
		log.WithError(err).Panic("Disk validation failed")
	}

	err = erasure.Init(ctx, disks)
	if err != nil {
		log.WithError(err).Panic("Failed to initialize erasure coding")
	}

	log.Infof("Initialized storage with volumes: %v", volumes)
}

func GetDisks() []disk.Disk {
	return disks
}

func expandPathPattern(pattern string) ([]string, error) {
	// Find "{start...end}" pattern
	re := regexp.MustCompile(`\{(\d+)\.\.\.(\d+)\}`)
	matches := re.FindStringSubmatch(pattern)
	if len(matches) != 3 {
		// No match, return the original as-is
		return nil, fmt.Errorf("invalid volumes: %s", pattern)
	}

	start := utils.AtoiDefault(matches[1], 0)
	end := utils.AtoiDefault(matches[2], 0)

	var paths []string
	for i := start; i <= end; i++ {
		expanded := strings.Replace(pattern, matches[0], fmt.Sprintf("%d", i), 1)
		paths = append(paths, expanded)
	}

	return paths, nil
}
