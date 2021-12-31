package helpers

import "net/url"

// The readString() helper returns a string value from the query string, or the provided
// default value if no matching key could be found.
func ReadString(qs url.Values, key string, defaultValue string) (string, bool) {
	exists := qs.Has(key)
	s := qs.Get(key)

	if s == "" {
		return defaultValue, exists
	}

	return s, exists
}
