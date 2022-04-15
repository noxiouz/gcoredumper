package report

import (
	"context"
	"sync"
	"time"
)

var (
	reportKey     struct{}
	defaultReport = New()

	R = GetReport
)

// WithReport attaches report to Context.
func WithReport(ctx context.Context, r *Report) context.Context {
	return context.WithValue(ctx, reportKey, r)
}

// GetReport returns a Report attached to Context. Or returns a global one.
func GetReport(ctx context.Context) *Report {
	if v := ctx.Value(reportKey); v != nil {
		return v.(*Report)
	}
	return defaultReport
}

// Report provides an interface to collect specific types of key-value pairs.
type Report struct {
	mu sync.Mutex
	m  map[string]any
}

func New() *Report {
	return &Report{
		m: make(map[string]any),
	}
}

// AddInt adds int64 value to report
func (r *Report) AddInt(key string, v int64) {
	r.add(key, v)
}

// AddString adds string value to report
func (r *Report) AddString(key string, v string) {
	r.add(key, v)
}

func (r *Report) AddError(key string, err error) {
	r.add(key, err)
}

func (r *Report) AddDuration(key string, value time.Duration) {
	r.add(key, value)
}

func (r *Report) add(key string, v any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[key] = v
}

func (r *Report) Report(sink Sink) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, value := range r.m {
		var err error
		switch value := value.(type) {
		case int64:
			err = sink.ReportInt(key, value)
		case string:
			err = sink.ReportString(key, value)
		case error:
			err = sink.ReportError(key, value)
		case time.Duration:
			err = sink.ReportDuration(key, value)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

type Sink interface {
	ReportError(key string, value error) error
	ReportInt(key string, value int64) error
	ReportString(key string, value string) error
	ReportDuration(key string, value time.Duration) error
}
