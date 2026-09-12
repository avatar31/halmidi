package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/avatar31/omashu"
	"github.com/dgraph-io/badger/v4"

	dbstore "github.com/avatar31/halmidi/internal/db_store"
	"github.com/avatar31/halmidi/internal/logger"
	"github.com/avatar31/halmidi/utils"
)

var (
	ErrEmptyQueue          = errors.New("queue is empty")
	ErrNoMsgToAckOrRequeue = errors.New("no message to ack/requeue")
)

type Queue interface {
	Count() int
	IsEmpty() bool
	GetHead() uint64
	GetTail() uint64
	GetName() string
	Enqueue(ctx context.Context, msg []byte) error
	Dequeue(ctx context.Context) ([]byte, error)
	Peek(ctx context.Context) ([]byte, error)
	Ack(ctx context.Context) error
	Flush(ctx context.Context) error
	Requeue(ctx context.Context) error
}

type BadgerQueue struct {
	mu       sync.Mutex
	name     string
	prefix   string
	head     uint64 // read pointer
	tail     uint64 // write pointer
	lastPeek []byte // store last peeked item
	lastKey  string // store key of last peek
}

func GetOrCreateQueue(ctx context.Context, name string) (Queue, bool, error) {
	q := BadgerQueue{
		name:   name,
		prefix: dbstore.GetSystemQueueDBKeyWithNS(name),
	}

	// Load head/tail on startup
	newQueue, err := q.loadPointers(ctx)
	if err != nil {
		return nil, false, err
	}

	if newQueue {
		logger.GetLogger(ctx).WithField("queue", name).Info("Created new queue")
	}

	return &q, newQueue, nil
}

func (q *BadgerQueue) Count() int {
	return int(q.tail - q.head)
}

func (q *BadgerQueue) IsEmpty() bool {
	return q.head == q.tail
}

func (q *BadgerQueue) loadPointers(ctx context.Context) (bool, error) {
	headKey := getQueueHeadKey(q.prefix)
	tailKey := getQueueTailKey(q.prefix)
	newQueue := false

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	log.Info("Loading queue details from db")
	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		headBytes, exist, err := qdb.GetWithTxn(ctx, txn, headKey)
		if err != nil {
			return err
		}
		if !exist {
			_ = txn.Set(ctx, headKey, utils.Uint64ToBytes(0))
			_ = txn.Set(ctx, tailKey, utils.Uint64ToBytes(0))
			newQueue = true
			return nil
		}
		q.head = utils.BytesToUint64(headBytes)

		tailBytes, exist, err := qdb.GetWithTxn(ctx, txn, tailKey)
		if err != nil {
			return err
		}

		if exist {
			q.tail = utils.BytesToUint64(tailBytes)
		} else {
			q.tail = q.head
		}

		return nil
	})

	log.Info("Loaded queue details from db")
	return newQueue, err
}

func (q *BadgerQueue) Enqueue(ctx context.Context, msg []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap()).WithField("queueMessage", string(msg))
	log.Info("Enqueueing message")

	tail := q.tail
	key := getQueueItemKey(q.prefix, tail)
	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, key, msg)
		if err != nil {
			return err
		}

		err = txn.IncrBy(ctx, getQueueTailKey(q.prefix), 1)
		tail++
		return err
	})
	if err != nil {
		return err
	}

	q.tail = tail

	log.Info("Enqueued message")
	return nil
}

func (q *BadgerQueue) Dequeue(ctx context.Context) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	log.Info("Dequeueing message")

	if q.IsEmpty() {
		log.Info("Queue is empty")
		return nil, nil // Empty queue
	}

	head := q.head
	key := getQueueItemKey(q.prefix, head)
	var msg []byte
	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		var exist bool
		msg, exist, _ = qdb.GetWithTxn(ctx, txn, key)
		if !exist {
			// We shouldn't come here
			log.Warning("Queue tail and head pointers are mislocated")
			return errors.New("interanl error")
		}

		// Delete after reading
		err := txn.Delete(ctx, key)
		if err != nil {
			return err
		}

		err = txn.IncrBy(ctx, getQueueHeadKey(q.prefix), 1)
		head++
		return err
	})

	if err != nil {
		log.WithError(err).Error("Error while dequeueing message")
		return nil, err
	}

	q.head = head

	log.WithField("queueMessage", string(msg)).Info("Dequeued message")
	return msg, nil
}

