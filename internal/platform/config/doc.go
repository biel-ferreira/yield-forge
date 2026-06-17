// Package config loads typed application configuration from the environment.
//
// It exposes config.Load(), which reads environment variables (optionally
// seeded from a local .env file in development), applies defaults, validates
// required fields, and returns a typed Config or a descriptive error.
//
// Implemented in SPEC-001 phase 2.
package config
