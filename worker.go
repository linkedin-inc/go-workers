package workers

import (
	"fmt"
	"time"
)

type worker struct {
	manager    *manager
	stop       chan bool
	exit       chan bool
	currentMsg *Msg
	startedAt  int64
}

func (w *worker) start() {
	go w.work(w.manager.fetch.Messages())
}

func (w *worker) quit() {
	w.stop <- true
	<-w.exit
}

func (w *worker) work(messages chan *Msg) {
	for {
		select {
		case message := <-messages:
			w.startedAt = time.Now().UTC().Unix()
			w.currentMsg = message

			fmt.Println("[work] work:", message)
			t1 := time.Now().UnixNano()
			if w.process(message) {
				w.manager.confirm <- message
			}
			t2 := time.Now().UnixNano()
			fmt.Println("[work] work cost: ", float64(t2-t1)/float64(1e6))
			w.startedAt = 0
			w.currentMsg = nil
		case w.manager.fetch.Ready() <- true:
			// Signaled to fetcher that we're
			// ready to accept a message
		case <-w.stop:
			w.exit <- true
			break
		}
	}
}

func (w *worker) process(message *Msg) (acknowledge bool) {
	acknowledge = true

	defer func() {
		recover()
	}()

	return w.manager.mids.call(w.manager.queueName(), message, func() {
		w.manager.job(message)
	})
}

func (w *worker) processing() bool {
	return w.startedAt > 0
}

func newWorker(m *manager) *worker {
	return &worker{m, make(chan bool), make(chan bool), nil, 0}
}
