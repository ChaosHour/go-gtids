package gtids

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Unit tests for the fix orchestration using a mocked database, so the
// statement sequences are verified without a live MySQL instance.

func TestApplyGtidEntries_StatementSequence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	entries := []string{
		"1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1",
		"1d1fff5a-c9bc-11ed-9c19-02a36d996b94:2",
	}

	for _, entry := range entries {
		mock.ExpectExec("SET GTID_NEXT='" + entry + "'").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("BEGIN").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("COMMIT").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectExec("SET GTID_NEXT='AUTOMATIC'").WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Close()

	if err := applyGtidEntries(ctx, conn, entries, "test"); err != nil {
		t.Fatalf("applyGtidEntries failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestApplyGtidEntries_RejectsInvalidEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Only the GTID_NEXT reset should run — the invalid entry must be
	// rejected before any statement is sent.
	mock.ExpectExec("SET GTID_NEXT='AUTOMATIC'").WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Close()

	injections := []string{
		"not-a-uuid:1",
		"1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1'; DROP TABLE users; --",
		"1d1fff5a-c9bc-11ed-9c19-02a36d996b94:",
		"",
	}
	for _, bad := range injections {
		if err := applyGtidEntries(ctx, conn, []string{bad}, "test"); err == nil {
			t.Errorf("expected error for invalid entry %q, got nil", bad)
		}
	}
}

func TestApplyGtidEntries_ResetsGtidNextOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	entry := "1d1fff5a-c9bc-11ed-9c19-02a36d996b94:1"
	mock.ExpectExec("SET GTID_NEXT='" + entry + "'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("BEGIN").WillReturnError(errors.New("server has gone away"))
	// Even after the failure, GTID_NEXT must be reset to AUTOMATIC.
	mock.ExpectExec("SET GTID_NEXT='AUTOMATIC'").WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Close()

	if err := applyGtidEntries(ctx, conn, []string{entry}, "test"); err == nil {
		t.Fatal("expected error from failed BEGIN, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("GTID_NEXT was not reset after failure: %v", err)
	}
}

func TestGetBinaryLogInfo_FallsBackToShowMasterStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// MySQL < 8.2 doesn't know SHOW BINARY LOG STATUS.
	mock.ExpectQuery("SHOW BINARY LOG STATUS").WillReturnError(errors.New("Error 1064 (42000): syntax error"))
	rows := sqlmock.NewRows([]string{"File", "Position", "Binlog_Do_DB", "Binlog_Ignore_DB", "Executed_Gtid_Set"}).
		AddRow("binlog.000042", 1234, "", "", "uuid:1-10")
	mock.ExpectQuery("SHOW MASTER STATUS").WillReturnRows(rows)

	logName, err := getBinaryLogInfo(context.Background(), db)
	if err != nil {
		t.Fatalf("getBinaryLogInfo failed: %v", err)
	}
	if logName != "binlog.000042" {
		t.Errorf("expected binlog.000042, got %q", logName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetBinaryLogInfo_PrefersShowBinaryLogStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// MySQL 8.4+ answers the new statement; the old one must not be issued.
	rows := sqlmock.NewRows([]string{"File", "Position", "Binlog_Do_DB", "Binlog_Ignore_DB", "Executed_Gtid_Set"}).
		AddRow("binlog.000007", 999, "", "", "uuid:1-10")
	mock.ExpectQuery("SHOW BINARY LOG STATUS").WillReturnRows(rows)

	logName, err := getBinaryLogInfo(context.Background(), db)
	if err != nil {
		t.Fatalf("getBinaryLogInfo failed: %v", err)
	}
	if logName != "binlog.000007" {
		t.Errorf("expected binlog.000007, got %q", logName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDetermineReplicationCommands_Mock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT VERSION").
		WillReturnRows(sqlmock.NewRows([]string{"VERSION()"}).AddRow("8.4.2"))

	stop, start, status, err := determineReplicationCommands(context.Background(), db)
	if err != nil {
		t.Fatalf("determineReplicationCommands failed: %v", err)
	}
	if stop != "STOP REPLICA" || start != "START REPLICA" || status != "SHOW REPLICA STATUS" {
		t.Errorf("expected REPLICA commands for 8.4.2, got %q / %q / %q", stop, start, status)
	}
}

func TestVerifyReplicationStatus_ReplicaColumnNames(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// MySQL 8.0.22+ column names (Source/Replica instead of Master/Slave).
	rows := sqlmock.NewRows([]string{
		"Source_Host", "Replica_IO_Running", "Replica_SQL_Running",
		"Seconds_Behind_Source", "Replica_SQL_Running_State",
		"Retrieved_Gtid_Set", "Executed_Gtid_Set",
	}).AddRow("10.0.0.1", "Yes", "Yes", 0, "waiting", "uuid:1-5", "uuid:1-5")
	mock.ExpectQuery("SHOW REPLICA STATUS").WillReturnRows(rows)

	err = verifyReplicationStatus(context.Background(), db, "SHOW REPLICA STATUS", "replica", "")
	if err != nil {
		t.Fatalf("verifyReplicationStatus failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRetryDatabaseOperation_NonRetryableFailsFast(t *testing.T) {
	calls := 0
	err := retryDatabaseOperation(context.Background(), func() error {
		calls++
		return errors.New("Error 1064 (42000): You have an error in your SQL syntax")
	}, 3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("non-retryable error should not be retried: got %d calls", calls)
	}
}

func TestRetryDatabaseOperation_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A retryable error with a cancelled context must abort during backoff.
	err := retryDatabaseOperation(ctx, func() error {
		return &timeoutError{}
	}, 3)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// timeoutError implements net.Error to exercise the retryable path.
type timeoutError struct{}

func (*timeoutError) Error() string   { return "i/o timeout" }
func (*timeoutError) Timeout() bool   { return true }
func (*timeoutError) Temporary() bool { return true }
