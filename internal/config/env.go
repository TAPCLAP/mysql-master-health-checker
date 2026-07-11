package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func resolveString(envKey string, flagValue string, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if flagValue != "" {
		return flagValue
	}
	return fallback
}

func resolveBool(envKey string, flagValue bool, fallback bool) bool {
	if _, ok := os.LookupEnv(envKey); ok {
		return parseBoolEnv(envKey, fallback)
	}
	return flagValue
}

func resolveDuration(envKey string, flagValue time.Duration, fallback time.Duration) time.Duration {
	if _, ok := os.LookupEnv(envKey); ok {
		return parseDurationEnv(envKey, fallback)
	}
	return flagValue
}
