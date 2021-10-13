package engine

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"

	shell "github.com/ipfs/go-ipfs-api"
	logging "github.com/ipfs/go-log"
	pinning "github.com/ipfs/go-pinning-service-http-client"

	"github.com/coryschwartz/gateway-monitor/pkg/queue"
	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

var log = logging.Logger("engine")

type Engine struct {
	c    *cron.Cron
	q    *queue.TaskQueue
	sh   *shell.Shell
	ps   *pinning.Client
	gw   string
	done chan bool
}

// Create an engine with Cron and Prometheus setup
func New(sh *shell.Shell, ps *pinning.Client, gw string, tsks ...task.Task) *Engine {
	eng := Engine{
		c:    cron.New(),
		q:    queue.NewTaskQueue(),
		sh:   sh,
		ps:   ps,
		gw:   gw,
		done: make(chan bool),
	}

	for _, t := range tsks {
		reg := t.Registration()
		eng.c.AddFunc(reg.Schedule, func() {
			log.Info("queueing scheduled task")
			eng.q.Push(t)
		})
		for _, col := range reg.Collectors {
			prometheus.Register(col)
		}
	}
	eng.c.Start()
	return &eng
}

// Create an engine without Cron and prometheus.
func NewSingle(sh *shell.Shell, ps *pinning.Client, gw string, tsks ...task.Task) *Engine {
	eng := Engine{
		c:    cron.New(),
		q:    queue.NewTaskQueue(),
		sh:   sh,
		ps:   ps,
		gw:   gw,
		done: make(chan bool, 1),
	}

	for _, t := range tsks {
		eng.q.Push(t)
	}
	eng.q.Push(
		&task.TerminalTask{
			Done: eng.done,
		})
	return &eng
}

func (e *Engine) Start(ctx context.Context) chan error {
	errCh := make(chan error)

	go func() {
		defer close(errCh)
		tch := e.q.Subscribe()
		for {
			select {
			case t := <-tch:
				c, cancel := context.WithTimeout(ctx, 10*time.Minute)
				defer cancel()
				if err := t.Run(c, e.sh, e.ps, e.gw); err != nil {
					errCh <- err
				}
			case <-e.done:
				return
			}
		}
	}()

	return errCh
}

func (e *Engine) Stop() {
	log.Info("graceful shutdown")
	e.done <- true
}
