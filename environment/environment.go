package environment

import (
	"os"
	"strings"
	"sync"
)

// Environment represents the application environment
type Environment string

const (
	// Development environment
	Development Environment = "development"

	// Testing environment
	Testing Environment = "testing"

	// Staging environment
	Staging Environment = "staging"

	// Production environment
	Production Environment = "production"

	// Default environment variable name
	defaultEnvVar = "ENVIRONMENT"
)

var (
	currentEnv     Environment
	currentEnvOnce sync.Once
)

// GetEnvironment returns the current environment
func GetEnvironment() Environment {
	currentEnvOnce.Do(func() {
		// Try reading from environment variable
		envValue := strings.ToLower(os.Getenv(defaultEnvVar))

		switch envValue {
		case string(Development):
			currentEnv = Development
		case string(Testing):
			currentEnv = Testing
		case string(Staging):
			currentEnv = Staging
		case string(Production):
			currentEnv = Production
		default:
			// Default to production if unspecified for safety
			currentEnv = Production
		}
	})

	return currentEnv
}

// IsDevelopment returns true if the current environment is development
func IsDevelopment() bool {
	return GetEnvironment() == Development
}

// IsTesting returns true if the current environment is testing
func IsTesting() bool {
	return GetEnvironment() == Testing
}

// IsStaging returns true if the current environment is staging
func IsStaging() bool {
	return GetEnvironment() == Staging
}

// IsProduction returns true if the current environment is production
func IsProduction() bool {
	return GetEnvironment() == Production
}
