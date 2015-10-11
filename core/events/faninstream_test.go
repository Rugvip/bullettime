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

func TestFanInStream(t *testing.T) {
	counter := NewCounter(0)
	b, err := NewFanInStreamBuffer(counter)
	if err != nil {
		t.Fatal("failed to create signal buffer: ", err)
	}
	buffer := fanInTestBuffer{t, b}

	buffer.Send("event1", "user1")
	buffer.Send("event2", "user2")
	buffer.Send("event3", "user1")
	buffer.Send("event4", "user1")
	buffer.Send("event5", "user2")
	buffer.Send("event6", "user3")
	buffer.Send("event7", "user1")

	buffer.SelectBackwards("user3", 2, 0, 1, 0, "event6")
	buffer.SelectBackwards("user3", 1, 0, 1, 0, "event6")
	buffer.SelectBackwards("user3", 3, 1, 1, 1)
	buffer.SelectBackwards("user3", 0, 0, 0, 0)

	buffer.SelectBackwards("user2", 2, 0, 2, 0, "event5", "event2")
	buffer.SelectBackwards("user2", 1, 0, 1, 0, "event2")
	buffer.SelectBackwards("user2", 2, 1, 2, 1, "event5")
	buffer.SelectBackwards("user2", 3, 1, 2, 1, "event5")
	buffer.SelectBackwards("user2", 0, 0, 0, 0)

	buffer.SelectBackwards("user1", 4, 0, 4, 0, "event7", "event4", "event3", "event1")
	buffer.SelectBackwards("user1", 8, 0, 4, 0, "event7", "event4", "event3", "event1")
	buffer.SelectBackwards("user1", 8, 2, 4, 2, "event7", "event4")
	buffer.SelectBackwards("user1", 3, 1, 3, 1, "event4", "event3")
	buffer.SelectBackwards("user1", 3, 0, 3, 0, "event4", "event3", "event1")
	buffer.SelectBackwards("user1", 0, 0, 0, 0)

	buffer.SelectBackwards("user0", 3, 0, 0, 0)

	expectPanic(t, func() {
		buffer.SelectBackwards("user0", 1, 2, 0, 0)
	})

	buffer.SelectForwards([]string{}, 0, 0, 0, 0)
	buffer.SelectForwards([]string{}, 1, 1, 1, 1)
	buffer.SelectForwards([]string{}, 0, 1, 0, 1)
	buffer.SelectForwards([]string{}, 0, 7, 0, 7)
	buffer.SelectForwards([]string{}, 0, 8, 0, 7)
	buffer.SelectForwards([]string{}, 3, 8, 3, 7)
	buffer.SelectForwards([]string{"user0"}, 2, 10, 2, 7)

	buffer.SelectForwards([]string{"user1"}, 0, 10, 0, 7, "event1", "event3", "event4", "event7")
	buffer.SelectForwards([]string{"user1"}, 3, 10, 3, 7, "event4", "event7")
	buffer.SelectForwards([]string{"user1"}, 4, 10, 4, 7, "event7")
	buffer.SelectForwards([]string{"user1"}, 8, 10, 7, 7)

	buffer.SelectForwards([]string{"user1", "user2"}, 4, 7, 4, 7, "event5", "event7")
	buffer.SelectForwards([]string{"user1", "user2"}, 4, 6, 4, 6, "event5")
	buffer.SelectForwards([]string{"user1", "user2"}, 4, 10, 4, 7, "event5", "event7")
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 4, 10, 4, 7, "event5", "event6", "event7")
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 0, 0, 0, 0)
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 0, 1, 0, 1, "event1")
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 1, 1, 1, 1)
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 1, 2, 1, 2, "event2")
	buffer.SelectForwards([]string{"user1", "user2", "user3"}, 1, 5, 1, 5, "event2", "event3", "event4", "event5")

	expectPanic(t, func() {
		buffer.SelectForwards([]string{}, 1, 0, 0, 0)
	})
}

type fanInTestBuffer struct {
	t      *testing.T
	buffer interfaces.FanInStream
}

func (b *fanInTestBuffer) Send(id string, user string) {
	err := b.buffer.Send(types.EventInfo{
		EventId:   types.Id(types.NewEventId(id, "test")),
		Sender:    types.Id(types.NewUserId("tester", "test")),
		ContextId: types.Id(types.NewRoomId("room1", "test")),
		EventType: "m.test",
	}, types.Id(types.NewUserId(user, "test")))
	if err != nil {
		debug.PrintStack()
		b.t.Fatal("error pushing signal: ", err)
	}
}

func (b *fanInTestBuffer) SelectBackwards(
	user string,
	from, to,
	expectedFromIndex, expectedToIndex uint64,
	expected ...string,
) {
	signals, fromIndex, toIndex, err := b.buffer.SelectBackwards(types.Id(types.NewUserId(user, "test")), from, to)
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

func (b *fanInTestBuffer) SelectForwards(
	users []string,
	from, to,
	expectedFromIndex, expectedToIndex uint64,
	expected ...string,
) {
	userIds := make([]types.Id, len(users))
	for i, user := range users {
		userIds[i] = types.Id(types.NewUserId(user, "test"))
	}
	signals, fromIndex, toIndex, err := b.buffer.SelectForwards(userIds, from, to)
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

func expectPanic(t *testing.T, fun func()) {
	panicked := func() (ret bool) {
		retP := &ret
		defer func() {
			if p := recover(); p != nil {
				*retP = true
			}
		}()
		fun()
		return
	}()
	if !panicked {
		debug.PrintStack()
		t.Fatal("Function expected to panic but didn't")
	}
}
