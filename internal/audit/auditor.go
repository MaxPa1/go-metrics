// Package audit реализует аудит изменений метрик по паттерну «Наблюдатель».
package audit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"go.uber.org/zap"
)

const queueSize = 1024

// Observer получает события аудита.
type Observer interface {
	Notify(ctx context.Context, e Event) error
}

// Auditor асинхронно рассылает события аудита всем наблюдателям.
type Auditor struct {
	log       *zap.SugaredLogger
	observers []Observer
	queue     chan Event
	done      chan struct{}

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// NewAuditor создаёт аудитор и запускает фоновую рассылку событий.
func NewAuditor(log *zap.SugaredLogger, observers ...Observer) *Auditor {
	ctx, cancel := context.WithCancel(context.Background())
	a := &Auditor{
		log:       log.With("component", "audit"),
		observers: observers,
		queue:     make(chan Event, queueSize),
		done:      make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
	}
	go a.run()
	return a
}

// Notify ставит событие в очередь на рассылку.
func (a *Auditor) Notify(e Event) {
	if len(a.observers) == 0 {
		return
	}
	select {
	case a.queue <- e:
	default:
		a.log.Warn("audit queue is full, event dropped")
	}
}

func (a *Auditor) run() {
	defer close(a.done)
	for e := range a.queue {
		for _, o := range a.observers {
			if err := o.Notify(a.ctx, e); err != nil {
				a.log.Errorw("failed to deliver audit event", "error", err)
			}
		}
	}
}

// Close дожидается рассылки оставшихся событий и останавливает аудитор.
func (a *Auditor) Close(ctx context.Context) error {
	var errs []error
	a.closeOnce.Do(func() {
		close(a.queue)
		select {
		case <-a.done:
		case <-ctx.Done():
			errs = append(errs, fmt.Errorf("audit: drain queue: %w", ctx.Err()))
			a.cancel()
			<-a.done
		}
		a.cancel()

		for _, o := range a.observers {
			if c, ok := o.(io.Closer); ok {
				if err := c.Close(); err != nil {
					errs = append(errs, fmt.Errorf("audit: close observer: %w", err))
				}
			}
		}
	})
	return errors.Join(errs...)
}
