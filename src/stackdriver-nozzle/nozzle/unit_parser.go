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

type UnitParser interface {
	Parse(string) string
}

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