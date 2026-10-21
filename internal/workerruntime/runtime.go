// Package workerruntime composes fixed private acquisition routes without enabling live accounts.
package workerruntime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/syncdispatch"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerhost"
)

// Fixed runtime errors deliberately discard infrastructure diagnostics and configuration values.
var (
	ErrConfiguration = errors.New("worker_configuration_rejected")
	ErrUnavailable   = errors.New("worker_dependency_unavailable")
)
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Config contains identifiers only; mode chooses an exact private route and database capability.
type Config struct {
	Mode, Instance, Listen, BindingID, PublisherPolicy string
	Caller                                             observer.CallerPolicy
	Queue                                              observer.QueueConfig
}

// component separates publisher delivery from the sync login that can acquire source observations.
func (c Config) component() string {
	if c.Mode == "publisher" {
		return "publisher"
	}
	return "sync"
}

// Validate rejects ambiguous mode, sockets and cross-project queue configuration before external access.
func (c Config) Validate() error {
	if c.Mode != "publisher" && c.Mode != "consumer" {
		return ErrConfiguration
	}
	if (clouddb.Config{Instance: c.Instance, Component: c.component()}).Validate() != nil {
		return ErrConfiguration
	}
	host, port, e := net.SplitHostPort(c.Listen)
	n, pe := strconv.Atoi(port)
	if e != nil || pe != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port || (host != "127.0.0.1" && host != "0.0.0.0") {
		return ErrConfiguration
	}
	if c.Mode == "publisher" {
		fixed := (c.PublisherPolicy == "" || c.PublisherPolicy == "binding-v1") && uuid.MatchString(c.BindingID) && c.BindingID != "00000000-0000-0000-0000-000000000000"
		shared := c.PublisherPolicy == "shared-v1" && c.BindingID == ""
		if (!fixed && !shared) || c.Queue.Project != (clouddb.Config{Instance: c.Instance}).Project() {
			return ErrConfiguration
		}
	} else if c.BindingID != "" || c.PublisherPolicy != "" || c.Queue != (observer.QueueConfig{}) {
		return ErrConfiguration
	}
	return nil
}

// Dependencies are injected capabilities, never raw credentials or caller-controlled provider routes.
type Dependencies struct {
	DB        acquisition.Beginner
	Verifier  observer.TokenVerifier
	Transport observer.Transport
	Runner    syncdispatch.Runner
	Close     func()
}

// Runtime owns one fixed-route worker and an idempotent dependency cleanup hook.
type Runtime struct {
	host    *workerhost.Host
	listen  string
	close   func()
	once    sync.Once
	started atomic.Bool
}

// New validates and composes the existing authenticated handlers without opening a listener.
func New(c Config, d Dependencies) (*Runtime, error) {
	if c.Validate() != nil || d.DB == nil || d.Verifier == nil {
		return nil, ErrConfiguration
	}
	var h http.Handler
	var e error
	if c.Mode == "publisher" {
		if d.Runner != nil || d.Transport == nil {
			return nil, ErrConfiguration
		}
		var p *syncdispatch.Publisher
		var err error
		if c.PublisherPolicy == "shared-v1" {
			p, err = syncdispatch.NewSharedPublisher(d.DB, d.Transport)
		} else {
			p, err = syncdispatch.NewPublisher(d.DB, c.BindingID, d.Transport)
		}
		if err != nil {
			return nil, ErrConfiguration
		}
		h, e = observer.NewAcquisitionPublisherHTTP(p, d.Verifier, c.Caller)
	} else {
		if d.Transport != nil || d.Runner == nil {
			return nil, ErrConfiguration
		}
		worker, err := syncdispatch.NewConsumer(d.DB, d.Runner)
		if err != nil {
			return nil, ErrConfiguration
		}
		h, e = observer.NewAcquisitionConsumerHTTP(worker, d.Verifier, c.Caller)
	}
	if e != nil {
		return nil, ErrConfiguration
	}
	host, e := workerhost.New(h)
	if e != nil {
		return nil, ErrConfiguration
	}
	return &Runtime{host: host, listen: c.Listen, close: d.Close}, nil
}

// Close releases owned database and transport resources at most once.
func (r *Runtime) Close() {
	if r != nil {
		r.once.Do(func() {
			if r.close != nil {
				r.close()
			}
		})
	}
}

// Serve accepts an explicit listener for synthetic lifecycle proof and closes dependencies after draining.
func (r *Runtime) Serve(ctx context.Context, l net.Listener) error {
	if r == nil || r.host == nil || ctx == nil || l == nil {
		return ErrConfiguration
	}
	if !r.started.CompareAndSwap(false, true) {
		return workerhost.ErrAlreadyServed
	}
	defer r.Close()
	return r.host.Serve(ctx, l)
}

// Run opens only the previously validated TCP bind; requests still require pinned workload identity.
func (r *Runtime) Run(ctx context.Context) error {
	if r == nil || r.host == nil || ctx == nil {
		return ErrConfiguration
	}
	if !r.started.CompareAndSwap(false, true) {
		return workerhost.ErrAlreadyServed
	}
	defer r.Close()
	if ctx.Err() != nil {
		return ErrUnavailable
	}
	l, e := net.Listen("tcp", r.listen)
	if e != nil {
		return ErrUnavailable
	}
	return r.host.Serve(ctx, l)
}
