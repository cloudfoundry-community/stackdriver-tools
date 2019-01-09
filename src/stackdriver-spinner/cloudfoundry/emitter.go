/*
 * Copyright 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Emitter struct {
	writer io.Writer
	count  int
	delay  time.Duration
}

type Payload struct {
	Timestamp string `json:"timestamp"`
	GUID      string `json:"guid"`
	Count     int    `json:"count"`
}

func (e *Emitter) Emit(guid string) (int, error) {
	for i := 0; i < e.count; i++ {
		pl := Payload{
			Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000-07:00"),
			GUID:      guid,
			Count:     i + 1,
		}

		msg, err := json.Marshal(pl)
		if err != nil {
			return i, err
		}

		_, err = fmt.Fprintf(e.writer, string(msg)+"\n")
		time.Sleep(e.delay)
		if err != nil {
			return i, err
		}
	}
	return e.count, nil
}

func NewEmitter(writer io.Writer, count int, delay time.Duration) *Emitter {
	return &Emitter{writer, count, delay}
}
