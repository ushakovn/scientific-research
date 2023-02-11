package db

import (
  "database/sql"
  "fmt"
  "os"
  "scientific-research/pkg/utils/retries"

  _ "github.com/mattn/go-sqlite3"
  log "github.com/sirupsen/logrus"
)

const (
  driverName = "sqlite3"
  dataSource = "DB"
)

type Client interface {
  Exec(query string, args ...any) error
  Query(query string, args ...any) (*sql.Rows, error)
  QueryRow(query string, args ...any) *sql.Row
  Open() error
  Close() error
}

type sqlite struct {
  db *sql.DB
}

func NewSqliteClient() Client {
  return &sqlite{}
}

func (c *sqlite) Open() error {
  var (
    db  *sql.DB
    ds  string
    err error
  )
  ds, err = getDataSourcePath()
  if err != nil {
    return err
  }
  connStr := fmt.Sprint(ds, "?cache=shared")

  err = retries.DoWithRetry(func() error {
    db, err = sql.Open(driverName, connStr)
    return err
  })
  if err != nil {
    return fmt.Errorf("cannot open connection to db: %v", err)
  }
  c.db = db

  return db.Ping()
}

func (c *sqlite) Close() error {
  err := retries.DoWithRetry(func() error {
    return c.db.Close()
  })
  if err != nil {
    log.Fatalf("cannot close db connection: %v", err)
  }
  return nil
}

func (c *sqlite) Query(query string, args ...any) (*sql.Rows, error) {
  s, err := c.db.Prepare(query)
  if err != nil {
    return nil, err
  }
  return s.Query(args...)
}

func (c *sqlite) QueryRow(query string, args ...any) *sql.Row {
  s, err := c.db.Prepare(query)
  if err != nil {
    log.Fatal(err)
  }
  return s.QueryRow(args...)
}

func (c *sqlite) Exec(query string, args ...any) error {
  s, err := c.db.Prepare(query)
  if err != nil {
    return err
  }
  _, err = s.Exec(args...)
  return err
}

func getDataSourcePath() (string, error) {
  path := os.Getenv(dataSource)
  if path == "" {
    return "", fmt.Errorf("data source path not found")
  }
  return path, nil
}
