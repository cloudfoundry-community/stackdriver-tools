package nozzle

import (
	"fmt"
	"regexp"
	"strings"
)

func UnitParser(fhUnit string) string {
	fhPrefixes := []string{
		"k",
		"M",
		"G",
		"T",
		"P",
		"E",
		"Z",
		"Y",
		"m",
		"μ",
		"n",
		"p",
		"f",
		"a",
		"z",
		"y",
		"Ki",
		"Mi",
		"Gi",
		"Ti",
	}
	fhUnits := []string{"b", "B", "s", "M", "h", "d"}
	prefixRegex := strings.Join(fhPrefixes, "|")
	unitRegex := strings.Join(fhUnits, "|")
	componentRegex := regexp.MustCompile(fmt.Sprintf("^(%s)?(%s)$", prefixRegex, unitRegex))

	matches := componentRegex.FindStringSubmatch(fhUnit)
	if matches == nil {
		annotationRegex := regexp.MustCompile("[{}]")
		return fmt.Sprintf("{%s}", annotationRegex.ReplaceAllString(fhUnit, ""))
	}

	prefix := matches[1]
	unit := matches[2]

	unitLookup := map[string]string{
		"b": "bit",
		"B": "By",
		"M": "min",
	}

	if prefix == "μ" {
		prefix = "u"
	}

	if lookup, ok := unitLookup[unit]; ok {
		unit = lookup
	}

	return fmt.Sprintf("%s%s", prefix, unit)
}
