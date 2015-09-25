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

type Error interface {
	error
	Code() string
}

type internalError struct {
	code    string
	message string
}

func (e internalError) Code() string {
	return e.code
}

func (e internalError) Error() string {
	return e.message
}

func InvalidStateError(message string) Error {
	return internalError{
		code:    "INVALID_STATE",
		message: message,
	}
}

func ParseError(message string) Error {
	return internalError{
		code:    "PARSE_ERROR",
		message: message,
	}
}