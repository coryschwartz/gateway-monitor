package tasks

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	shell "github.com/ipfs/go-ipfs-api"
	pinning "github.com/ipfs/go-pinning-service-http-client"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type KnownGoodCheck struct {
	reg        *task.Registration
	checks     map[string][]byte
	start_time prometheus.Histogram
	fetch_time prometheus.Histogram
}

func NewKnownGoodCheck(schedule string, checks map[string][]byte) *KnownGoodCheck {
	start_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "known_good",
			Name:      "latency",
		})
	fetch_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "known_good",
			Name:      "fetch_time",
		})
	reg := task.Registration{
		Schedule: schedule,
		Collectors: []prometheus.Collector{
			start_time,
			fetch_time,
		},
	}
	return &KnownGoodCheck{
		reg:        &reg,
		checks:     checks,
		start_time: start_time,
		fetch_time: fetch_time,
	}
}

func (t *KnownGoodCheck) Run(ctx context.Context, sh *shell.Shell, ps *pinning.Client, gw string) error {

	for ipfspath, value := range t.checks {
		// request from gateway, observing client metrics
		url := fmt.Sprintf("%s%s", gw, ipfspath)
		log.Infow("fetching from gateway", "url", url)
		req, _ := http.NewRequest("GET", url, nil)
		start := time.Now()
		trace := &httptrace.ClientTrace{
			GotFirstResponseByte: func() {
				latency := time.Since(start).Milliseconds()
				log.Infow("first byte received", "ms", latency)
				t.start_time.Observe(float64(latency))
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(ctx, trace))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Errorw("failed to fetch from gateway", "err", err)
			return err
		}
		respb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		total_time := time.Since(start).Milliseconds()
		log.Infow("finished download", "ms", total_time)
		t.fetch_time.Observe(float64(total_time))

		log.Info("checking result")
		// compare response with what we sent
		if !reflect.DeepEqual(respb, value) {
			log.Warnw("response from gateway did not match", "url", url, "found", respb, "expected", value)
			return fmt.Errorf("expected response from gateway to match generated cid")
		}
	}

	return nil
}

func (t *KnownGoodCheck) Registration() *task.Registration {
	return t.reg
}
