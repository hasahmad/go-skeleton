package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// shutdownError channel will be used to receive any errors returned
	// by the graceful Shutdown() function.
	shutdownError := make(chan error)

	go func() {
		// Create a quit channel which carries os.Signal values.
		quit := make(chan os.Signal, 1)

		// Use signal.Notify() to listen for incoming SIGINT and SIGTERM signals and
		// relay them to the quit channel.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read the signal from the quit channel. This code will block until a signal is
		// received.
		s := <-quit

		app.logger.WithFields(log.Fields{
			"signal": s.String(),
		}).Info("caught signal")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Call Shutdown() on our server with context we just made.
		// Shutdown() will return nil if the graceful shutdown was successful, or an
		// error (which may happen because of a problem closing the listeners, or
		// because the shutdown didn't complete before the 5-second context deadline is
		// hit). We relay this return value to the shutdownError channel if has error
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.WithFields(log.Fields{"addr": srv.Addr}).Info("completing background tasks")
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.WithFields(log.Fields{
		"addr": srv.Addr,
		"env":  app.config.env,
	}).Info("starting server")

	// Calling Shutdown() on our server will cause ListenAndServe() to immediately
	// return a http.ErrServerClosed error. So if we see this error, it is actually a
	// good thing and an indication that the graceful shutdown has started. So we check
	// specifically for this, only returning the error if it is NOT http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the
	// shutdownError channel. If return value is an error, we know that there was a
	// problem with the graceful shutdown and we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// we know that the graceful shutdown completed successfully and we
	// log a "stopped server" message.
	app.logger.WithFields(log.Fields{
		"addr": srv.Addr,
	}).Info("stopped server")

	return nil
}
