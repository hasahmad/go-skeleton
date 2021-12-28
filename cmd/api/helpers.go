package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hasahmad/go-skeleton/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]interface{}

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) readUUIDParam(r *http.Request) (uuid.UUID, error) {
	params := httprouter.ParamsFromContext(r.Context())

	var id uuid.UUID

	if params.ByName("id") == "" {
		return id, errors.New("invalid id parameter")
	}

	err := id.Scan(params.ByName("id"))
	if err != nil || id.String() == "" {
		return id, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) readUUIDParamByKey(r *http.Request, key string) (uuid.UUID, error) {
	params := httprouter.ParamsFromContext(r.Context())

	var uid uuid.UUID

	if key == "" {
		key = "id"
	}

	if params.ByName(key) == "" {
		return uid, errors.New("invalid id parameter")
	}

	err := uid.Scan(params.ByName(key))
	if err != nil || uid.String() == "" {
		return uid, errors.New("invalid id parameter")
	}

	return uid, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// The readString() helper returns a string value from the query string, or the provided
// default value if no matching key could be found.
func (app *application) readString(qs url.Values, key string, defaultValue string) (string, bool) {
	exists := qs.Has(key)
	s := qs.Get(key)

	if s == "" {
		return defaultValue, exists
	}

	return s, exists
}

// readBool Read Boolean value from string and retutns the value and
// if the value exists in the querystring (!= "")
func (app *application) readBool(qs url.Values, key string, defaultValue bool) (bool, bool) {
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

// The readCSV() helper reads a string value from the query string and then splits it
// into a slice on the comma character. If no matching key could be found, it returns
// the provided default value.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) ([]string, bool) {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue, false
	}

	return strings.Split(csv, ","), true
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) (int, bool) {
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

func (app *application) readFloat(qs url.Values, key string, defaultValue float64, v *validator.Validator) (float64, bool) {
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

// The background() helper accepts an arbitrary function as a parameter.
func (app *application) background(fn func()) {
	app.wg.Add(1)

	// Launch a background goroutine.
	go func() {

		// decrement the WaitGroup counter before the goroutine returns.
		defer app.wg.Done()

		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Errorf("%s", err))
			}
		}()

		// Execute the arbitrary function that we passed as the parameter.
		fn()
	}()
}
