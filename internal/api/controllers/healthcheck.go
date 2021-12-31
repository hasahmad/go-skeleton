package controllers

import (
	"fmt"
	"net/http"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
)

func (ctrl Controllers) HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := helpers.Envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": ctrl.cfg.Env,
			"port":        fmt.Sprintf("%d", ctrl.cfg.Port),
		},
	}

	err := helpers.WriteJSON(w, http.StatusOK, data, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}
