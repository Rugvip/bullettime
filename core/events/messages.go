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
	"container/list"
	"log"
	"sync"
	"sync/atomic"

	"github.com/matrix-org/bullettime/core/types"
	"github.com/matrix-org/bullettime/matrix/interfaces"
	matrixTypes "github.com/matrix-org/bullettime/matrix/types"
)

type indexedEvent struct {
	event matrixTypes.Event
	index uint64
}

func (m *indexedEvent) Event() matrixTypes.Event {
	return m.event
}

func (m *indexedEvent) Index() uint64 {
	return m.index
}

type messageStream struct {
	lock           sync.RWMutex
	list           *list.List
	byId           map[types.Id]indexedEvent
	byIndex        []*indexedEvent
	max            uint64
	members        interfaces.MembershipStore
	asyncEventSink interfaces.AsyncEventSink
}

func NewMessageStream(
	members interfaces.MembershipStore,
	asyncEventSink interfaces.AsyncEventSink,
) (interfaces.EventStream, error) {
	return &messageStream{
		list:           list.New(),
		byId:           map[types.Id]indexedEvent{},
		byIndex:        []*indexedEvent{},
		members:        members,
		asyncEventSink: asyncEventSink,
	}, nil
}

func (s *messageStream) Send(event matrixTypes.Event) (uint64, matrixTypes.Error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	index := atomic.AddUint64(&s.max, 1) - 1
	indexed := indexedEvent{event, index}

	if currentItem, ok := s.byId[event.GetEventKey()]; ok {
		s.byIndex[currentItem.index] = nil
	}
	s.byIndex = append(s.byIndex, &indexed)
	s.byId[event.GetEventKey()] = indexed

	users, err := s.members.Users(*event.GetRoomId())
	if err != nil {
		return 0, nil
	}
	if extraUser := extraUserForEvent(event); extraUser != nil {
		l := len(users)
		allUsers := make([]types.UserId, l+1)
		copy(allUsers, users)
		allUsers[l] = *extraUser
		users = allUsers
	}
	s.asyncEventSink.Send(users, &indexed)
	return index, nil
}

func extraUserForEvent(event matrixTypes.Event) *types.UserId {
	if event.GetEventType() == matrixTypes.EventTypeMembership {
		membership := event.GetContent().(*matrixTypes.MembershipEventContent).Membership
		isInvited := membership == matrixTypes.MembershipInvited
		isKnocking := membership == matrixTypes.MembershipKnocking
		isBanned := membership == matrixTypes.MembershipBanned
		if isInvited || isKnocking || isBanned {
			state, ok := event.(*matrixTypes.State)
			if !ok {
				log.Println("membership event was not a state event:", event)
				return nil
			}
			user, err := types.ParseUserId(state.StateKey)
			if err != nil {
				log.Println("failed to parse user id state key:", state.StateKey)
				return nil
			}
			return &user
		}
	}
	return nil
}

func (s *messageStream) Event(
	user types.UserId,
	eventId types.EventId,
) (matrixTypes.Event, matrixTypes.Error) {
	s.lock.RLock()
	indexed := s.byId[types.Id(eventId)]
	s.lock.RUnlock()
	extraUser := extraUserForEvent(indexed.event)
	if extraUser != nil && *extraUser == user {
		return indexed.event, nil
	}
	rooms, err := s.members.Rooms(user)
	if err != nil {
		return nil, err
	}
	for _, room := range rooms {
		if room == *indexed.event.GetRoomId() {
			return indexed.event, nil
		}
	}
	return nil, nil
}

// ignores userSet
func (s *messageStream) Range(
	user *types.UserId,
	userSet map[types.UserId]struct{},
	roomSet map[types.RoomId]struct{},
	from, to uint64,
	limit uint,
) ([]matrixTypes.IndexedEvent, matrixTypes.Error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	result := make([]matrixTypes.IndexedEvent, 0, limit)
	reverse := to < from

	max := atomic.LoadUint64(&s.max)
	if reverse {
		if from >= max {
			from = max
		}
		from -= 1
		if from < to {
			return result, nil
		}
	} else {
		if from == to {
			return result, nil
		}
	}
	i := from
	for uint(len(result)) < limit && i < max {
		indexed := s.byIndex[i]
		if indexed != nil {
			_, ok := roomSet[*indexed.Event().GetRoomId()]
			if ok {
				result = append(result, indexed)
			} else if user != nil {
				extra := extraUserForEvent(indexed.event)
				if extra != nil && *extra == *user {
					result = append(result, indexed)
				}
			}
		}
		if reverse {
			i -= 1
			if i < to {
				break
			}
		} else {
			i += 1
			if i >= to {
				break
			}
		}
	}
	return result, nil
}

func (s *messageStream) Max() uint64 {
	return atomic.LoadUint64(&s.max)
}
