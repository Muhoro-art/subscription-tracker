package service

import (
	"fmt"
	"strconv"
	"strings"
)

// parseDate parses "MM-YYYY" into year and month.
func parseDate(s string) (year, month int, err error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid date format: %s", s)
	}
	month, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month in date: %s", s)
	}
	year, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year in date: %s", s)
	}
	return year, month, nil
}

// dateToInt converts year/month to comparable YYYYMM integer.
func dateToInt(year, month int) int {
	return year*100 + month
}

// overlapMonths calculates the number of overlapping months between a subscription
// period [subStart, subEnd] and a query period [qStart, qEnd].
// If subEnd is nil, the subscription is ongoing (treated as active through qEnd).
func overlapMonths(subStart string, subEnd *string, qStart, qEnd string) (int, error) {
	sY, sM, err := parseDate(subStart)
	if err != nil {
		return 0, err
	}
	qsY, qsM, err := parseDate(qStart)
	if err != nil {
		return 0, err
	}
	qeY, qeM, err := parseDate(qEnd)
	if err != nil {
		return 0, err
	}

	subStartInt := dateToInt(sY, sM)
	qStartInt := dateToInt(qsY, qsM)
	qEndInt := dateToInt(qeY, qeM)

	var subEndInt int
	if subEnd != nil {
		eY, eM, err := parseDate(*subEnd)
		if err != nil {
			return 0, err
		}
		subEndInt = dateToInt(eY, eM)
	} else {
		subEndInt = qEndInt // ongoing sub treated as active through query end
	}

	overlapStart := max(subStartInt, qStartInt)
	overlapEnd := min(subEndInt, qEndInt)

	if overlapStart > overlapEnd {
		return 0, nil
	}

	osY, osM := overlapStart/100, overlapStart%100
	oeY, oeM := overlapEnd/100, overlapEnd%100

	months := (oeY-osY)*12 + (oeM - osM) + 1
	return months, nil
}

// isDateBefore returns true if date a (MM-YYYY) is strictly before date b.
func isDateBefore(a, b string) (bool, error) {
	aY, aM, err := parseDate(a)
	if err != nil {
		return false, err
	}
	bY, bM, err := parseDate(b)
	if err != nil {
		return false, err
	}
	return dateToInt(aY, aM) < dateToInt(bY, bM), nil
}
