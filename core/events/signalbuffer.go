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
	signals map[types.UserId][]indexedSignal
	counter interfaces.Counter
}

type indexedSignal struct {
	info  types.EventInfo
	index uint64
}

func NewSignalBuffer(
	counter interfaces.Counter,
) (interfaces.SignalBuffer, error) {
	return &signalBuffer{
		signals: map[types.UserId][]indexedSignal{},
		counter: counter,
	}, nil
}

func (s *signalBuffer) PushSignal(eventInfo types.EventInfo, recipients []types.UserId) types.Error {
	s.lock.Lock()
	defer s.lock.Unlock()
	index := s.counter.Inc() - 1
	for _, userId := range recipients {
		signals, existed := s.signals[userId]
		if !existed {
			signals = []indexedSignal{}
		}
		pos := -1
		for i, signal := range signals {
			if signal.info.EventId == eventInfo.EventId {
				pos = i
				break
			}
		}
		signal := indexedSignal{eventInfo, index}
		if pos >= 0 {
			signals[pos] = signal
		} else {
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
) (events []types.EventInfo, maxIndex uint64, err types.Error) {
	if fromIndex >= toIndex {
		return []types.EventInfo{}, 0, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	signals := s.signals[recipient]
	if signals == nil {
		return []types.EventInfo{}, 0, nil
	}
	events = make([]types.EventInfo, 0, len(signals))
	for _, signal := range signals {
		if signal.index >= fromIndex && signal.index < toIndex {
			events = append(events, signal.info)
			if maxIndex < signal.index {
				maxIndex = signal.index
			}
		}
	}
	return events, maxIndex, nil
}
