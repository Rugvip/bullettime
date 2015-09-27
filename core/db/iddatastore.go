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

	"github.com/matrix-org/bullettime/core/interfaces"
	"github.com/matrix-org/bullettime/core/types"
)

type idDataStore struct {
	sync.RWMutex
	data map[types.Id]interface{}
}

func NewIdDataStore() (interfaces.IdDataStore, error) {
	return &idDataStore{
		data: map[types.Id]interface{}{},
	}, nil
}

func (s *idDataStore) Put(id types.Id, data interface{}) types.Error {
	s.Lock()
	defer s.Unlock()
	s.data[id] = data
	return nil
}

func (s *idDataStore) Lookup(id types.Id) (data interface{}, err types.Error) {
	s.RLock()
	defer s.RUnlock()
	return s.data[id], nil
}

func (s *idDataStore) LookupMany(ids []types.Id) (data []interface{}, err types.Error) {
	s.RLock()
	defer s.RUnlock()
	result := make([]interface{}, len(ids))
	for i, id := range ids {
		result[i] = s.data[id]
	}
	return result, nil
}
