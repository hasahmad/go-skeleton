package helpers

import (
	"net/url"
	"strconv"

	"github.com/hasahmad/go-skeleton/internal/validator"
)

func ReadFloat(qs url.Values, key string, defaultValue float64, v *validator.Validator) (float64, bool) {
	s := qs.Get(key)

	if s == "" {
		return defaultValue, false
	}

	val, err := strconv.ParseFloat(s, 32)
	if err != nil {
		v.AddError(key, "must be a float value")
		return defaultValue, true
	}

	return val, true
}
