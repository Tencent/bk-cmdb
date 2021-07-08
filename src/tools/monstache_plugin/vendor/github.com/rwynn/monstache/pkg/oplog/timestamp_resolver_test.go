package oplog

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"os"
	"testing"
)

// Tests an earliest timestamp resolver with 3 mongodb shards
func TestTimestampResolverEarliest_GetResumeTimestamp_ThreeShards(t *testing.T) {
	resolver := NewTimestampResolverEarliest(3, log.New(os.Stdout, "INFO ", log.Flags()))

	chanB := resolver.GetResumeTimestamp(
		// this value is not an expected result,
		// because values with source=monstache have a higher priority
		primitive.Timestamp{
			T: 3,
			I: 1,
		},
		TS_SOURCE_OPLOG,
	)
	chanC := resolver.GetResumeTimestamp(
		// this value is not an expected result,
		// because it's larger than the next one
		primitive.Timestamp{
			T: 10000,
			I: 10050,
		},
		TS_SOURCE_MONSTACHE,
	)
	chanA := resolver.GetResumeTimestamp(
		// this is  an expected result
		primitive.Timestamp{
			T: 10,
			I: 15,
		},
		TS_SOURCE_MONSTACHE,
	)

	resultA := <-chanA
	resultB := <-chanB
	resultC := <-chanC

	if resultA.T != 10 || resultA.I != 15 {
		t.Fatalf(
			"Expected an earliest timestamp to be 10.15, got %d.%d",
			resultA.T,
			resultA.I,
		)
	}

	if !resultB.Equal(resultA) || !resultC.Equal(resultA) {
		t.Fatalf(
			"An earliest timestamp must be consistent for all callers",
		)
	}

	repeatedCallResult := <-resolver.GetResumeTimestamp(primitive.Timestamp{
		T: 1,
		I: 1,
	}, TS_SOURCE_OPLOG)
	if !repeatedCallResult.Equal(resultA) {
		t.Fatalf(
			"A repeated call to GetResumeTimestamp must return a previously calculated timestamp; got %d.%d.",
			repeatedCallResult.T,
			repeatedCallResult.I,
		)
	}
}

// Tests an earliest timestamp resolver with a single mongodb shard
func TestTimestampResolverEarliest_GetResumeTimestamp_SingleShard(t *testing.T) {
	resolver := NewTimestampResolverEarliest(1, log.New(os.Stdout, "INFO ", log.Flags()))

	result := <-resolver.GetResumeTimestamp(primitive.Timestamp{
		T: 1000,
		I: 3,
	}, TS_SOURCE_OPLOG)

	if result.T != 1000 || result.I != 3 {
		t.Fatalf(
			"Expected an earliest timestamp to be 1000.3, got %d.%d",
			result.T,
			result.I,
		)
	}
}
