package helpers

import (
	"net/url"
	"strings"
)

// The readCSV() helper reads a string value from the query string and then splits it
// into a slice on the comma character. If no matching key could be found, it returns
// the provided default value.
func ReadCSV(qs url.Values, key string, defaultValue []string) ([]string, bool) {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue, false
	}

	return strings.Split(csv, ","), true
}
