package tests

import (
	"fmt"
	"net"
	"testing"

	"github.com/dicedb/dice/core"
	"github.com/dicedb/dice/internal/constants"
	"gotest.tools/v3/assert"
)

// need to test the following here:
// qwatch should work as expected (which is covered in qwatch cases)
// after a subscribe, and a few updates, unwatch should remove the subscriber from the watch list

func TestQWatchUnwatch(t *testing.T) {
	publisher := getLocalConnection()
	subscribers := []net.Conn{getLocalConnection(), getLocalConnection(), getLocalConnection()}

	// Cleanup Store to remove any existing keys from other qwatch tests
	// The cleanup is done both on the start and finish just to keep the order of tests run agnostic
	for _, tc := range qWatchTestCases {
		fireCommand(publisher, fmt.Sprintf("DEL match:100:user:%d", tc.userID))
	}
	defer func() {
		publisher.Close()
		for _, sub := range subscribers {
			sub.Close()
		}
	}()

	// Subscribe to the watch query
	respParsers := make([]*core.RESPParser, len(subscribers))

	for i, sub := range subscribers {
		rp := fireCommandAndGetRESPParser(sub, "QWATCH \""+qWatchQuery+"\"")
		respParsers[i] = rp

		// Check if the response is OK
		resp, err := rp.DecodeOne()
		assert.NilError(t, err)
		assert.Equal(t, 3, len(resp.([]interface{})))
	}

	// Make updates to the store
	runQWatchScenarios(t, publisher, respParsers)

	// Unwatch the query on two of the subscribers
	for _, sub := range subscribers[0:2] {
		rp := fireCommandAndGetRESPParser(sub, "QUNWATCH \""+qWatchQuery+"\"")
		resp, err := rp.DecodeOne()
		assert.NilError(t, err)
		assert.Equal(t, "OK", resp)
	}

	// qwatch scenarios on the third subscriber should continue to run as expected
	// AND
	// continue from the qwatch scenarios that ran previously
	fireCommand(publisher, "SET match:100:user:1 62")
	resp, err := respParsers[2].DecodeOne()
	assert.NilError(t, err)
	expectedUpdate := []interface{}{[]interface{}{"match:100:user:5", int64(70)}, []interface{}{"match:100:user:1", int64(62)}, []interface{}{"match:100:user:0", int64(60)}}
	assert.DeepEqual(t, []interface{}{constants.Qwatch, qWatchQuery, expectedUpdate}, resp)

	fireCommand(publisher, "SET match:100:user:5 75")
	resp, err = respParsers[2].DecodeOne()
	assert.NilError(t, err)
	expectedUpdate = []interface{}{[]interface{}{"match:100:user:5", int64(75)}, []interface{}{"match:100:user:1", int64(62)}, []interface{}{"match:100:user:0", int64(60)}}
	assert.DeepEqual(t, []interface{}{constants.Qwatch, qWatchQuery, expectedUpdate}, resp)

	fireCommand(publisher, "SET match:100:user:0 80")
	resp, err = respParsers[2].DecodeOne()
	assert.NilError(t, err)
	expectedUpdate = []interface{}{[]interface{}{"match:100:user:0", int64(80)}, []interface{}{"match:100:user:5", int64(75)}, []interface{}{"match:100:user:1", int64(62)}}
	assert.DeepEqual(t, []interface{}{constants.Qwatch, qWatchQuery, expectedUpdate}, resp)

	// Cleanup Store for next tests
	for _, tc := range qWatchTestCases {
		fireCommand(publisher, fmt.Sprintf("DEL match:100:user:%d", tc.userID))
	}
}