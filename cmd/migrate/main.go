package main

import (
	"flag"
	"log"
	"os"

	"github.com/pressly/goose/v3"

	_ "github.com/hasahmad/go-skeleton/migrations"
	_ "github.com/lib/pq"
)

func main() {
	flags := flag.NewFlagSet("goose", flag.ExitOnError)
	dir := flags.String("dir", ".", "directory with migration files")

	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) < 1 {
		flags.Usage()
		return
	}

	command := args[0]

	db, err := goose.OpenDBWithDriver("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v\n", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	arguments := []string{}
	if len(args) > 2 {
		arguments = append(arguments, args[2:]...)
	}

	if err := goose.Run(command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose %v: %v", command, err)
	}
}
