# go-skeleton

Learning Go

## Directory Structure

- cmd
  - api - setup and start api
  - migrate - migrations

- internal
  - api - all handler, middlewares and utils
  - config - application config
  - data - db related (models)
  - mailer
  - validator
  - app.go
  - routes.go - all routes
  - server.go - create and start http server

- migrations


## Libs
Check go.mod

- `goqu` Query Builder (github.com/doug-martin/goqu)

- Gomail (github.com/go-mail/mail)

- UUID (github.com/google/uuid)

- `sqlx` - `database/sql` with more features (github.com/jmoiron/sqlx)

- httprouter (github.com/julienschmidt/httprouter)

- pq - Postgres driver (github.com/lib/pq)

- goose - for migrations (github.com/pressly/goose)

- logs (github.com/sirupsen/logrus)

- realip (github.com/tomasen/realip)

- structs fields for `null` DB values (gopkg.in/guregu/null)

- ...

