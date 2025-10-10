package topic

import (
	"context"
	"sync"
	"time"

	"github.com/axmq/ax/store"
	"github.com/axmq/ax/types/message"
)

type RetainedManager struct {
	store           *store.RetainedStore
	cleanupTicker   *time.Ticker
	cleanupInterval time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	onCleanup       func(count int)
}

type RetainedConfig struct {
	CleanupInterval time.Duration
	OnCleanup       func(count int)
}

func DefaultRetainedConfig() *RetainedConfig {
	return &RetainedConfig{
		CleanupInterval: 5 * time.Minute,
	}
}

func NewRetainedManager(config *RetainedConfig) *RetainedManager {
	if config == nil {
		config = DefaultRetainedConfig()
	}

	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	rm := &RetainedManager{
		store:           store.NewRetainedStore(),
		cleanupInterval: config.CleanupInterval,
		cleanupTicker:   time.NewTicker(config.CleanupInterval),
		stopCh:          make(chan struct{}),
		onCleanup:       config.OnCleanup,
	}

	rm.wg.Add(1)
	go rm.cleanupLoop()

	return rm
}

func (rm *RetainedManager) Set(ctx context.Context, topic string, msg *message.Message) error {
	return rm.store.Set(ctx, topic, msg)
}

func (rm *RetainedManager) Get(ctx context.Context, topic string) (*message.Message, error) {
	return rm.store.Get(ctx, topic)
}

func (rm *RetainedManager) Delete(ctx context.Context, topic string) error {
	return rm.store.Delete(ctx, topic)
}

func (rm *RetainedManager) Match(ctx context.Context, topicFilter string, matcher store.TopicMatcher) ([]*message.Message, error) {
	return rm.store.Match(ctx, topicFilter, matcher)
}

func (rm *RetainedManager) Count(ctx context.Context) (int64, error) {
	return rm.store.Count(ctx)
}

func (rm *RetainedManager) cleanupLoop() {
	defer rm.wg.Done()

	for {
		select {
		case <-rm.cleanupTicker.C:
			rm.cleanup()
		case <-rm.stopCh:
			return
		}
	}
}

func (rm *RetainedManager) cleanup() {
	ctx := context.Background()
	count, err := rm.store.CleanupExpired(ctx)
	if err == nil && count > 0 && rm.onCleanup != nil {
		rm.onCleanup(count)
	}
}

func (rm *RetainedManager) Close() error {
	close(rm.stopCh)
	rm.cleanupTicker.Stop()
	rm.wg.Wait()
	return rm.store.Close()
}
