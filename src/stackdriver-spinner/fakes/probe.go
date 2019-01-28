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

package fakes

import "time"

// LosslessProbe never loses any logs.
type LosslessProbe struct {
}

// Find finds all logs.
func (m *LosslessProbe) Find(start time.Time, needle string, count int) (int, error) {
	return count, nil
}

// ConfigurableProbe is a logging probe which can return a configurable number of logs.
type ConfigurableProbe struct {
	FindFunc func(time.Time, string, int) (int, error)
}

// Find returns whatever FindFunc determines.
func (m *ConfigurableProbe) Find(start time.Time, needle string, count int) (int, error) {
	return m.FindFunc(start, needle, count)
}
