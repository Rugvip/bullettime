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

package interfaces

import "github.com/matrix-org/bullettime/core/types"

type IdMap interface {
	// Does nothing and returns false if the mapping already exists
	Insert(key types.Id, value types.Id) (inserted bool, err types.Error)
	// Does nothing and returns false if the mapping doesn't already exist
	Replace(key types.Id, value types.Id) (replaced bool, err types.Error)
	// Inserts or replaces as needed
	Put(key types.Id, value types.Id) types.Error
	// Does noting and returns false if the mapping doesn't exist
	Delete(key types.Id, value types.Id) (deleted bool, err types.Error)
	Lookup(key types.Id) (*types.Id, types.Error)
	ReverseLookup(value types.Id) ([]types.Id, types.Error)
}

type IdMultiMap interface {
	// Stores a key-value pair in the map, returns true of the mapping didn't already exist
	Put(key types.Id, value types.Id) (inserted bool, err types.Error)
	// Removes a mapping from the map, returns true if the mapping existed
	Delete(key types.Id, value types.Id) (deleted bool, err types.Error)
	// Returns true if the given key/value pair exists in the map
	Contains(key types.Id, value types.Id) (exists bool, err types.Error)

	Lookup(key types.Id) ([]types.Id, types.Error)
	ReverseLookup(value types.Id) ([]types.Id, types.Error)
}

type DataTransformFunc func(data interface{}) interface{}

type IdDataCache interface {
	Put(id types.Id, fieldId int, data interface{})
	LockedTransform(id types.Id, fieldId int, fun DataTransformFunc)
	Lookup(id types.Id, fieldId int) interface{}
}

type IdDataStore interface {
	Put(id types.Id, data interface{}) types.Error
	Lookup(id types.Id) (data interface{}, err types.Error)
	LookupMany(ids []types.Id) (data []interface{}, err types.Error)
}

// A counter to be shared between multiple consumers, operations must be atomic
type Counter interface {
	Get() uint64
	// Increments the counter by 1 and returns the new value
	Inc() uint64
}

// A stream where each event has multiple recipients, but queries are done for a single recipient
type FanOutStream interface {
	// Sends an event to a set of recipients
	Send(eventInfo types.EventInfo, recipients []types.Id) (index uint64, err types.Error)

	// Gets the current maximum index, this is the highest index of any event + 1
	Max() uint64

	// Queries events for a recipient in chronological order.
	// from <= to, fromIndex <= toIndex
	// from index is inclusive, while to index is exclusive
	SelectForwards(
		recipient types.Id,
		from, to uint64,
	) (events []types.EventInfo, fromIndex, toIndex uint64, err types.Error)
}

// A stream where each even has a single recipient, but queries can span multiple recipients
type FanInStream interface {
	// Sends an event to a single recipient
	Send(eventInfo types.EventInfo, recipient types.Id) (index uint64, err types.Error)

	// Gets the current maximum index, this is the highest index of any event + 1
	Max() uint64

	// Queries past events for a single recipient in reversed chronological order.
	// from >= to, fromIndex >= toIndex
	// from index is exclusive, while to index is inclusive
	SelectBackwards(
		recipient types.Id,
		from, to uint64,
	) (events []types.EventInfo, fromIndex, toIndex uint64, err types.Error)

	// Queries events for a set of recipients in chronological order.
	// from <= to, fromIndex <= toIndex
	// from index is inclusive, while to index is exclusive
	SelectForwards(
		recipients []types.Id,
		from, to uint64,
	) (events []types.EventInfo, fromIndex, toIndex uint64, err types.Error)
}

type State interface {
	Key() string
	Value() []byte
}

type StateStore interface {
	CreateBucket(types.Id) (exists bool, err types.Error)
	BucketExists(types.Id) (exists bool, err types.Error)
	SetState(id types.Id, key string, value []byte) (oldValue []byte, err types.Error)
	State(id types.Id, key string) (value []byte, err types.Error)
	States(id types.Id) ([]State, types.Error)
}
