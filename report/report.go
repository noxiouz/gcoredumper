package report

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
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
	mu      sync.Mutex
	records []*Record
}

func New() *Report {
	return &Report{}
}

// AddInt adds int64 value to report
func (r *Report) AddInt(key string, v int64) {
	r.add(&Record{
		Name:  key,
		Value: &Record_Number{v},
	})
}

// AddString adds string value to report
func (r *Report) AddString(key string, v string) {
	r.add(&Record{
		Name:  key,
		Value: &Record_Str{v},
	})
}

func (r *Report) AddError(key string, err error) {
	var msg string
	if err != nil {
		msg = err.Error()
	} else {
		msg = "<nil>"
	}
	r.add(&Record{
		Name:  key,
		Value: &Record_Str{msg},
	})
}

func (r *Report) AddDuration(key string, value time.Duration) {
	r.add(&Record{
		Name:  key,
		Value: &Record_Duration{durationpb.New(value)},
	})
}

func (r *Report) add(record *Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}

func (r *Report) Report(s Sink) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, record := range r.records {
		s.Log(record)
	}
}

type Sink interface {
	Log(record *Record)
}
