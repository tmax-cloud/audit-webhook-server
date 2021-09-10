package audit

import (
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	auditDataFactory "github.com/tmax-cloud/audit-webhook-server/dataFactory"
	"k8s.io/apiserver/pkg/apis/audit"
)

var EventBuffer buffer

func init() {
	EventBuffer = newBuffer()
	EventBuffer.run()
}

const (
	BufferSize int           = 256
	batchSize  int           = 16
	batchWait  time.Duration = time.Second * 10
)

type buffer struct {
	Buffer chan audit.Event

	batchSize int

	batchWait time.Duration

	wg sync.WaitGroup
}

func newBuffer() buffer {
	return buffer{
		Buffer:    make(chan audit.Event, BufferSize),
		batchSize: batchSize,
		batchWait: batchWait,
		wg:        sync.WaitGroup{},
	}
}

func (b *buffer) run() {
	go func() {
		defer close(b.Buffer)
		var (
			maxWaitChan  <-chan time.Time
			maxWaitTimer *time.Timer
		)

		maxWaitTimer = time.NewTimer(b.batchWait)
		maxWaitChan = maxWaitTimer.C

		defer maxWaitTimer.Stop()
		for {
			maxWaitTimer.Reset(b.batchWait)
			eventList := audit.EventList{}

			if eventList.Items = b.collectEvents(maxWaitChan); len(eventList.Items) != 0 {
				b.wg.Add(1)
				go func() {
					defer b.wg.Done()
					auditDataFactory.Insert(eventList.Items)
				}()

				b.wg.Add(1)
				go func() {
					defer b.wg.Done()
					if len(hub.clients) > 0 {
						hub.broadcast <- eventList
					}
				}()
				b.wg.Wait()
			}
		}
	}()
}

func (b *buffer) collectEvents(timer <-chan time.Time) []audit.Event {
	var events []audit.Event

L:
	for i := 0; i < b.batchSize; i++ {
		select {
		case ev, ok := <-b.Buffer:
			if !ok {
				break L
			}
			events = append(events, ev)
		case <-timer:
			// Timer has expired. Send currently accumulated batch.
			break L
		}
	}

	return events
}
