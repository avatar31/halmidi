package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/core/models"
	"github.com/avatar31/halmidi/internal/core/s3common"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/queue"
	workerpool "github.com/avatar31/halmidi/internal/threads/worker_pool"
)

const (
	expiredObjectDeleteJobWorkerCount = 3
)

type msg struct {
	Bucket string
	Key    string
}

type ObjectExpiryTracker struct {
	Name  string
	Spec  string
	queue queue.Queue
}

func NewObjectExpiryTracker(ctx context.Context) (*ObjectExpiryTracker, error) {
	// TODO: Read from env variable
	// Runs every day midnight at 12:30 AM in Production
	spec := "30 0 * * *"
	if config.IsDevEnv() {
		// Runs every 10 mins in Dev environment
		spec = "*/10 * * * *"
	}

	q, newQueue, err := queue.GetOrCreateQueue(ctx, "expiredObjects")
	if err != nil {
		return nil, err
	}

	// TODO: Revisit
	empty := q.IsEmpty()
	if !newQueue && empty {
		_ = q.Flush(ctx)
	}

	tracker := &ObjectExpiryTracker{Name: "ObjectExpiryTracker", Spec: spec, queue: q}
	if !empty {
		// Restarting workerpool in case of accidental restart
		wp := workerpool.NewWorkerPool(q, expiredObjectDeleteJobWorkerCount, tracker.WorkerHandler)
		wp.Start(ctx)
	}

	return tracker, nil
}

func (tracker ObjectExpiryTracker) SchedHandler(ctx context.Context) {
	log := logger.GetLogger(ctx).WithField("schedular", tracker.Name)
	log.Info("Wokeup for job execution")
	start := time.Now()

	// TODO: Should we flush table before starting
	wp := workerpool.NewWorkerPool(tracker.queue, expiredObjectDeleteJobWorkerCount, tracker.WorkerHandler)
	wp.Start(ctx)

	bucketService := NewBucketService(ctx)
	listBucketOpts := ListBucketOptions{
		Prefix:   "",
		MaxItems: 10000,
	}
	buckets, _, err := bucketService.ListBuckets(ctx, listBucketOpts)
	if err != nil {
		// TODO: P0: handle err
		log.WithError(err).Error("Error while listing buckets")
	}

	for _, bucket := range buckets {
		if !bucket.IsBucketLifecycleConfigured() {
			continue
		}

		objectService := newObjectServiceInt(bucketService.log, bucketService.bdb, bucket)
		opts := &ListObjectOptions{
			Prefix:     "",
			Delimiter:  "",
			StartAfter: "",
			MaxKeys:    10000,
		}
		objects, _, _, err := objectService.ListObjects(ctx, opts)
		if err != nil {
			// TODO: P0: handle err
			log.WithError(err).WithField("bucket", bucket.Name).Error("Error while listing objects for bucket")
		}

		for _, rule := range bucket.LifecycleConfig.Rules {
			if !rule.IsEnabled() {
				continue
			}

			for _, obj := range objects {
				if shouldExpire(ctx, rule, obj) {
					msg, _ := json.Marshal(msg{Bucket: bucket.Name, Key: obj.Key})
					err := tracker.queue.Enqueue(ctx, msg)
					if err != nil {
						log.WithError(err).WithFields(map[string]any{
							"bucket": bucket.Name,
							"key":    obj.Key,
						}).Error("Error while scheduling expired object to delete")
						continue
					}
				}
			}
		}
	}

	log.WithField("elapsedTime", fmt.Sprint(time.Since(start))).Info("Job completed. Going to Sleep..")
}

func (tracker ObjectExpiryTracker) WorkerHandler(ctx context.Context, msgBytes []byte) error {
	var m msg
	err := json.Unmarshal(msgBytes, &m)
	if err != nil {
		return err
	}

	service, err := NewObjectService(ctx, m.Bucket)
	if err != nil {
		return err
	}

	e, err := service.GetObjectEntity(ctx, m.Key)
	if err != nil {
		return err
	}

	_, err = e.DeleteObject(ctx, "", false)
	if err != nil {
		if s3Err, ok := err.(s3common.S3Error); ok && s3Err.S3ErrorCode == s3common.NoSuchKey {
			return nil
		}
		return err
	}

	return nil
}

func shouldExpire(ctx context.Context, rule *models.LifeCycleRule, obj *models.ObjectMeta) bool {
	// now := time.Now()

	// // --- Skip if rule doesn't match object (prefix/tags/size)
	// if !rule.ObjectMatchesRule(rule, obj) {
	// 	return false
	// }

	// // --- Check ExpiredObjectDeleteMarker (for delete markers)
	// if obj.IsDeleteMarker && rule.Expiration != nil &&
	// 	rule.Expiration.ExpiredObjectDeleteMarker != nil &&
	// 	rule.Expiration.ExpiredObjectDeleteMarker.Enabled {
	// 	return true
	// }

	// // --- Check Days since LastModified
	// if rule.Expiration != nil && rule.Expiration.Days != nil {
	// 	days := *rule.Expiration.Days
	// 	expireDate := obj.LastModified.Add(time.Duration(days) * 24 * time.Hour)
	// 	if now.After(expireDate) {
	// 		return true
	// 	}
	// }

	return false
}
