package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	shell "github.com/ipfs/go-ipfs-api"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type NoopTask struct {
	i int
	g prometheus.Gauge
}

func (t *NoopTask) Run(ctx context.Context, sh *shell.Shell, gw string) error {
	for i := 0; i < t.i; i++ {
		time.Sleep(time.Second)
		fmt.Println("test")
		t.g.Add(1)
	}
	return nil
}

func (t *NoopTask) Registration() *task.Registration {
	return &task.Registration{
		Collectors: []prometheus.Collector{t.g},
		Schedule:   "0 * * * *",
	}
}

func NewNoopTask(i int) *NoopTask {
	return &NoopTask{
		i: i,
		g: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "gatewaymonitor",
				Subsystem: "noop",
				Name:      "noopgauge",
			},
		),
	}
}
