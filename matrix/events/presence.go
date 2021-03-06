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
	"sync"
	"sync/atomic"

	ct "github.com/matrix-org/bullettime/core/types"
	"github.com/matrix-org/bullettime/matrix/interfaces"
	"github.com/matrix-org/bullettime/matrix/types"
)

type presenceStream struct {
	lock           sync.RWMutex
	events         map[ct.UserId]indexedPresenceEvent
	max            uint64
	members        interfaces.MembershipStore
	asyncEventSink interfaces.AsyncEventSink
}

type indexedPresenceEvent struct {
	event types.PresenceEvent
	index uint64
}

func (m *indexedPresenceEvent) Event() types.Event {
	return &m.event
}

func (s *indexedPresenceEvent) Index() uint64 {
	return s.index
}

type updateFunc func(*types.User)

func NewPresenceStream(
	members interfaces.MembershipStore,
	asyncEventSink interfaces.AsyncEventSink,
) (interfaces.PresenceStream, error) {
	return &presenceStream{
		events:         map[ct.UserId]indexedPresenceEvent{},
		members:        members,
		asyncEventSink: asyncEventSink,
	}, nil
}

func (s *presenceStream) SetUserProfile(userId ct.UserId, profile types.UserProfile) (types.IndexedEvent, types.Error) {
	return s.update(userId, func(user *types.User) {
		user.UserProfile = profile
	})
}

func (s *presenceStream) SetUserStatus(userId ct.UserId, status types.UserStatus) (types.IndexedEvent, types.Error) {
	return s.update(userId, func(user *types.User) {
		user.UserStatus = status
	})
}

func (s *presenceStream) update(userId ct.UserId, updateFunc updateFunc) (types.IndexedEvent, types.Error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	indexed, existed := s.events[userId]
	if !existed {
		indexed.event.Content.UserId = userId
		indexed.event.EventType = types.EventTypePresence
	}
	updateFunc(&indexed.event.Content)
	index := atomic.AddUint64(&s.max, 1) - 1
	indexed.index = index
	s.events[userId] = indexed
	peerSet, err := s.members.Peers(userId)
	if err != nil {
		return nil, err
	}
	peers := make([]ct.UserId, len(peerSet))
	for peer := range peerSet {
		peers = append(peers, peer)
	}
	s.asyncEventSink.Send(peers, &indexed)
	return &indexed, nil
}

func (s *presenceStream) Profile(user ct.UserId) (types.UserProfile, types.Error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if indexed, ok := s.events[user]; ok {
		return indexed.event.Content.UserProfile, nil
	}
	return types.UserProfile{}, nil
}

func (s *presenceStream) Status(user ct.UserId) (types.UserStatus, types.Error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if indexed, ok := s.events[user]; ok {
		return indexed.event.Content.UserStatus, nil
	}
	return types.UserStatus{}, nil
}

func (s *presenceStream) Max() uint64 {
	return atomic.LoadUint64(&s.max)
}

// Ignores user, roomSet, and limit
func (s *presenceStream) Range(
	_ *ct.UserId,
	userSet map[ct.UserId]struct{},
	roomSet map[ct.RoomId]struct{},
	from, to uint64,
	limit uint,
) ([]types.IndexedEvent, types.Error) {
	var result []types.IndexedEvent
	if len(userSet) == 0 || from >= to {
		return result, nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	result = make([]types.IndexedEvent, 0, len(userSet))
	for user := range userSet {
		event := s.events[user]
		if event.index >= from && event.index < to {
			result = append(result, &event)
		}
	}
	return result, nil
}
