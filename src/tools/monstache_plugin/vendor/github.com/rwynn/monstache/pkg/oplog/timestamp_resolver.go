package oplog

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"sync"
	"time"
)

// A TimestampResolver decides on a timestamp from which to start reading an oplog from.
// A result may not be immediately available (see TimestampResolverEarliest), so it is returned in a channel.
type TimestampResolver interface {
	GetResumeTimestamp(candidateTs primitive.Timestamp, source string) chan primitive.Timestamp
}

// An oplog resume timestamp saved by a monstache instance
const TS_SOURCE_MONSTACHE = "monstache"

// An oplog resume timestamp taken from the last mongodb operation
const TS_SOURCE_OPLOG = "oplog"

// A simple resolver immediately returns a timestamp it's been given.
type TimestampResolverSimple struct{}

func (r TimestampResolverSimple) GetResumeTimestamp(candidateTs primitive.Timestamp, source string) chan primitive.Timestamp {
	tmpResultChan := make(chan primitive.Timestamp, 1)
	tmpResultChan <- candidateTs

	return tmpResultChan
}

// TimestampResolverEarliest waits until oplog resume timestamps have been queried from all the available mongodb shards, and returns an earliest one.
type TimestampResolverEarliest struct {
	connectionsTotal   int
	connectionsQueried int
	earliestTs         primitive.Timestamp
	earliestTsSource   string
	resultChan         chan primitive.Timestamp
	logger             *log.Logger
	m                  sync.Mutex
}

func NewTimestampResolverEarliest(connectionsTotal int, logger *log.Logger) *TimestampResolverEarliest {
	return &TimestampResolverEarliest{
		connectionsTotal: connectionsTotal,
		resultChan:       make(chan primitive.Timestamp, connectionsTotal),
		logger:           logger,
	}
}

// Returns a channel from which an earliest resume timestamp can be received
func (resolver *TimestampResolverEarliest) GetResumeTimestamp(candidateTs primitive.Timestamp, source string) chan primitive.Timestamp {
	resolver.m.Lock()
	defer resolver.m.Unlock()

	if resolver.connectionsQueried >= resolver.connectionsTotal {
		// in this case, an earliest timestamp is already calculated,
		// so it is just returned in a temporary channel
		resolver.logger.Printf(
			"Earliest oplog resume timestamp is already calculated: %s",
			tsToString(resolver.earliestTs),
		)
		tmpResultChan := make(chan primitive.Timestamp, 1)
		tmpResultChan <- resolver.earliestTs
		return tmpResultChan
	}

	resolver.connectionsQueried++
	resolver.updateEarliestTs(source, candidateTs)

	// if this function has been called for every mongodb connection,
	// then a final earliest resume timestamp can be returned to every caller
	if resolver.connectionsQueried == resolver.connectionsTotal {
		resolver.logger.Printf(
			"Earliest oplog resume timestamp calculated: %s, source: %s",
			tsToString(resolver.earliestTs),
			resolver.earliestTsSource,
		)

		for i := 0; i < resolver.connectionsTotal; i++ {
			resolver.resultChan <- resolver.earliestTs
		}
	}

	return resolver.resultChan
}

// Updates a timestamp to resume syncing from
func (resolver *TimestampResolverEarliest) updateEarliestTs(source string, candidateTs primitive.Timestamp) {
	// a timestamp from oplog has a lower priority
	if resolver.earliestTsSource == TS_SOURCE_MONSTACHE && source == TS_SOURCE_OPLOG {
		return
	}

	// a timestamp from monstache has a higher priority,
	// and among timestamps from the same source, the earlier is preferred
	if resolver.earliestTs.T == 0 ||
		(resolver.earliestTsSource == TS_SOURCE_OPLOG && source == TS_SOURCE_MONSTACHE) ||
		(primitive.CompareTimestamp(candidateTs, resolver.earliestTs) < 0) {
		resolver.logger.Printf(
			"Candidate resume timestamp: %s, source: %s",
			tsToString(candidateTs),
			source,
		)
		resolver.earliestTs = candidateTs
		resolver.earliestTsSource = source
	}
}

// Converts a bson timestamp to string
func tsToString(ts primitive.Timestamp) string {
	return fmt.Sprintf(
		"%s, I=%d",
		time.Unix(int64(ts.T), 0).Format(time.RFC3339),
		ts.I,
	)
}
