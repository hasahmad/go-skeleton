package helpers

import "net/url"

// readBool Read Boolean value from string and retutns the value and
// if the value exists in the querystring (!= "")
func ReadBool(qs url.Values, key string, defaultValue bool) (bool, bool) {
	s := qs.Get(key)

	if s == "" {
		return defaultValue, false
	}

	if s == "true" || s == "t" || s == "y" || s == "1" {
		return true, true
	} else if s == "false" || s == "f" || s == "n" || s == "0" {
		return false, true
	}

	return defaultValue, true
}
