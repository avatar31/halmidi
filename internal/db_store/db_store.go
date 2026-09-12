package dbstore

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/avatar31/omashu"
	"github.com/dgraph-io/badger/v4"
	"go.etcd.io/raft/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/logger"
)

var (
	appDB    *omashu.DistributedBadger
	once     sync.Once
	dbLogger *zap.Logger
)

func InitDBStore(ctx context.Context, id uint64, nodename string, peers map[uint64]string) {
	once.Do(func() {
		dbPath := filepath.Join(config.GetConfig().MetaDataPath, "metadb")
		protoSchema, err := models.LoadModelsDescriptorSet()
		if err != nil {
			logger.GetLogger(ctx).WithError(err).Panic("Failed to load protobuf schema for DBStore")
		}

		dbLogger = logger.NewZapLogger(config.APP_NAME + ".omashu")

		c := Cluster{}
		cfg := omashu.Config{
			Name:           config.APP_NAME,
			BaseDir:        dbPath,
			GCInterval:     10 * time.Minute, // TODO: P1: Is 10 mins is suitable interval?
			GCDiscardRatio: 0.5,
			BadgerOptions:  badger.DefaultOptions(""),
			Cluster:        c,
			Logger:         dbLogger,
			RaftConfig: &omashu.RaftConfig{
				Nodename: nodename,
				Peers:    peers,
				// TODO: P0: These values are just placeholders,
				// 	we need to tune them based on our workload and environment
				Config: raft.Config{
					ID:                        id,
					ElectionTick:              10,          // 1s
					HeartbeatTick:             1,           // 100ms
					MaxSizePerMsg:             1024 * 1024, // 1 MB		// TODO: P0: Tune this value
					MaxInflightMsgs:           256,         // TODO: P0: Tune this value
					CheckQuorum:               true,
					PreVote:                   true,
					DisableProposalForwarding: true,    // rest middleware will take care of forwarding
					MaxUncommittedEntriesSize: 1 << 26, // 64MB
				},
			},
			SchemaConfig: &omashu.SchemaConfig{
				Type:            omashu.SchemaTypeProtobuf,
				ProtoSchemaList: []*descriptorpb.FileDescriptorSet{protoSchema},
			},
		}

		appDB, err = omashu.NewDistributedBadger(ctx, &cfg)
		if err != nil {
			logger.GetLogger(ctx).WithError(err).Panic("Failed to initialize DBStore")
		}
	})
}

func GetDBStore(ctx context.Context) *omashu.DistributedBadger {
	return appDB
}

func CloseDB(ctx context.Context) {
	if appDB != nil {
		appDB.Close(ctx)
	}

	if dbLogger != nil {
		if err := dbLogger.Sync(); err != nil {
			logger.GetLogger(ctx).WithError(err).Error("Error while syncing dbstore zap logs")
		}
	}
}

type Cluster struct{}

func (c Cluster) GetID() uint64 {
	return 1
}

func (c Cluster) GetName() string {
	return config.APP_NAME
}

func (c Cluster) IsNodeRemoved(id uint64) bool {
	return false
}
