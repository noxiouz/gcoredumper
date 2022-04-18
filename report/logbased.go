package report

import (
	"log"
)

type LogBasedReporter struct {
	*log.Logger
}

func (l *LogBasedReporter) Log(record *Record) {
	key := record.Name
	switch value := record.Value.(type) {
	case *Record_Str:
		log.Printf("%s = %s", key, value)
	case *Record_Number:
		log.Printf("%s = %d", key, value)
	case *Record_Duration:
		log.Printf("%s = %v", key, value)
	}
}
