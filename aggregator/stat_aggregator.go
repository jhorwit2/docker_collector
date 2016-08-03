package aggregator

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"golang.org/x/net/context"
)

type ContainerInfo struct {
	ContainerID string
	TaskID      string
	ServiceID   string
}

type StatAggregator struct {
	cli    *client.Client
	err    chan error
	closed chan struct{}

	mu    sync.Mutex
	stats map[ContainerInfo]types.StatsJSON
}

func New(ctx context.Context) (*StatAggregator, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	s := &StatAggregator{
		cli:   cli,
		err:   make(chan error, 1),
		stats: make(map[ContainerInfo]types.StatsJSON),
	}
	go s.run(ctx)

	return s, nil
}

func (s *StatAggregator) Err() <-chan error {
	return s.err
}

func (s *StatAggregator) Stats() map[ContainerInfo]types.StatsJSON {
	copy := make(map[ContainerInfo]types.StatsJSON)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, s := range s.stats {
		copy[i] = s
	}
	return copy
}

func (s *StatAggregator) startStatStream(ctx context.Context, req ContainerInfo) {
	reader, err := s.cli.ContainerStats(ctx, req.ContainerID, true)
	if err != nil {
		s.err <- err
		return
	}

	dec := json.NewDecoder(reader)

	for {
		var stats types.StatsJSON
		err := dec.Decode(&stats)
		if err != nil {
			s.err <- err
			return
		}

		s.mu.Lock()
		s.stats[req] = stats
		s.mu.Unlock()
	}
}

func (s *StatAggregator) run(ctx context.Context) {
	filter := filters.NewArgs()
	filter.Add("type", "container")
	events, errs := s.cli.Events(ctx, types.EventsOptions{Filters: filter})

	streams := make(map[string]context.CancelFunc)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errs:
			if err != io.EOF {
				// Send the error and for the time being restart the stream.
				// The caller can then cancel the new stream if the error
				// isn't something they care about.
				s.err <- err
			}

			events, errs = s.cli.Events(ctx, types.EventsOptions{Filters: filter})

		case evt := <-events:
			switch evt.Action {
			case "create":
				streamCTX, cancel := context.WithCancel(ctx)

				streams[evt.ID] = cancel

				attributes := evt.Actor.Attributes
				req := ContainerInfo{
					ContainerID: evt.ID,
					ServiceID:   attributes["com.docker.swarm.service.id"],
					TaskID:      attributes["com.docker.swarm.task.id"],
				}

				logrus.WithFields(logrus.Fields{
					"container_id": req.ContainerID,
					"sevice_id":    req.ServiceID,
					"task_id":      req.TaskID,
				}).Infof("create received")

				go s.startStatStream(streamCTX, req)

			case "die":

				logrus.WithFields(logrus.Fields{
					"container_id": evt.ID,
				}).Infof("die received")

				cancelFunc, ok := streams[evt.ID]
				if ok {
					cancelFunc()
				}
			}
		}
	}
}
