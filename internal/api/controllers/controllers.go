package controllers

import (
	"sync"

	apierrors "github.com/hasahmad/go-skeleton/internal/api/errors"
	"github.com/hasahmad/go-skeleton/internal/config"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/mailer"
	"github.com/sirupsen/logrus"
)

type Controllers struct {
	logger *logrus.Logger
	cfg    config.Config
	errors apierrors.ErrorResponses
	mailer mailer.Mailer
	models data.Models
	wg     sync.WaitGroup
}

func New(
	logger *logrus.Logger,
	cfg config.Config,
	errors apierrors.ErrorResponses,
	models data.Models,
	mailer mailer.Mailer,
	wg sync.WaitGroup,
) Controllers {
	return Controllers{
		logger: logger,
		cfg:    cfg,
		errors: errors,
		models: models,
		mailer: mailer,
		wg:     wg,
	}
}
