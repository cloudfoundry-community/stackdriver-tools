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

package nozzle

import (
	"fmt"
	"regexp"
)

const (
	prefixRegex = "k|M|G|T|P|E|Z|Y|m|μ|n|p|f|a|z|y|Ki|Mi|Gi|Ti"
	unitRegex   = "b|B|s|M|h|d"
)

/*
Expression = Component { "." Component } { "/" Component } ;

Component = [ PREFIX ] UNIT [ Annotation ]
          | Annotation
          | "1"
          ;

Annotation = "{" NAME "}" ;
*/

// UnitParser parses Sonde metric unit strings to their Stackdriver equivalents.
type UnitParser interface {

	// Parse converts a Sonde metric unit to a Stackdriver metric unit.
	Parse(string) string
}

// NewUnitParser constructs a new UnitParser.
func NewUnitParser() UnitParser {
	componentRegex := regexp.MustCompile(fmt.Sprintf("^(%s)?(%s)$", prefixRegex, unitRegex))
	annotationRegex := regexp.MustCompile("[{}]")
	expressionRegex := regexp.MustCompile("^([^/]*)(/([^/]*))?$")

	return &unitParser{
		componentRegex:  componentRegex,
		annotationRegex: annotationRegex,
		expressionRegex: expressionRegex,
	}
}

type unitParser struct {
	componentRegex  *regexp.Regexp
	annotationRegex *regexp.Regexp
	expressionRegex *regexp.Regexp
}

func (up *unitParser) Parse(input string) string {
	matches := up.expressionRegex.FindStringSubmatch(input)
	if matches == nil {
		return up.annotate(input)
	}

	numerator := up.parseComponent(matches[1])
	if matches[2] == "" {
		return numerator
	}

	denominator := up.parseComponent(matches[3])
	return fmt.Sprintf("%s/%s", numerator, denominator)
}

func (up *unitParser) parseComponent(input string) string {
	matches := up.componentRegex.FindStringSubmatch(input)
	if matches == nil {
		return up.annotate(input)
	}

	prefix := prefixLookup(matches[1])
	unit := unitLookup(matches[2])
	return fmt.Sprintf("%s%s", prefix, unit)
}

// Not sure if this is faster than a map or not - if we
// are looking for perf gains, maybe do some benchmarking
// around here.
func unitLookup(unit string) string {
	switch unit {
	case "b":
		return "bit"
	case "B":
		return "By"
	case "M":
		return "min"
	default:
		return unit
	}
}

func prefixLookup(prefix string) string {
	if prefix == "μ" {
		return "u"
	}
	return prefix
}

func (up *unitParser) annotate(input string) string {
	return fmt.Sprintf("{%s}", up.annotationRegex.ReplaceAllString(input, ""))
}
