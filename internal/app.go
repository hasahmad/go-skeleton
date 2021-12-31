package internal

import (
	"sync"

	"github.com/hasahmad/go-skeleton/internal/api/controllers"
	apierrors "github.com/hasahmad/go-skeleton/internal/api/errors"
	"github.com/hasahmad/go-skeleton/internal/api/middlewares"
	"github.com/hasahmad/go-skeleton/internal/config"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/mailer"
	"github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

type Application struct {
	logger      *logrus.Logger
	cfg         config.Config
	errors      apierrors.ErrorResponses
	mailer      mailer.Mailer
	models      data.Models
	controllers controllers.Controllers
	middlewares middlewares.Middlewares
	wg          sync.WaitGroup
}

func NewApplication(
	logger *logrus.Logger,
	cfg config.Config,
	db *sqlx.DB,
	wg sync.WaitGroup,
) *Application {
	errorReps := apierrors.New(logger)
	models := data.NewModels(db)
	mailer := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Sender)
	return &Application{
		logger:      logger,
		cfg:         cfg,
		errors:      errorReps,
		wg:          wg,
		mailer:      mailer,
		models:      models,
		middlewares: middlewares.New(logger, cfg, errorReps, models),
		controllers: controllers.New(logger, cfg, errorReps, models, mailer, wg),
	}
}
