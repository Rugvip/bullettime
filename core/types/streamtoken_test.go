// Copyright 2015  Ericsson AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"strconv"
	"testing"
)

func TestValueParsing(t *testing.T) {
	testParseValue(t, "A", 0)
	testParseValue(t, "B", 1)
	testParseValue(t, "C", 2)
	testParseValue(t, "AA", 0)
	testParseValue(t, "AB", 1)
	testParseValue(t, "BA", 64)
	testParseValue(t, "BB", 65)
	testParseValue(t, "BAA", 64*64)
	testParseValue(t, "BBA", 64*64+64)
	testParseValue(t, "BAB", 64*64+1)
	testParseValue(t, "BBB", 64*64+64+1)
	testParseValue(t, "BBBB", 64*64*64+64*64+64+1)
}

func testParseValue(t *testing.T, str string, expected uint64) {
	value, err := parseValue(str)
	if err != nil {
		t.Fatal("got parse error for \"" + str + "\" :" + err.Error())
	}
	if value != expected {
		t.Fatal("unexpected value: got " + strconv.Itoa(int(value)) +
			", but expected " + strconv.Itoa(int(expected)))
	}
}

func TestShardId(t *testing.T) {
	testShardId(t, 0, 0, 0)
	testShardId(t, 1, 1, 0)
	testShardId(t, 2, 2, 0)
	testShardId(t, 3, 2, 1)
	testShardId(t, 4, 4, 0)
	testShardId(t, 6, 4, 2)
	testShardId(t, 7, 4, 3)
	testShardId(t, 64, 64, 0)
	testShardId(t, 256, 256, 0)
	testShardId(t, 257, 256, 1)
}

func testShardId(t *testing.T, shardId ShardId, totalCount, index int) {
	if shardId.TotalCount() != totalCount {
		t.Fatal("unexpected shard id total count: got " + strconv.Itoa(int(shardId.TotalCount())) +
			", but expected " + strconv.Itoa(int(totalCount)))
	}
	if shardId.Index() != index {
		t.Fatal("unexpected shard id index: got " + strconv.Itoa(int(shardId.Index())) +
			", but expected " + strconv.Itoa(int(index)))
	}
}

func TestEventStreamMux(t *testing.T) {
	var err Error
	var token StreamToken

	token, err = ParseStreamToken("")
	if err == nil {
		t.Fatal("expected error")
	}
	token, err = ParseStreamToken("B.BA.BBA*")
	if err != nil {
		t.Fatal(err)
	}
	expectLengths(t, token, 1, 0)
	expectValues(t, token.RoomTuples[0], 1, 64, 64*64+64)

	token, err = ParseStreamToken("B.B.B*B.B.B_B.B.B")
	if err != nil {
		t.Fatal(err)
	}
	expectLengths(t, token, 1, 2)
	expectValues(t, token.RoomTuples[0], 1, 1, 1)
	expectValues(t, token.UserTuples[0], 1, 1, 1)
	expectValues(t, token.UserTuples[1], 1, 1, 1)

	token, err = ParseStreamToken("*B.B.B_B.B.B_B.B.B")
	if err != nil {
		t.Fatal(err)
	}
	expectLengths(t, token, 0, 3)
	expectValues(t, token.UserTuples[0], 1, 1, 1)
	expectValues(t, token.UserTuples[1], 1, 1, 1)
	expectValues(t, token.UserTuples[2], 1, 1, 1)
}

func expectLengths(t *testing.T, token StreamToken, roomLength, userLength int) {
	if len(token.RoomTuples) != roomLength {
		t.Fatal("unexpected room length: got " + strconv.Itoa(len(token.RoomTuples)) +
			", but expected " + strconv.Itoa(roomLength))
	}
	if len(token.UserTuples) != userLength {
		t.Fatal("unexpected user length: got " + strconv.Itoa(len(token.UserTuples)) +
			", but expected " + strconv.Itoa(userLength))
	}
}

func expectValues(t *testing.T, tuple Tuple, shardId, eventIndex, signalIndex uint64) {
	if tuple.ShardId != ShardId(shardId) {
		t.Fatal("unexpected shard id: got " + strconv.Itoa(int(tuple.ShardId)) +
			", but expected " + strconv.Itoa(int(shardId)))
	}
	if tuple.EventIndex != eventIndex {
		t.Fatal("unexpected event index: got " + strconv.Itoa(int(tuple.EventIndex)) +
			", but expected " + strconv.Itoa(int(eventIndex)))
	}
	if tuple.SignalIndex != signalIndex {
		t.Fatal("unexpected signal index: got " + strconv.Itoa(int(tuple.SignalIndex)) +
			", but expected " + strconv.Itoa(int(signalIndex)))
	}
}
