package notification

import (
	"context"
	"sync"

	"github.com/wyw14/cry-071/internal/application"
)

type LocalSink struct {
	mu            sync.RWMutex
	notifications []application.Notification
}

func NewLocalSink() *LocalSink { return &LocalSink{} }

func (s *LocalSink) Send(ctx context.Context, notification application.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyNotification := notification
	copyNotification.Data = cloneMap(notification.Data)
	s.notifications = append(s.notifications, copyNotification)
	return nil
}

func (s *LocalSink) List() []application.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]application.Notification, len(s.notifications))
	for index, item := range s.notifications {
		item.Data = cloneMap(item.Data)
		result[index] = item
	}
	return result
}

func cloneMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
