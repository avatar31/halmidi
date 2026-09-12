package objectio

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/avatar31/halmidi/internal/persistent/storage/erasure"
)

var (
	ErrMissingVersion = errors.New("missing object version")
)

type writeStream func(io.Reader) error
type saveStream func(success bool) (*erasure.FileInfo, error)

type ObjectIO interface {
	CreateStreamForWrite(ctx context.Context, input ObjectInput) (writeStream, saveStream, error)
	Write(ctx context.Context, input ObjectInput) (*erasure.FileInfo, error)
	Copy(ctx context.Context, src, dest ObjectInput) (*erasure.FileInfo, error)
	Read(ctx context.Context, input ObjectInput) (*os.File, error)
	ReadBytesRange(ctx context.Context, input ObjectInput, start, end int64) (*os.File, error)
	Remove(ctx context.Context, input ObjectInput) error
}

type ObjectInput struct {
	Bucket  string
	Key     string
	Id      string
	Version string

	// Object Write Specific field which contains data to write
	Data       io.Reader
	ContentLen int64

	// Object Read/Remove Specific field which contains shard info
	Shards       []erasure.Shard
	DataShards   int32
	ParityShards int32
	TotalSize    int64
	Checksum     string

	// Object Remove Specific field
	DoCleanup bool
}

func NewObjectIO() ObjectIO {
	return &adaptableIO{}
}
