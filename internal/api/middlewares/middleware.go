package middlewares

import (
	apierrors "github.com/hasahmad/go-skeleton/internal/api/errors"
	"github.com/hasahmad/go-skeleton/internal/config"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/sirupsen/logrus"
)

type Middlewares struct {
	logger *logrus.Logger
	cfg    config.Config
	errors apierrors.ErrorResponses
	models data.Models
}

func New(logger *logrus.Logger, cfg config.Config, errors apierrors.ErrorResponses, models data.Models) Middlewares {
	return Middlewares{
		logger: logger,
		cfg:    cfg,
		errors: errors,
		models: models,
	}
}
