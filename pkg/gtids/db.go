package gtids

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ReadMyCnf returns the database credentials, preferring the MYSQL_USER and
// MYSQL_PASSWORD environment variables and falling back to ~/.my.cnf.
func ReadMyCnf() (user, password string, err error) {
	user = os.Getenv("MYSQL_USER")
	password = os.Getenv("MYSQL_PASSWORD")
	if user != "" && password != "" {
		return user, password, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to locate home directory: %w", err)
	}
	file, err := os.ReadFile(filepath.Join(home, ".my.cnf"))
	if err != nil {
		return "", "", fmt.Errorf("failed to read ~/.my.cnf: %w", err)
	}
	for _, line := range strings.Split(string(file), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "user":
			if user == "" {
				user = value
			}
		case "password":
			if password == "" {
				password = value
			}
		}
	}
	if user == "" || password == "" {
		return "", "", fmt.Errorf("user and password not found in ~/.my.cnf (or set MYSQL_USER and MYSQL_PASSWORD)")
	}
	return user, password, nil
}

// sleepCtx sleeps for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// connectWithRetry attempts to connect to a database with retry logic for transient failures
func connectWithRetry(ctx context.Context, dsn string, maxRetries int) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.PingContext(ctx)
			if err == nil {
				return db, nil
			}
			db.Close()
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
		}
		if serr := sleepCtx(ctx, time.Duration(attempt)*100*time.Millisecond); serr != nil {
			return nil, serr
		}
	}

	return nil, fmt.Errorf("unexpected error in connection retry logic")
}

// ConnectToDatabases connects to the source and target databases with retry logic
func ConnectToDatabases(ctx context.Context, sourceHost, sourcePort, targetHost, targetPort string) (db1, db2 *sql.DB, err error) {
	user, password, err := ReadMyCnf()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read MySQL credentials: %w", err)
	}

	// Connection/read/write timeouts so a hung server can't hang the tool forever.
	const dsnOptions = "?timeout=10s&readTimeout=1m&writeTimeout=1m"
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql%s", user, password, sourceHost, sourcePort, dsnOptions)
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql%s", user, password, targetHost, targetPort, dsnOptions)

	// Connect to source database
	db1, err = connectWithRetry(ctx, sourceDSN, 3)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to source database %s:%s: %w", sourceHost, sourcePort, err)
	}

	// Connect to target database
	db2, err = connectWithRetry(ctx, targetDSN, 3)
	if err != nil {
		db1.Close() // Clean up source connection
		return nil, nil, fmt.Errorf("failed to connect to target database %s:%s: %w", targetHost, targetPort, err)
	}

	return db1, db2, nil
}

// isRetryableError reports whether an error is transient enough to retry:
// broken/invalid connections, network errors, lock wait timeouts, deadlocks.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1205, 1213: // ER_LOCK_WAIT_TIMEOUT, ER_LOCK_DEADLOCK
			return true
		}
	}
	return false
}

// retryDatabaseOperation retries a transient-failing database operation with backoff
func retryDatabaseOperation(ctx context.Context, operation func() error, maxRetries int) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		if !isRetryableError(err) {
			return err
		}
		if attempt == maxRetries {
			return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, err)
		}
		if serr := sleepCtx(ctx, time.Duration(attempt+1)*100*time.Millisecond); serr != nil {
			return serr
		}
	}
	return err
}
