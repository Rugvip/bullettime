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
	"strings"
)

/*

char = A-Z | a-z | 0-9 | "_" | "-";
valueSeparator = ".";
tupleSeparator = "_";

shardId = 1-2 * char;
eventIndex = 1-6 * char;
signalIndex = 1-6 * chat;
tuple = shardId, valueSeparator, eventIndex, valueSeparator, signalIndex;
streamToken = tuple, { tupleSeparator, tuple };
*/
// A.BC.DE_F.GH.IJ_K.LM.NO_P.Q.R_S.TU.V

const valueSeparator = "."
const tupleSeparator = "_"

type ShardId uint32

type ShardTuple struct {
	ShardId     ShardId
	EventIndex  uint64
	SignalIndex uint64
}

type StreamToken []ShardTuple

func parseError(message string) Error {
	return ParseError("failed to parse stream token: " + message)
}

func (id ShardId) TotalCount() int {
	if id == 0 {
		return 0
	}
	value := 1
	for {
		id >>= 1
		if id > 0 {
			value <<= 1
		} else {
			break
		}
	}
	return value
}

func (id ShardId) Index() int {
	high := id.TotalCount()
	return ^high & int(id)
}

func parseShardId(str string) (ShardId, Error) {
	if len(str) == 0 || len(str) > 2 {
		return 0, parseError("invalid shard id length: " + strconv.Itoa(len(str)))
	}
	value, err := parseValue(str)
	if err != nil {
		return 0, err
	}
	return ShardId(value), nil
}

func parseIndex(str string) (uint64, Error) {
	if len(str) == 0 || len(str) > 8 {
		return 0, parseError("invalid index length: " + strconv.Itoa(len(str)))
	}
	return parseValue(str)
}

// str length should be validated before calling
func parseValue(str string) (uint64, Error) {
	var value uint64 = 0
	for index, charRune := range str {
		char := uint64(charRune)
		value *= 64 // base 64 encoded, url-safe
		if 65 <= char && char < 91 {
			value += char - 65
		} else if 97 <= char && char < 122 {
			value += char - 97 + (91 - 65)
		} else if 48 <= char && char < 58 {
			value += char - 48 + (91 - 65) + (122 - 97)
		} else if char == '-' {
			value += 62
		} else if char == '_' {
			value += 63
		} else {
			return 0, parseError("invalid character '" + string(char) + "' at index " + strconv.Itoa(index))
		}
	}
	return uint64(value), nil
}

func parseTuple(str string) (ShardTuple, Error) {
	valueStrs := strings.Split(str, valueSeparator)
	if len(valueStrs) != 3 {
		return ShardTuple{}, parseError("invalid tuple, " + str)
	}
	shardId, err := parseShardId(valueStrs[0])
	if err != nil {
		return ShardTuple{}, err
	}
	eventIndex, err := parseValue(valueStrs[1])
	if err != nil {
		return ShardTuple{}, err
	}
	signalIndex, err := parseValue(valueStrs[2])
	if err != nil {
		return ShardTuple{}, err
	}
	return ShardTuple{shardId, eventIndex, signalIndex}, nil
}

func ParseStreamToken(str string) (StreamToken, Error) {
	tupleStrs := strings.Split(str, tupleSeparator)
	if len(str) == 0 {
		return []ShardTuple{}, nil
	}
	tuples := make([]ShardTuple, len(tupleStrs))
	for index, tupleStr := range tupleStrs {
		tuple, err := parseTuple(tupleStr)
		if err != nil {
			return nil, err
		}
		tuples[index] = tuple
	}
	return StreamToken(tuples), nil
}
