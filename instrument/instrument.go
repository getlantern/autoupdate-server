package instrument

import (
	"context"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	log = golog.LoggerFor("autoupdate-server.instrument")
)

type Instrument interface {
	ReportToOTELPeriodically(ctx context.Context, reportingInterval time.Duration)
	Stop()
	UpdateStats(clientKey ClientDetails, status string)
}

type ClientDetails struct {
	// version of the application updating itself
	AppVersion string
	// operating system of target platform
	OS string
	// hardware architecture of target platform
	Arch string `json:"-"`
	// country of the user running the application updating itself
	Country string
	// locale of the user running the application updating itself
	Locale string

	RemoteAddr string
}

type usage struct {
	error   string
	success bool
	sent    int
	recv    int
}

type instrument struct {
	clientStats map[ClientDetails]*usage
	tp          trace.TracerProvider
	statsMu     sync.Mutex
	stop        func()
}

func New(tp trace.TracerProvider, stop func()) Instrument {
	return &instrument{
		clientStats: make(map[ClientDetails]*usage),
		tp:          tp,
		stop:        stop,
	}
}

// UpdateStats records the client details and result of downloading an update
func (i *instrument) UpdateStats(clientKey ClientDetails, status string) {
	i.statsMu.Lock()
	defer i.statsMu.Unlock()
	u := &usage{}
	u.success = status == "200 OK"
	if !u.success {
		u.error = status
	}
	i.clientStats[clientKey] = u
}

// ReportToOTELPeriodically periodically reports to OpenTelemetry clientStats that represent
// when new releases of Lantern are downloaded via the auto-update server
func (i *instrument) ReportToOTELPeriodically(ctx context.Context, interval time.Duration) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug("Stopping periodic loading and processing of pending orders")
				return
			case <-time.After(interval):
				i.ReportToOTEL()
			}
		}
	}()
}

func (i *instrument) ReportToOTEL() {
	i.statsMu.Lock()
	clientStats := i.clientStats
	i.statsMu.Unlock()
	for key, value := range clientStats {
		_, span := i.tp.Tracer("").
			Start(
				context.Background(),
				"autoupdate_download",
				trace.WithAttributes(
					attribute.Int("bytes_sent", value.sent),
					attribute.Int("bytes_recv", value.recv),
					attribute.Int("bytes_total", value.sent+value.recv),
					attribute.Bool("success", value.success),
					attribute.String("error", value.error),
					attribute.String("client_ip", key.RemoteAddr),
					attribute.String("client_platform", key.OS),
					attribute.String("client_arch", key.Arch),
					attribute.String("client_version", key.AppVersion),
					attribute.String("client_locale", key.Locale),
					attribute.String("client_country", key.Country)))
		span.End()
	}
}

func (i *instrument) Stop() {
	if i.stop != nil {
		i.stop()
	}
}
