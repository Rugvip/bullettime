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

	"github.com/matrix-org/bullettime/core/interfaces"
	"github.com/matrix-org/bullettime/core/types"
)

// Each user get their own buffer of received signals
//TODO: garbage collection(?)
type signalBuffer struct {
	lock    sync.RWMutex
	signals map[types.UserId][]types.IndexedEventId
	counter interfaces.Counter
}

func NewSignalBuffer(
	counter interfaces.Counter,
) (interfaces.SignalBuffer, error) {
	return &signalBuffer{
		signals: map[types.UserId][]types.IndexedEventId{},
		counter: counter,
	}, nil
}

func (s *signalBuffer) PushSignal(eventId types.EventId, recipients []types.UserId) types.Error {
	s.lock.Lock()
	defer s.lock.Unlock()
	index := s.counter.Inc() - 1
	for _, userId := range recipients {
		signals, existed := s.signals[userId]
		if !existed {
			signals = []types.IndexedEventId{}
		}
		replaced := false
		for _, signal := range signals {
			if signal.EventId == eventId {
				replaced = true
				signal.EventId = eventId
				signal.Index = index
				break
			}
		}
		if !replaced {
			signal := types.IndexedEventId{eventId, index}
			s.signals[userId] = append(signals, signal)
		}
	}
	return nil
}

func (s *signalBuffer) Max() uint64 {
	return s.counter.Get()
}

func (s *signalBuffer) Range(
	recipient types.UserId,
	fromIndex, toIndex uint64,
) ([]types.IndexedEventId, types.Error) {
	if fromIndex >= toIndex {
		return []types.IndexedEventId{}, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	signals := s.signals[recipient]
	if signals == nil {
		return []types.IndexedEventId{}, nil
	}
	result := make([]types.IndexedEventId, 0, len(signals))
	for _, signal := range signals {
		if signal.Index >= fromIndex && signal.Index < toIndex {
			result = append(result, signal)
		}
	}
	return result, nil
}
