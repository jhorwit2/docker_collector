package events

import (
	goevents "github.com/docker/go-events"
	"golang.org/x/net/context"
)

// Watcher watches an event stream from the daemon providing
// an easy way to filter events to specific channels
type Watcher interface {
	// Watch all the events from the daemon.
	//
	// If specified, matcher functions are ORed together stopping
	// at the first match. If you want to do a boolean AND match then
	// you should implement your own MatcherFunc with multiple checks.
	//
	// Cancel the context to quit watching the events on this channel
	Watch(ctx context.Context, matcher ...MatcherFunc) <-chan Message
}

type watcher struct {
	broadcast *goevents.Broadcaster
	events    <-chan Message
	buffer    int
	closed    chan struct{}
}

func (w *watcher) Watch(ctx context.Context, matchers ...MatcherFunc) <-chan Message {
	return w.createSinkWrapper(ctx, func(event goevents.Event) bool {
		if len(matchers) == 0 {
			return true
		}

		msg, ok := event.(Message)
		if !ok {
			return false
		}

		for _, matcher := range matchers {
			if matcher.Matches(msg) {
				return true
			}
		}

		return false
	})
}

func (w *watcher) createSinkWrapper(ctx context.Context, matcher goevents.MatcherFunc) <-chan Message {
	eventq := make(chan Message, w.buffer)
	ch := goevents.NewChannel(w.buffer)
	sink := goevents.Sink(goevents.NewQueue(ch))

	if matcher != nil {
		sink = goevents.NewFilter(sink, matcher)
	}

	cleanup := func() {
		close(eventq)
		w.broadcast.Remove(sink)
		sink.Close()
	}

	// magic that lets us receive each event that passes the matcher on the ch sink.
	w.broadcast.Add(sink)

	go func() {
		defer cleanup()

		for {
			select {
			case <-w.closed:
				// stream of events being broadcast was closed
				return
			case <-ctx.Done():
				// caller is done watching this specific stream
				return
			case e := <-ch.C:
				// event received from broadcast
				select {
				case <-w.closed:
					// stream of events being broadcast was closed
					return
				case <-ctx.Done():
					// caller is done watching this specific stream
					return
				case eventq <- e.(Message):
				}
			}
		}
	}()

	return eventq
}

func (w *watcher) startWatching(ctx context.Context) {
	defer close(w.closed)
	defer w.broadcast.Close()

	for {
		select {
		case <-ctx.Done():
			// caller is watching.
			return
		case e, ok := <-w.events:
			if !ok {
				// events channel got closed by caller
				return
			}

			select {
			case <-ctx.Done():
				// caller is watching.
				return
			default:
				// send the event out so all appropriate sinks get it.
				w.broadcast.Write(e)
			}
		}
	}
}

// NewWatcher returns a new event watcher for the given channel.
// buffer should be specified to allow every channel returned by watch
// to buffer events.
//
// It's up to the caller to stop the watcher by canceling the context. By
// canceling the context all the channels returned by Watch will be closed.
func NewWatcher(ctx context.Context, events <-chan Message, buffer int) Watcher {
	w := &watcher{
		buffer:    buffer,
		broadcast: goevents.NewBroadcaster(),
		events:    events,
		closed:    make(chan struct{}),
	}
	go w.startWatching(ctx)
	return w
}
