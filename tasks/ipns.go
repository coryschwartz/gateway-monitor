package tasks

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
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

type IpnsBench struct {
	reg          *task.Registration
	size         int
	publish_time prometheus.Histogram
	start_time   prometheus.Histogram
	fetch_time   prometheus.Histogram
}

func NewIpnsBench(schedule string, size int) *IpnsBench {
	publish_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "ipns",
			Name:      "publish",
		})
	start_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "ipns",
			Name:      fmt.Sprintf("%d_latency", size),
		})
	fetch_time := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gatewaymonitor",
			Subsystem: "ipns",
			Name:      fmt.Sprintf("%d_fetch_time", size),
		})
	reg := task.Registration{
		Schedule: schedule,
		Collectors: []prometheus.Collector{
			publish_time,
			start_time,
			fetch_time,
		},
	}
	return &IpnsBench{
		reg:          &reg,
		size:         size,
		publish_time: publish_time,
		start_time:   start_time,
		fetch_time:   fetch_time,
	}
}

func (t *IpnsBench) Run(ctx context.Context, sh *shell.Shell, ps *pinning.Client, gw string) error {

	// generate random data
	log.Infof("generating %d bytes random data", t.size)
	randb := make([]byte, t.size)
	if _, err := rand.Read(randb); err != nil {
		log.Errorw("failed to generate random values", "err", err)
		return err
	}
	buf := bytes.NewReader(randb)

	// add to local ipfs
	log.Info("writing data to local IPFS node")
	cidstr, err := sh.Add(buf)
	if err != nil {
		log.Errorw("failed to write to IPFS", "err", err)
		return err
	}
	defer func() {
		log.Info("cleaning up IPFS node")
		err := sh.Unpin(cidstr)
		if err != nil {
			log.Warnw("failed to clean unpin cid.", "cid", cidstr)
		}
	}()

	// Generate a new key
	// we already have a random value lying around, might as
	// well use it for the ney name.
	keyName := base64.StdEncoding.EncodeToString(randb[:8])
	_, err = sh.KeyGen(ctx, keyName)
	if err != nil {
		log.Errorw("failed to generate new key", "err", err)
		return err
	}
	defer func() {
		sh.KeyRm(ctx, keyName)
	}()

	// Publish IPNS
	pub_start := time.Now()
	pubResp, err := sh.PublishWithDetails(cidstr, keyName, time.Hour, time.Hour, true)
	publish_time := time.Since(pub_start).Milliseconds()
	log.Infow("published IPNS", "ms", publish_time, "cid", cidstr, "ipns", pubResp.Name)
	t.publish_time.Observe(float64(publish_time))

	// request from gateway, observing client metrics
	url := fmt.Sprintf("%s/ipns/%s", gw, pubResp.Name)
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
	if !reflect.DeepEqual(respb, randb) {
		log.Warnw("response from gateway did not match", "url", url)
		return fmt.Errorf("expected response from gateway to match generated cid")
	}

	return nil
}

func (t *IpnsBench) Registration() *task.Registration {
	return t.reg
}
