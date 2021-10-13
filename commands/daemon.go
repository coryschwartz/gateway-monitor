package commands

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	"github.com/coryschwartz/gateway-monitor/pkg/engine"
	"github.com/coryschwartz/gateway-monitor/tasks"
)

var daemonCommand = &cli.Command{
	Name:  "daemon",
	Usage: "run commands on schedule",
	Action: func(cctx *cli.Context) error {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(":2112", nil)
		}()
		ipfs := GetIPFS(cctx)
		ps := GetPinningService(cctx)
		gw := GetGW(cctx)
		eng := engine.New(ipfs, ps, gw, tasks.All...)
		return <-eng.Start(cctx.Context)
	},
}
