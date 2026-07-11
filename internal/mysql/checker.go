package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const pingTimeout = 3 * time.Second

// Status is the result of a MySQL inspection.
type Status struct {
	Available     bool
	ReadOnly      bool
	ReadOnlyKnown bool
}

// Checker performs MySQL availability and read_only checks.
type Checker struct {
	db *sql.DB
}

// Open creates a Checker using the provided DSN.
func Open(dsn string) (*Checker, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)

	return &Checker{db: db}, nil
}

// Close releases database resources.
func (c *Checker) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

// Inspect checks MySQL availability and read_only state separately.
func (c *Checker) Inspect(ctx context.Context) Status {
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	status := Status{}
	if err := c.db.PingContext(ctx); err != nil {
		return status
	}

	status.Available = true
	readOnly, err := c.readOnly(ctx)
	if err != nil {
		return status
	}

	status.ReadOnlyKnown = true
	status.ReadOnly = readOnly
	return status
}

// Check verifies that MySQL is reachable and not in read-only mode.
func (c *Checker) Check(ctx context.Context) error {
	status := c.Inspect(ctx)
	if !status.Available {
		return fmt.Errorf("mysql ping failed")
	}
	if !status.ReadOnlyKnown {
		return fmt.Errorf("query read_only failed")
	}
	if status.ReadOnly {
		return fmt.Errorf("mysql read_only is enabled")
	}
	return nil
}

func (c *Checker) readOnly(ctx context.Context) (bool, error) {
	var value string
	err := c.db.QueryRowContext(ctx, "SELECT @@global.read_only").Scan(&value)
	if err != nil {
		return false, fmt.Errorf("query read_only: %w", err)
	}
	return ParseReadOnly(value)
}

// ParseReadOnly interprets MySQL read_only values.
func ParseReadOnly(value string) (bool, error) {
	normalized := strings.TrimSpace(strings.ToUpper(value))
	switch normalized {
	case "ON", "1", "TRUE":
		return true, nil
	case "OFF", "0", "FALSE":
		return false, nil
	}

	if parsed, err := strconv.ParseBool(normalized); err == nil {
		return parsed, nil
	}
	if asInt, err := strconv.Atoi(normalized); err == nil {
		return asInt != 0, nil
	}

	return false, fmt.Errorf("unknown read_only value %q", value)
}
