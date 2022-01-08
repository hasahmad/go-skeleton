package handlers

import (
	"fmt"
	"net/http"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
)

func (h Handlers) HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := helpers.Envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": h.cfg.Env,
			"port":        fmt.Sprintf("%d", h.cfg.Port),
		},
	}

	err := helpers.WriteJSON(w, http.StatusOK, data, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}
