package helpers

import (
	"net/url"
	"strconv"

	"github.com/hasahmad/go-skeleton/internal/validator"
)

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func ReadInt(qs url.Values, key string, defaultValue int, v *validator.Validator) (int, bool) {
	s := qs.Get(key)

	if s == "" {
		return defaultValue, false
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue, true
	}

	return i, true
}
