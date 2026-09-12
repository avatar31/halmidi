package fileutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/avatar31/halmidi/config"
	osfileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
	"github.com/avatar31/halmidi/internal/logger"
)

const (
	TMP_FILES_DIR_FORMAT = "2006_01_02_15_04_05"
)

func CreateTempFile(ctx context.Context, prefix string) (*os.File, error) {
	tmpDir := config.GetTmpDir()
	childDir := time.Now().UTC().Truncate(time.Hour).Format(TMP_FILES_DIR_FORMAT)

	err := osfileutils.CreateDirIfNotExists(filepath.Join(tmpDir, childDir))
	if err != nil {
		return nil, err
	}

	p := "tmp_*"
	if prefix != "" {
		p = fmt.Sprintf("tmp_%s_*", prefix)
	}
	f, err := os.CreateTemp(filepath.Join(tmpDir, childDir), p)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	// TODO: Check why NewAutoCleanupFile(f, ctx).File is not working
	return f, nil
}

// autoCleanupFile wraps an os.File and automatically removes it when closed
type autoCleanupFile struct {
	*os.File
	ctx context.Context
}

func NewAutoCleanupFile(f *os.File, ctx context.Context) *autoCleanupFile {
	return &autoCleanupFile{File: f, ctx: ctx}
}

func (a *autoCleanupFile) Close() error {
	name := a.Name()
	err := a.File.Close()
	if err != nil {
		return err
	}

	logger.GetLogger(a.ctx).Infof("Auto-cleanup temp file: %s", name)

	// Remove the file after closing
	if err := os.Remove(name); err != nil {
		logger.GetLogger(a.ctx).WithError(err).Warningf("Failed to auto-cleanup temp file: %s", name)
		return err
	}

	return nil
}

type TmpFilesCleaner struct {
	Name string
	Spec string
}

func NewTmpFilesCleaner(ctx context.Context) *TmpFilesCleaner {
	return &TmpFilesCleaner{
		Name: "TmpFilesCleaner",
		Spec: "0 */2 * * *", // Every 2 hours
	}
}

func (c TmpFilesCleaner) SchedHandler(ctx context.Context) {
	tmpDir := config.GetTmpDir()
	now := time.Now().UTC()
	log := logger.GetLogger(ctx).WithField("schedular", c.Name)

	err := filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		t, err := time.Parse(TMP_FILES_DIR_FORMAT, d.Name())
		if err != nil {
			return nil // skip non-matching names
		}

		// KEEP if newer than 2 hours
		if now.Sub(t) < 2*time.Hour {
			return nil
		}

		// DELETE if older than 2 hours
		err = os.RemoveAll(path)
		if err != nil {
			log.WithError(err).Errorf("Failed to remove tmp files directory: %s", d.Name())
			return err
		}

		log.Infof("Removed tmp files directory: %s", d.Name())
		return nil
	})

	if err != nil {
		log.WithError(err).Errorf("Error while cleaning tmp files dir")
	}

	// TODO: Add multipart upload temp files cleanup
}
