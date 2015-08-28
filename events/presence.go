package events

import (
	"sync"
	"sync/atomic"

	"github.com/Rugvip/bullettime/interfaces"

	"github.com/Rugvip/bullettime/types"
)

type presenceSource struct {
	lock           sync.RWMutex
	events         map[types.UserId]indexedPresenceEvent
	max            uint64
	members        interfaces.MembershipStore
	asyncEventSink interfaces.AsyncEventSink
}

type indexedPresenceEvent struct {
	types.PresenceEvent
	index uint64
}

func (s *indexedPresenceEvent) Index() uint64 {
	return s.index
}

type updateFunc func(*types.User)

func NewPresenceSource(
	members interfaces.MembershipStore,
	asyncEventSink interfaces.AsyncEventSink,
) (interfaces.PresenceStream, error) {
	return &presenceSource{
		events:         map[types.UserId]indexedPresenceEvent{},
		members:        members,
		asyncEventSink: asyncEventSink,
	}, nil
}

func (s *presenceSource) SetUserProfile(userId types.UserId, profile types.UserProfile) (types.IndexedEvent, types.Error) {
	return s.update(userId, func(user *types.User) {
		user.UserProfile = profile
	})
}

func (s *presenceSource) SetUserPresence(userId types.UserId, presence types.UserPresence) (types.IndexedEvent, types.Error) {
	return s.update(userId, func(user *types.User) {
		user.UserPresence = presence
	})
}

func (s *presenceSource) update(userId types.UserId, updateFunc updateFunc) (types.IndexedEvent, types.Error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	event, existed := s.events[userId]
	if !existed {
		event.Content.UserId = userId
	}
	updateFunc(&event.PresenceEvent.Content)
	index := atomic.AddUint64(&s.max, 1) - 1
	event.index = index
	s.events[userId] = event
	peers, err := s.members.Peers(userId)
	if err != nil {
		return nil, err
	}
	s.asyncEventSink.Send(peers, &event.PresenceEvent, index)
	return &event, nil
}

func (s *presenceSource) User(user types.UserId) (types.User, types.Error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if event, ok := s.events[user]; ok {
		return event.Content, nil
	}
	return types.User{}, nil
}

func (s *presenceSource) Max() (uint64, types.Error) {
	return atomic.LoadUint64(&s.max), nil
}

// Ignores user, roomSet, and limit
func (s *presenceSource) Range(
	user types.UserId,
	userSet map[types.UserId]struct{},
	roomSet map[types.RoomId]struct{},
	from, to uint64,
	limit int,
) ([]types.Event, types.Error) {
	var result []types.Event
	if len(userSet) == 0 || from >= to {
		return result, nil
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	result = make([]types.Event, len(userSet))
	for user := range userSet {
		event := s.events[user]
		if event.index >= from && event.index < to {
			result = append(result, &event)
		}
	}
	return result, nil
}
