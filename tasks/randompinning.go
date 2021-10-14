package tasks

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ipfs/go-cid"
	shell "github.com/ipfs/go-ipfs-api"
	pinning "github.com/ipfs/go-pinning-service-http-client"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type RandomPinningBench struct {
	reg        *task.Registration
	size       int
	start_time prometheus.Histogram
	fetch_time prometheus.Histogram
	fails      prometheus.Counter
	errors     prometheus.Counter
}

func NewRandomPinningBench(schedule string, size int) *RandomPinningBench {
	start_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "random_pinning",
			Name:      fmt.Sprintf("%d_latency", size),
		})
	fetch_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "random_pinning",
			Name:      fmt.Sprintf("%d_fetch_time", size),
		})
	fails := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "random_pinning",
			Name:      "fail_count",
		})
	errors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "random_pinning",
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
	return &RandomPinningBench{
		reg:        &reg,
		size:       size,
		start_time: start_time,
		fetch_time: fetch_time,
		fails:      fails,
		errors:     errors,
	}
}

func (t *RandomPinningBench) Run(ctx context.Context, sh *shell.Shell, ps *pinning.Client, gw string) error {
	// generate random data
	log.Infof("generating %d bytes random data", t.size)
	randb := make([]byte, t.size)
	if _, err := rand.Read(randb); err != nil {
		log.Errorw("failed to generate random values", "err", err)
		t.errors.Inc()
		return err
	}
	buf := bytes.NewReader(randb)

	// add to local ipfs
	log.Info("writing data to local IPFS node")
	cidstr, err := sh.Add(buf)
	if err != nil {
		log.Errorw("failed to write to IPFS", "err", err)
		t.errors.Inc()
		return err
	}
	defer func() {
		log.Info("cleaning up IPFS node")
		// don't bother error checking. We clean it up explicitly in the happy path.
		sh.Unpin(cidstr)
	}()

	// Pin to pinning service
	c, err := cid.Decode(cidstr)
	if err != nil {
		log.Errorw("failed to decode cid after it was returned from IPFS", "cid", cidstr, "err", err)
		t.errors.Inc()
		return err
	}
	getter, err := ps.Add(ctx, c)
	if err != nil {
		log.Errorw("failed to pin cid to pinning service", "cid", cidstr, "err", err)
		t.errors.Inc()
		return err
	}

	// long poll pinning service
	log.Info("waiting for pinning service to complete the pin")
	var pinned bool
	for !pinned {
		status, err := ps.GetStatusByID(ctx, getter.GetRequestId())
		if err == nil {
			fmt.Println(status.GetStatus())
			pinned = status.GetStatus() == pinning.StatusPinned
		} else {
			fmt.Println(err)
		}
		time.Sleep(time.Minute)
	}

	// delete this from our local IPFS node.
	log.Info("removing pin from local IPFS node")
	err = sh.Unpin(cidstr)
	if err != nil {
		log.Errorw("could not unpin cid after adding it earlier")
		t.errors.Inc()
		return err
	}

	// request from gateway, observing client metrics
	url := fmt.Sprintf("%s/ipfs/%s", gw, cidstr)
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
	respb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("failed to downlaod content", "err", err)
		t.errors.Inc()
		return err
	}
	total_time := time.Since(start).Milliseconds()
	log.Infow("finished download", "ms", total_time)
	t.fetch_time.Observe(float64(total_time))

	log.Info("checking result")
	// compare response with what we sent
	if !reflect.DeepEqual(respb, randb) {
		log.Warnw("response from gateway did not match", "url", url)
		t.fails.Inc()
		return fmt.Errorf("expected response from gateway to match generated cid")
	}

	return nil
}

func (t *RandomPinningBench) Registration() *task.Registration {
	return t.reg
}
