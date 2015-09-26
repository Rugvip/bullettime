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
	"strconv"
	"sync"
	"testing"
)

func TestCounter(t *testing.T) {
	iterationCount := 100
	callCount := 10
	initial := 50
	expected := initial + iterationCount*callCount

	wg := sync.WaitGroup{}
	wg.Add(iterationCount)
	counter := NewCounter(uint64(initial))
	for i := 0; i < iterationCount; i++ {
		go func() {
			for j := 0; j < callCount; j++ {
				counter.Inc()
			}
			wg.Done()
		}()
	}
	wg.Wait()

	value := counter.Get()
	if int(value) != expected {
		t.Fatal("invalid counter state: expected " + strconv.Itoa(int(expected)) +
			" but got " + strconv.Itoa(int(value)))
	}

	value2 := counter.Inc()
	if value+1 != value2 {
		t.Fatal("expected value2 to be " + strconv.Itoa(int(value+1)) +
			" but was " + strconv.Itoa(int(value2)))
	}
}
