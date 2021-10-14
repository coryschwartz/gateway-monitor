package tasks

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ipfs/go-cid"
	shell "github.com/ipfs/go-ipfs-api"
	pinning "github.com/ipfs/go-pinning-service-http-client"
	"github.com/multiformats/go-multihash"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type NonExistCheck struct {
	reg        *task.Registration
	start_time prometheus.Histogram
	fetch_time prometheus.Histogram
	fails      prometheus.Counter
	errors     prometheus.Counter
}

func NewNonExistCheck(schedule string) *NonExistCheck {
	start_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "nonexsist",
			Name:      "latency",
		})
	fetch_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "non_exist",
			Name:      "fetch_time",
		})
	fails := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "non_exist",
			Name:      "fail_count",
		})
	errors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "non_exist",
			Name:      "error_count",
		})
	reg := task.Registration{
		Schedule: schedule,
		Collectors: []prometheus.Collector{
			start_time,
			fetch_time,
			fails,
			errors,
		},
	}
	return &NonExistCheck{
		reg:        &reg,
		start_time: start_time,
		fetch_time: fetch_time,
		fails:      fails,
		errors:     errors,
	}
}

func (t *NonExistCheck) Run(ctx context.Context, sh *shell.Shell, ps *pinning.Client, gw string) error {

	buf := make([]byte, 128)
	_, err := rand.Read(buf)
	if err != nil {
		log.Error("failed to generate random bytes")
		t.errors.Inc()
		return err
	}

	encoded, err := multihash.EncodeName(buf, "sha3")
	if err != nil {
		log.Error("failed to generate multihash of random bytes")
		t.errors.Inc()
		return err
	}
	cast, err := multihash.Cast(encoded)
	if err != nil {
		log.Error("failed to cast as multihash")
		return err
	}

	c := cid.NewCidV1(cid.Raw, cast)
	log.Info("generated random CID", "cid", c.String())

	url := fmt.Sprintf("%s/ipfs/%s", gw, c.String())

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
		t.errors.Inc()
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("failed to download content", "err", err)
		t.errors.Inc()
		return err
	}
	total_time := time.Since(start).Milliseconds()
	log.Infow("finished download", "ms", total_time)
	t.fetch_time.Observe(float64(total_time))

	log.Info("checking that we got a 404")
	if resp.StatusCode != 404 {
		log.Warnw("expected to see 404 from gateway, but didn't.", "status", resp.StatusCode)
		t.fails.Inc()
		return fmt.Errorf("expected 404")
	}

	return nil
}

func (t *NonExistCheck) Registration() *task.Registration {
	return t.reg
}