// On Peek → Read the value but do not delete
func (q *BadgerQueue) Peek(ctx context.Context) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.IsEmpty() {
		return nil, ErrEmptyQueue
	}

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	key := getQueueItemKey(q.prefix, q.head)
	qdb := dbstore.GetDBStore(ctx)
	msg, exist, _ := qdb.Get(ctx, key)
	if !exist {
		// We shouldn't come here
		log.Warning("Queue tail and head pointers are mislocated")
		return nil, errors.New("interanl error")
	}

	q.lastPeek = msg
	q.lastKey = key

	return msg, nil
}

// On Ack → Delete the current head
// Confirm successful processing (delete after peek)
func (q *BadgerQueue) Ack(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.lastKey == "" {
		return ErrNoMsgToAckOrRequeue
	}

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	log.WithFields(map[string]any{"key": q.lastKey, "queueMsg": string(q.lastPeek)}).Info("Acknowledging Message.")

	head := q.head
	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Delete(ctx, q.lastKey)
		if err != nil {
			return err
		}

		err = txn.IncrBy(ctx, getQueueHeadKey(q.prefix), 1)
		head++
		return err
	})

	if err != nil {
		log.WithError(err).Error("Error while acknowledging message")
		return err
	}

	q.head = head
	q.lastKey = ""
	q.lastPeek = nil
	return nil
}

// On Requeue → Enqueue the item again at the tail, and optionally advance the head
// Put the peeked item back (e.g. if failed)
// Push back failed work strategy
func (q *BadgerQueue) Requeue(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.lastPeek == nil {
		return ErrNoMsgToAckOrRequeue
	}

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	log.WithFields(map[string]any{"key": q.lastKey, "queueMsg": string(q.lastPeek)}).Info("Requeueing Message.")

	// Enqueue at tail
	head := q.head
	tail := q.tail
	newKey := getQueueItemKey(q.prefix, tail)
	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		err := txn.Set(ctx, newKey, q.lastPeek)
		if err != nil {
			return err
		}

		err = txn.Delete(ctx, q.lastKey)
		if err != nil {
			return err
		}

		err = txn.IncrBy(ctx, getQueueTailKey(q.prefix), 1)
		if err != nil {
			return err
		}
		tail++

		err = txn.IncrBy(ctx, getQueueHeadKey(q.prefix), 1)
		head++
		return err
	})

	if err != nil {
		log.WithError(err).Error("Error while requeueing message")
		return err
	}

	q.head = head
	q.tail = tail
	q.lastKey = ""
	q.lastPeek = nil
	return nil
}

func (q *BadgerQueue) Flush(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.IsEmpty() {
		return nil
	}

	log := logger.GetLogger(ctx).WithFields(q.getQueueFieldsMap())
	log.Info("Flusing Queue")

	qdb := dbstore.GetDBStore(ctx)
	err := qdb.NewTransaction(ctx, func(ctx context.Context, txn *omashu.Txn) error {
		// Delete all queue items
		for i := q.head; i <= q.tail; i++ {
			key := getQueueItemKey(q.prefix, i)
			if err := txn.Delete(ctx, key); err != nil && err != badger.ErrKeyNotFound {
				return err
			}
		}

		// Delete metadata
		if err := txn.Delete(ctx, getQueueHeadKey(q.prefix)); err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if err := txn.Delete(ctx, getQueueTailKey(q.prefix)); err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		return nil
	})

	if err != nil {
		log.WithError(err).Error("Error while flushing queue")
		return err
	}

	q.head = 0
	q.tail = 0
	q.lastKey = ""
	q.lastPeek = nil
	return nil
}

func (q *BadgerQueue) GetHead() uint64 {
	return q.head
}

func (q *BadgerQueue) GetTail() uint64 {
	return q.tail
}

func (q *BadgerQueue) GetName() string {
	return q.name
}

func getQueueHeadKey(queuePrefix string) string {
	return fmt.Sprintf("%s:head", queuePrefix)
}

func getQueueTailKey(queuePrefix string) string {
	return fmt.Sprintf("%s:tail", queuePrefix)
}

func getQueueItemKey(queuePrefix string, headOrTail uint64) string {
	return fmt.Sprintf("%s:%012d", queuePrefix, headOrTail)
}

func (q *BadgerQueue) getQueueFieldsMap() map[string]any {
	return map[string]any{
		"queue": q.name,
		"head":  q.head,
		"tail":  q.tail,
		"count": q.Count(),
	}
}
