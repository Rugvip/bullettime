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

package db

import (
	"sync"
	"time"

	"github.com/matrix-org/bullettime/core/types"
	matrixInterfaces "github.com/matrix-org/bullettime/matrix/interfaces"
	matrixTypes "github.com/matrix-org/bullettime/matrix/types"
	"github.com/matrix-org/bullettime/utils"
)

type roomDb struct { // always lock in the same order as below
	roomsLock sync.RWMutex
	rooms     map[types.RoomId]*dbRoom
}

func NewRoomDb() (matrixInterfaces.RoomStore, error) {
	return &roomDb{
		rooms: map[types.RoomId]*dbRoom{},
	}, nil
}

type stateId struct {
	EventType string
	StateKey  string
}

type dbRoom struct { // always lock in the same order as below
	id        types.RoomId
	stateLock sync.RWMutex
	states    map[stateId]*matrixTypes.State
}

func (db *roomDb) CreateRoom(id types.RoomId) (exists bool, err matrixTypes.Error) {
	db.roomsLock.Lock()
	defer db.roomsLock.Unlock()
	if db.rooms[id] != nil {
		return true, nil
	}
	db.rooms[id] = &dbRoom{
		id:     id,
		states: map[stateId]*matrixTypes.State{},
	}
	return false, nil
}

func (db *roomDb) RoomExists(id types.RoomId) (bool, matrixTypes.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	if db.rooms[id] == nil {
		return false, nil
	}
	return true, nil
}

func (db *roomDb) SetRoomState(roomId types.RoomId, userId types.UserId, content matrixTypes.TypedContent, stateKey string) (*matrixTypes.State, matrixTypes.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, matrixTypes.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	var eventId = types.DeriveEventId(utils.RandomString(16), types.Id(userId))
	stateId := stateId{content.GetEventType(), stateKey}

	state := new(matrixTypes.State)
	state.EventId = eventId
	state.RoomId = roomId
	state.UserId = userId
	state.EventType = content.GetEventType()
	state.StateKey = stateKey
	state.Timestamp = types.Timestamp{time.Now()}
	state.Content = content
	state.OldState = (*matrixTypes.OldState)(room.states[stateId])

	room.stateLock.Lock()
	defer room.stateLock.Unlock()
	room.states[stateId] = state

	return state, nil
}

func (db *roomDb) RoomState(roomId types.RoomId, eventType, stateKey string) (*matrixTypes.State, matrixTypes.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, matrixTypes.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	room.stateLock.RLock()
	defer room.stateLock.RUnlock()
	state := room.states[stateId{eventType, stateKey}]
	return state, nil
}

func (db *roomDb) EntireRoomState(roomId types.RoomId) ([]*matrixTypes.State, matrixTypes.Error) {
	db.roomsLock.RLock()
	defer db.roomsLock.RUnlock()
	room := db.rooms[roomId]
	if room == nil {
		return nil, matrixTypes.NotFoundError("room '" + roomId.String() + "' doesn't exist")
	}
	room.stateLock.RLock()
	defer room.stateLock.RUnlock()
	states := make([]*matrixTypes.State, 0, len(room.states))
	for _, state := range room.states {
		states = append(states, state)
	}
	return states, nil
}
