package masterenv

import (
	"os"
	"strings"
)

const envName = "MYSQL_MASTER"

// Enabled reports whether MYSQL_MASTER indicates this node is the MySQL master.
func Enabled() bool {
	return Parse(os.Getenv(envName))
}

// Parse interprets MYSQL_MASTER values. Accepted truthy values: "1", "true" (case-insensitive).
func Parse(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true":
		return true
	default:
		return false
	}
}
