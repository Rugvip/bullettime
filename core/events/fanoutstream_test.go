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

package events

import (
	"testing"

	"github.com/matrix-org/bullettime/core/interfaces"
	"github.com/matrix-org/bullettime/core/types"
)

func TestFanOutStream(t *testing.T) {
	counter := NewCounter(0)
	b, err := NewFanOutStreamBuffer(counter)
	if err != nil {
		t.Fatal("failed to create signal buffer: ", err)
	}
	buffer := testBuffer{t, b}

	buffer.Push("event1", "user1", "user2")
	buffer.Range("user1", 0, 2, 0, "event1")
	buffer.Range("user2", 0, 1, 0, "event1")
	buffer.Range("user1", 0, 0, 0)
	buffer.Range("user3", 0, 2, 2)
	buffer.Range("user1", 2, 0, 2)
	buffer.Push("event2", "user1")
	buffer.Range("user1", 0, 2, 1, "event1", "event2")
	buffer.Range("user2", 0, 2, 0, "event1")
	buffer.Push("event3", "user1", "user3")
	buffer.Range("user1", 0, 3, 2, "event1", "event2", "event3")
	buffer.Range("user1", 0, 2, 1, "event1", "event2")
	buffer.Range("user1", 1, 3, 2, "event2", "event3")
	buffer.Range("user3", 0, 3, 2, "event3")
}

type testBuffer struct {
	t      *testing.T
	buffer interfaces.FanOutStream
}

func (b *testBuffer) Push(id string, users ...string) {
	userIds := make([]types.Id, len(users))
	for i, user := range users {
		userIds[i] = types.Id(types.NewUserId(user, "test"))
	}
	err := b.buffer.SendMany(types.EventInfo{
		EventId:   types.NewEventId(id, "test"),
		Sender:    types.NewUserId("tester", "test"),
		ContextId: types.Id(types.NewRoomId("room1", "test")),
		EventType: "m.test",
	}, userIds)
	if err != nil {
		b.t.Fatal("error pushing signal: ", err)
	}
}

func (b *testBuffer) Range(user string, from, to, expectedIndex uint64, expected ...string) {
	signals, minIndex, maxIndex, err := b.buffer.Range(types.Id(types.NewUserId(user, "test")), from, to, 0)
	if err != nil {
		b.t.Fatal("failed to get signal range: ", err)
	}
	if len(signals) != len(expected) {
		b.t.Fatalf("invalid signal count, expected %d but got %d", len(expected), len(signals))
	}
	if from != minIndex {
		b.t.Fatalf("invalid max index, expected %d but got %d", from, minIndex)
	}
	if expectedIndex != maxIndex {
		b.t.Fatalf("invalid max index, expected %d but got %d", expectedIndex, maxIndex)
	}
	for i, signal := range signals {
		if signal.EventId.Id != expected[i] {
			b.t.Fatalf("invalid event id: expected %s, got %s", expected[i], signal.EventId.Id)
		}
	}
}
