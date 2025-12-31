package gtids

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ReadMyCnf reads the ~/.my.cnf file to get the database credentials
func ReadMyCnf() (user, password string, err error) {
	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		return "", "", fmt.Errorf("failed to read ~/.my.cnf: %w", err)
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "user=") {
			user = strings.TrimSpace(line[5:])
		}
		if strings.HasPrefix(line, "password=") {
			password = strings.TrimSpace(line[9:])
		}
	}
	if user == "" || password == "" {
		return "", "", fmt.Errorf("user and password not found in ~/.my.cnf")
	}
	return user, password, nil
}

// connectWithRetry attempts to connect to a database with retry logic for transient failures
func connectWithRetry(dsn string, maxRetries int) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to open database connection after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		err = db.Ping()
		if err != nil {
			db.Close()
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		// Success
		return db, nil
	}

	return nil, fmt.Errorf("unexpected error in connection retry logic")
}

// ConnectToDatabases connects to the source and target databases with retry logic
func ConnectToDatabases(sourceHost, sourcePort, targetHost, targetPort string) (db1, db2 *sql.DB, err error) {
	user, password, err := ReadMyCnf()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read MySQL credentials: %w", err)
	}

	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql", user, password, sourceHost, sourcePort)
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql", user, password, targetHost, targetPort)

	// Connect to source database
	db1, err = connectWithRetry(sourceDSN, 3)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to source database %s:%s: %w", sourceHost, sourcePort, err)
	}

	// Connect to target database
	db2, err = connectWithRetry(targetDSN, 3)
	if err != nil {
		db1.Close() // Clean up source connection
		return nil, nil, fmt.Errorf("failed to connect to target database %s:%s: %w", targetHost, targetPort, err)
	}

	return db1, db2, nil
}

// retryDatabaseOperation retries a database operation with exponential backoff
func retryDatabaseOperation(operation func() error, maxRetries int) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if the error is retryable (connection-related errors)
		if strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "connection refused") {
			if attempt == maxRetries {
				return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, err)
			}
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		// For non-retryable errors, return immediately
		return err
	}
	return err
}
