package helpers

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func ReadUUIDParam(r *http.Request) (uuid.UUID, error) {
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

func ReadUUIDParamByKey(r *http.Request, key string) (uuid.UUID, error) {
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
