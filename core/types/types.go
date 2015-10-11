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

package types

import (
	"strconv"
	"time"
)

type EventInfo struct {
	ContextId Id
	EventId   Id
	Sender    Id
	EventType string
}

func NewEventInfo(contextId, eventId, sender Id, eventType string) EventInfo {
	return EventInfo{contextId, eventId, sender, eventType}
}

type IndexedEventInfo struct {
	EventInfo
	Index uint64
}

func NewIndexedEventInfo(eventInfo EventInfo, index uint64) IndexedEventInfo {
	return IndexedEventInfo{eventInfo, index}
}

type Timestamp struct {
	time.Time
}

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	ms := ts.UnixNano() / int64(time.Millisecond)
	return []byte(strconv.FormatInt(ms, 10)), nil
}
