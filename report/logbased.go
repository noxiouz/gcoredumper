package report

import (
	"log"
	"time"
)

type LogBasedReporter struct {
	*log.Logger
}

func (l *LogBasedReporter) ReportString(key string, value string) error {
	log.Printf("%s = %s", key, value)
	return nil
}

func (l *LogBasedReporter) ReportInt(key string, value int64) error {
	log.Printf("%s = %d", key, value)
	return nil
}

func (l *LogBasedReporter) ReportError(key string, err error) error {
	log.Printf("%s = %v", key, err)
	return nil
}

func (l *LogBasedReporter) ReportDuration(key string, d time.Duration) error {
	log.Printf("%s = %v", key, d)
	return nil
}
