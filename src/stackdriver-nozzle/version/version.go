/*
 * Copyright 2017 Google Inc.
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

package version

// Name is the string label for the stackdriver-nozzle.
const Name = "cf-stackdriver-nozzle"

var release string

func init() {
	// release is set by the linker on published builds
	if release == "" {
		release = "dev"
	}
}

// Release returns the version of the BOSH release.
func Release() string {
	return release
}

// UserAgent returns the user agent string to use for identifying connections to Stackdriver
// from the nozzle.
func UserAgent() string {
	return Name + "/" + release
}
