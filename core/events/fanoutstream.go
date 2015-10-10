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

// Each user get their own buffer of received events
//TODO: garbage collection(?)
type fanOutStreamBuffer struct {
	lock    sync.RWMutex
	events  map[types.Id][]indexedFanOutEvent
	counter interfaces.Counter
}

type indexedFanOutEvent struct {
	info  types.EventInfo
	index uint64
}

func NewFanOutStreamBuffer(
	counter interfaces.Counter,
) (interfaces.FanOutStream, error) {
	return &fanOutStreamBuffer{
		events:  map[types.Id][]indexedFanOutEvent{},
		counter: counter,
	}, nil
}

func (sb *fanOutStreamBuffer) Send(eventInfo types.EventInfo, recipient types.Id) types.Error {
	return sb.SendMany(eventInfo, []types.Id{recipient})
}

func (sb *fanOutStreamBuffer) SendMany(eventInfo types.EventInfo, recipients []types.Id) types.Error {
	sb.lock.Lock()
	defer sb.lock.Unlock()
	index := sb.counter.Inc() - 1
	for _, userId := range recipients {
		events, existed := sb.events[userId]
		if !existed {
			events = []indexedFanOutEvent{}
		}
		pos := -1
		for i, event := range events {
			if event.info.EventId == eventInfo.EventId {
				pos = i
				break
			}
		}
		event := indexedFanOutEvent{eventInfo, index}
		if pos >= 0 {
			events[pos] = event
		} else {
			sb.events[userId] = append(events, event)
		}
	}
	return nil
}

func (sb *fanOutStreamBuffer) Max() uint64 {
	return sb.counter.Get()
}

func (sb *fanOutStreamBuffer) SelectForwards(
	recipient types.Id,
	fromIndex, toIndex uint64,
) (result []types.EventInfo, minIndex, maxIndex uint64, err types.Error) {
	if fromIndex >= toIndex {
		return []types.EventInfo{}, fromIndex, fromIndex, nil
	}
	sb.lock.Lock()
	defer sb.lock.Unlock()
	events := sb.events[recipient]
	if events == nil {
		return []types.EventInfo{}, fromIndex, toIndex, nil
	}
	result = make([]types.EventInfo, 0, len(events))
	for _, event := range events {
		if event.index >= fromIndex && event.index < toIndex {
			result = append(result, event.info)
			if maxIndex < event.index {
				maxIndex = event.index
			}
		}
	}
	return result, fromIndex, maxIndex, nil
}
