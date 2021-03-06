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
	"runtime/debug"
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
	buffer := fanOutTestBuffer{t, b}

	buffer.Push("event1", 0, "user1", "user2")
	buffer.Select("user1", 0, 2, 0, 1, "event1")
	buffer.Select("user2", 0, 1, 0, 1, "event1")
	buffer.Select("user1", 0, 0, 0, 0)
	buffer.Select("user3", 0, 2, 0, 1)
	buffer.Select("user0", 0, 2, 0, 1)
	buffer.Push("event2", 1, "user1")
	buffer.Select("user1", 0, 2, 0, 2, "event1", "event2")
	buffer.Select("user2", 0, 2, 0, 2, "event1")
	buffer.Push("event3", 2, "user1", "user3")
	buffer.Select("user1", 0, 3, 0, 3, "event1", "event2", "event3")
	buffer.Select("user1", 0, 2, 0, 2, "event1", "event2")
	buffer.Select("user1", 1, 3, 1, 3, "event2", "event3")
	buffer.Select("user2", 0, 3, 0, 3, "event1")
	buffer.Select("user2", 0, 5, 0, 3, "event1")
	buffer.Select("user2", 4, 5, 3, 3)
	buffer.Select("user3", 0, 3, 0, 3, "event3")

	expectPanic(t, func() {
		buffer.Select("user1", 2, 0, 0, 0)
	})
}

type fanOutTestBuffer struct {
	t      *testing.T
	buffer interfaces.FanOutStream
}

func (b *fanOutTestBuffer) Push(id string, expectedIndex uint64, users ...string) {
	userIds := make([]types.Id, len(users))
	for i, user := range users {
		userIds[i] = types.Id(types.NewUserId(user, "test"))
	}
	index, err := b.buffer.Send(types.EventInfo{
		EventId:   types.Id(types.NewEventId(id, "test")),
		Sender:    types.Id(types.NewUserId("tester", "test")),
		ContextId: types.Id(types.NewRoomId("room1", "test")),
		EventType: "m.test",
	}, userIds)
	if err != nil {
		b.t.Fatal("error pushing signal: ", err)
	}
	if index != expectedIndex {
		debug.PrintStack()
		b.t.Fatal("invalid index, expected %d but got %d", expectedIndex, index)
	}
}

func (b *fanOutTestBuffer) Select(user string, from, to, expectedFromIndex, expectedToIndex uint64, expected ...string) {
	signals, fromIndex, toIndex, err := b.buffer.SelectForwards(types.Id(types.NewUserId(user, "test")), from, to)
	if err != nil {
		debug.PrintStack()
		b.t.Fatal("failed to get signal range: ", err)
	}
	if len(signals) != len(expected) {
		debug.PrintStack()
		b.t.Fatalf("invalid signal count, expected %d but got %d", len(expected), len(signals))
	}
	if expectedFromIndex != fromIndex {
		debug.PrintStack()
		b.t.Fatalf("invalid from index, expected %d but got %d", expectedFromIndex, fromIndex)
	}
	if expectedToIndex != toIndex {
		debug.PrintStack()
		b.t.Fatalf("invalid to index, expected %d but got %d", expectedToIndex, toIndex)
	}
	for i, signal := range signals {
		if signal.EventId.Id != expected[i] {
			debug.PrintStack()
			b.t.Fatalf("invalid event id: expected %s, got %s", expected[i], signal.EventId.Id)
		}
	}
}
