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
	"fmt"
	"sort"
	"sync"

	"github.com/matrix-org/bullettime/core/interfaces"
	"github.com/matrix-org/bullettime/core/types"
)

type fanInStream struct {
	lock     sync.RWMutex
	segments map[types.Id]*[]indexedFanInEvent
	counter  interfaces.Counter
}

type indexedFanInEvent struct {
	info  types.EventInfo
	index uint64
}

type eventsByIndex []indexedFanInEvent

func (e eventsByIndex) Len() int           { return len(e) }
func (e eventsByIndex) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e eventsByIndex) Less(i, j int) bool { return e[i].index < e[j].index }

func NewFanInStreamBuffer(counter interfaces.Counter) (interfaces.FanInStream, error) {
	return &fanInStream{
		segments: map[types.Id]*[]indexedFanInEvent{},
		counter:  counter,
	}, nil
}

func (s *fanInStream) Send(eventInfo types.EventInfo, recipient types.Id) (uint64, types.Error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	index := s.counter.Inc() - 1
	indexed := indexedFanInEvent{eventInfo, index}

	segment := s.segments[recipient]
	if segment == nil {
		s.segments[recipient] = &[]indexedFanInEvent{indexed}
	} else {
		*segment = append(*segment, indexed)
	}
	return index, nil
}

func (s *fanInStream) Max() uint64 {
	return s.counter.Get()
}

func (s *fanInStream) SelectBackwards(
	recipient types.Id,
	from, to uint64,
) (result []types.EventInfo, fromIndex, toIndex uint64, err types.Error) {
	if from < to {
		panic(fmt.Sprintf("Invalid arguments: from < to, (%d < %d)", from, to))
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	segment := s.segments[recipient]
	if segment == nil {
		return []types.EventInfo{}, 0, 0, nil
	}

	max := len(*segment)
	if from > uint64(max) {
		from = uint64(max)
	}
	if from <= to {
		return result, from, from, nil
	}
	events := (*segment)[to:from]
	eventCount := len(events)
	result = make([]types.EventInfo, eventCount)
	for i, indexed := range events {
		result[eventCount-i-1] = indexed.info
	}
	return result, from, to, nil
}
func (s *fanInStream) SelectForwards(
	recipients []types.Id,
	from, to uint64,
) (result []types.EventInfo, fromIndex, toIndex uint64, err types.Error) {
	if from > to {
		panic(fmt.Sprintf("Invalid arguments: from > to, (%d > %d)", from, to))
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	result = make([]types.EventInfo, 0, 0)

	max := s.counter.Get()
	if to > max {
		to = max
	}
	if from >= to {
		return result, to, to, nil
	}

	indexedResult := []indexedFanInEvent{}
	for _, recipient := range recipients {
		segment := s.segments[recipient]
		if segment == nil {
			continue
		}

		i := len(*segment) - 1
		for i >= 0 {
			indexed := (*segment)[i]
			i -= 1
			if indexed.index >= to {
				continue
			}
			if indexed.index < from {
				break
			}
			indexedResult = append(indexedResult, indexed)
		}
	}

	sort.Sort(eventsByIndex(indexedResult))

	result = make([]types.EventInfo, len(indexedResult))
	for i := range indexedResult {
		result[i] = indexedResult[i].info
	}

	return result, from, to, nil
}
