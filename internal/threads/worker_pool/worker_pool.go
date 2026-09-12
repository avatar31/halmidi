package workerpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/internal/persistent/queue"
)

var (
	EmptyQueueIdleTime = 1 * time.Minute
)

type WorkerPool struct {
	Queue   queue.Queue
	Workers int
	Handler func(context.Context, []byte) error
}

func NewWorkerPool(q queue.Queue, count int, handler func(context.Context, []byte) error) *WorkerPool {
	return &WorkerPool{Queue: q, Workers: count, Handler: handler}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	log := logger.GetLogger(ctx)
	var wg sync.WaitGroup
	for i := 0; i < wp.Workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			wp.workerLoop(ctx, id)
		}(i)
	}

	// Wait until context is canceled
	<-ctx.Done()
	log.Debug("Received stop signal from system")

	wg.Wait()
	log.Info("All workers stopped")
}

func (wp *WorkerPool) workerLoop(ctx context.Context, id int) {
	log := logger.GetLogger(ctx).WithFields(map[string]any{
		"queue":    wp.Queue.GetName(),
		"workerId": id,
	})

	log.Info("Worker Started")
	now := time.Now()
	workerStartTime := now
	idleStart := now
	processedMessages := 0
	for {
		select {
		case <-ctx.Done():
			log.WithFields(map[string]any{
				"processedMessages": processedMessages,
				"elapsedTime":       fmt.Sprint(time.Since(workerStartTime)),
			}).Info("Received stop signal from system. Stopping worker.")
			return

		default:
			// Peek at next message
			msg, err := wp.Queue.Peek(ctx)
			if err != nil {
				if errors.Is(err, queue.ErrEmptyQueue) {
					if time.Since(idleStart) > EmptyQueueIdleTime {
						log.WithFields(map[string]any{
							"processedMessages": processedMessages,
							"elapsedTime":       fmt.Sprint(time.Since(workerStartTime)),
						}).Infof("Waited %s, no more messages in Queue. Stopping worker.", idleStart)
						return
					}
					time.Sleep(1 * time.Second)
				} else {
					// TODO: Should we stop after retry
					log.WithError(err).Error("Error while peeking message from Queue")
					time.Sleep(500 * time.Millisecond)
				}
				continue
			}

			// Process
			err = wp.Handler(ctx, msg)
			if err != nil {
				log.WithError(err).WithField("queueMsg", string(msg)).Error("Handler Failed")

				// Put it back at end
				if err = wp.Queue.Requeue(ctx); err != nil {
					log.WithError(err).WithField("queueMsg", string(msg)).Error("Requeue Failed")
					// TODO: Retry
					continue
				}
			} else {
				// Confirm and remove
				if err = wp.Queue.Ack(ctx); err != nil {
					log.WithError(err).WithField("queueMsg", string(msg)).Error("Message Acknowledge Failed")
					// TODO: Retry
					continue
				}
				processedMessages++
			}
			idleStart = time.Now()
		}
	}
}
