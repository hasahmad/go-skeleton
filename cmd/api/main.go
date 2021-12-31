package main

import (
	"expvar"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hasahmad/go-skeleton/internal"
	"github.com/hasahmad/go-skeleton/internal/config"
	"github.com/hasahmad/go-skeleton/internal/db"

	log "github.com/sirupsen/logrus"

	// Import the pq driver so that it can register itself with the database/sql
	// package.
	_ "github.com/lib/pq"
)

var (
	version   string
	buildTime string
)

func main() {
	cfg, err := config.InitByFlag()
	if err != nil {
		fmt.Errorf(err.Error())
		os.Exit(0)
	}

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	// If the version flag value is true, then print out the version number and
	// immediately exit.
	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}

	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{})

	db, err := db.OpenDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()

	logger.Info("database connection pool established")

	expvar.NewString("version").Set(version)

	// Publish the number of active goroutines.
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	// Publish the database connection pool statistics.
	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))

	// Publish the current Unix timestamp.
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	app := internal.New(logger, cfg, db, sync.WaitGroup{})
	err = app.Serve()
	if err != nil {
		logger.Fatal(err)
	}
}
