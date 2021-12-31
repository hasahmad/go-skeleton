package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	Port int
	Env  string
	DB   struct {
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  string
	}
	// rps = requests-per-second
	// enable/disable rate limiting altogether
	Limiter struct {
		RPS     float64
		Burst   int
		Enabled bool
	}
	Smtp struct {
		Host     string
		Port     int
		Username string
		Password string
		Sender   string
	}
	Cors struct {
		TrustedOrigins []string
	}
}

func InitByFlag() (Config, error) {
	var cfg Config

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.DB.DSN, "db-dsn", os.Getenv("DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.DB.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.DB.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.Limiter.RPS, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.Smtp.Host, "smtp-host", "localhost", "SMTP host")
	flag.IntVar(&cfg.Smtp.Port, "smtp-port", 25, "SMTP Port")
	flag.StringVar(&cfg.Smtp.Username, "smtp-user", "", "SMTP username")
	flag.StringVar(&cfg.Smtp.Password, "smtp-pass", "", "SMTP password")
	flag.StringVar(&cfg.Smtp.Sender, "smtp-sender", "GO Skeleton <no-reply@hasahmad.github.io>", "SMTP sender")

	cfg.Cors.TrustedOrigins = []string{}
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(s string) error {
		cfg.Cors.TrustedOrigins = strings.Split(s, " ")
		return nil
	})

	return cfg, nil
}
