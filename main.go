package main

import (
	"net/http"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/jhorwit2/docker_collector/aggregator"
	"github.com/jhorwit2/docker_collector/collector"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {

	aggr, err := aggregator.New(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("aggregator error")
	}

	collector, _ := collector.New(aggr)
	prometheus.Register(collector)
	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(":8080", nil)

	// for {
	// 	select {
	// 	case <-time.Tick(5 * time.Second):
	//
	// 	case err := <-aggr.Err():
	// 		cancel()
	// 		logrus.Fatal(err)
	// 	case <-ctx.Done():
	// 		return
	// 	}
	// }

}
