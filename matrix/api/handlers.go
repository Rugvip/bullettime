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

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	ct "github.com/matrix-org/bullettime/core/types"
	"github.com/matrix-org/bullettime/matrix/interfaces"
	"github.com/matrix-org/bullettime/matrix/types"

	"github.com/julienschmidt/httprouter"
)

type Endpoint interface {
	Register(mux *httprouter.Router)
}

type WithStatus interface {
	Status() int
}

type OkStatus struct{}

func (s *OkStatus) Status() int {
	return 200
}

type JsonHandler func(req *http.Request, params httprouter.Params) interface{}
type JsonBodyHandler func(req *http.Request, params httprouter.Params, body interface{}) interface{}

func jsonHandler(i interface{}) httprouter.Handle {
	t := reflect.TypeOf(i)
	if reflect.Func != t.Kind() {
		panic("jsonHandler: must be a function")
	}
	if t.NumOut() != 1 {
		panic("jsonHandler: must return a single value")
	}
	argCount := t.NumIn()

	var jsonType reflect.Type
	firstParamIsParams := false
	if argCount > 0 {
		firstParamIsParams = t.In(0).String() == "httprouter.Params"
		lastParamIsParams := t.In(argCount-1).String() == "httprouter.Params"
		lastParamIsRequest := t.In(argCount-1).String() == "*http.Request"
		if !lastParamIsParams && !lastParamIsRequest {
			typ := t.In(argCount - 1)
			if typ.Kind() != reflect.Ptr {
				panic("jsonHandler: body argument must be a pointer type, was " + typ.String())
			}
			jsonType = typ.Elem()
		}
	}
	if jsonType == nil {
		if t.NumIn() > 2 {
			panic("jsonHandler: arity must be at most 2 if no body argument is preset")
		}
	} else {
		if t.NumIn() > 3 {
			panic("jsonHandler: arity must be at most 3 if body argument is preset")
		}
	}
	handlerFunc := reflect.ValueOf(i)

	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		var returnValue reflect.Value
		var args []reflect.Value
		if jsonType == nil {
			switch argCount {
			case 0:
				args = []reflect.Value{}
			case 1:
				if firstParamIsParams {
					args = []reflect.Value{reflect.ValueOf(params)}
				} else {
					args = []reflect.Value{reflect.ValueOf(req)}
				}
			case 2:
				if firstParamIsParams {
					args = []reflect.Value{reflect.ValueOf(params), reflect.ValueOf(req)}
				} else {
					args = []reflect.Value{reflect.ValueOf(req), reflect.ValueOf(params)}
				}
			}
		} else {
			body := reflect.New(jsonType)
			if err := json.NewDecoder(req.Body).Decode(body.Interface()); err != nil {
				switch err := err.(type) {
				case *json.SyntaxError:
					msg := fmt.Sprintf("error at [%d]: %s", err.Offset, err.Error())
					WriteJsonResponseWithStatus(rw, types.NotJsonError(msg))
				case *json.UnmarshalTypeError:
					msg := fmt.Sprintf("error at [%d]: expected type %s but got %s", err.Offset, err.Type, err.Value)
					WriteJsonResponseWithStatus(rw, types.BadJsonError(msg))
				default:
					WriteJsonResponseWithStatus(rw, types.BadJsonError(err.Error()))
				}
				return
			}
			switch argCount {
			case 1:
				args = []reflect.Value{body}
			case 2:
				if firstParamIsParams {
					args = []reflect.Value{reflect.ValueOf(params), body}
				} else {
					args = []reflect.Value{reflect.ValueOf(req), body}
				}
			case 3:
				if firstParamIsParams {
					args = []reflect.Value{reflect.ValueOf(params), reflect.ValueOf(req), body}
				} else {
					args = []reflect.Value{reflect.ValueOf(req), reflect.ValueOf(params), body}
				}
			}
		}
		returnValue = handlerFunc.Call(args)[0]
		res := returnValue.Interface()

		withStatus, ok := res.(WithStatus)
		if ok {
			WriteJsonResponseWithStatus(rw, withStatus)
		} else {
			WriteJsonResponse(rw, 200, res)
		}
	}
}

func WriteJsonResponse(rw http.ResponseWriter, status int, body interface{}) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	res, err := json.Marshal(body)
	if err != nil {
		rw.WriteHeader(500)
		log.Println("marshaling error: ", err)
		fmt.Fprintf(rw, "{\"errcode\":\"M_SERVER_ERROR\",\"error\":\"failed to marshal response\"}")
	} else {
		rw.WriteHeader(status)
		rw.Write(res)
	}
}

func WriteJsonResponseWithStatus(rw http.ResponseWriter, body WithStatus) {
	WriteJsonResponse(rw, body.Status(), body)
}

func readAccessToken(
	userService interfaces.UserService,
	tokenService interfaces.TokenService,
	req *http.Request,
) (ct.UserId, types.Error) {
	token := req.URL.Query().Get("access_token")
	if token == "" {
		return ct.UserId{}, types.DefaultMissingTokenError
	}
	info, err := tokenService.ParseAccessToken(token)
	if err != nil {
		return ct.UserId{}, types.DefaultUnknownTokenError
	}
	exists, err := userService.UserExists(info.UserId(), info.UserId())
	if err != nil {
		return ct.UserId{}, types.DefaultUnknownTokenError
	}
	if !exists {
		return ct.UserId{}, types.DefaultUnknownTokenError
	}
	return info.UserId(), nil
}

type urlParams struct {
	params httprouter.Params
}

func (p urlParams) user(paramPosition int, users interfaces.UserService) (ct.UserId, types.Error) {
	user, err := ct.ParseUserId(p.params[paramPosition].Value)
	if err != nil {
		return ct.UserId{}, types.BadParamError(err.Error())
	}
	if users != nil {
		exists, err := users.UserExists(user, user)
		if err != nil {
			return ct.UserId{}, err
		}
		if !exists {
			return ct.UserId{}, types.NotFoundError("user '" + user.String() + "' doesn't exist")
		}
	}
	return user, nil
}

func (p urlParams) room(paramPosition int) (ct.RoomId, types.Error) {
	room, parseErr := ct.ParseRoomId(p.params[paramPosition].Value)
	if parseErr != nil {
		return ct.RoomId{}, types.BadParamError(parseErr.Error())
	}
	return room, nil
}

type urlQuery struct {
	url.Values
}

func (q urlQuery) parseUint(name string, defaultValue uint64) (uint64, types.Error) {
	str := q.Get(name)
	if str == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, types.BadQueryError(err.Error())
	}
	return value, nil
}

func (q urlQuery) parseStreamToken(name string) (*types.StreamToken, types.Error) {
	str := q.Get(name)
	if str == "" {
		return nil, nil
	}

	token, err := types.ParseStreamToken(str)
	if err != nil {
		return nil, types.BadQueryError(err.Error())
	}
	return &token, nil
}
