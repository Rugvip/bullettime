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

	"github.com/matrix-org/bullettime/core/types"
	matrixTypes "github.com/matrix-org/bullettime/matrix/types"
)

func TestEventStreamMux(t *testing.T) {
	_es, err := NewStreamMux()
	if err != nil {
		t.Fatal(err)
	}
	es := StreamMuxTest{_es, t}
	cancel := make(chan struct{}, 1)
	streamA, err := es.Listen(types.NewUserId("userA", "test"), cancel)
	streamB, err := es.Listen(types.NewUserId("userB", "test"), cancel)
	streamC, err := es.Listen(types.NewUserId("userC", "test"), cancel)
	streamD, err := es.Listen(types.NewUserId("userD", "test"), cancel)
	streamE1, err := es.Listen(types.NewUserId("userE", "test"), cancel)
	streamE2, err := es.Listen(types.NewUserId("userE", "test"), cancel)
	streamE3, err := es.Listen(types.NewUserId("userE", "test"), cancel)
	es.send(typing("room1", "user1"), 1, "userA")
	es.send(typing("room2", "user2"), 1, "userB", "userC")
	es.send(typing("room3", "user3"), 1, "userA")
	close(cancel)
	resA := <-streamA
	resB := <-streamB
	resC := <-streamC
	resD := <-streamD
	resE1 := <-streamE1
	resE2 := <-streamE2
	resE3 := <-streamE3
	if resA == nil {
		t.Error("resA (", resA, ") expected to be not nil")
	} else if resA.Event().GetRoomId().Id != "room1" {
		t.Error("resA roomId (", resA.Event().GetRoomId(), ") expected to be room1")
	}
	if resB == nil {
		t.Error("resB (", resB, ") expected to be not nil")
	} else if resB.Event().GetRoomId().Id != "room2" {
		t.Error("resB roomId (", resB.Event().GetRoomId(), ") expected to be room2")
	}
	if resC == nil {
		t.Error("resC (", resC, ") expected to be not nil")
	} else if resC.Event().GetRoomId().Id != "room2" {
		t.Error("resC roomId (", resC.Event().GetRoomId(), ") expected to be room3")
	}
	if resD != nil {
		t.Error("resD (", resD, ") expected to be nil")
	}
	if resE1 != nil {
		t.Error("resE1( ", resE1, ") expected to be nil")
	}
	if resE2 != nil {
		t.Error("resE2( ", resE2, ") expected to be nil")
	}
	if resE3 != nil {
		t.Error("resE3( ", resE3, ") expected to be nil")
	}
}

type StreamMuxTest struct {
	*streamMux
	t *testing.T
}

func (es StreamMuxTest) send(event matrixTypes.Event, index uint64, ids ...string) {
	userIds := make([]types.UserId, len(ids))
	for i := range ids {
		userIds[i] = types.NewUserId(ids[i], "test")
	}
	err := es.Send(userIds, &indexedEvent{event, index})
	if err != nil {
		es.t.Fatal(err)
	}
}

func typing(id string, ids ...string) *matrixTypes.TypingEvent {
	event := &matrixTypes.TypingEvent{}
	event.EventType = "m.typing"
	event.RoomId = types.NewRoomId(id, "test")
	userIds := make([]types.UserId, len(ids))
	for i := range ids {
		userIds[i] = types.NewUserId(ids[i], "test")
	}
	event.Content.UserIds = userIds
	return event
}
