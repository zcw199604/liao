package database

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMySQLDialect_DuplicateDetectors(t *testing.T) {
	d := MySQLDialect{}

	if !d.IsDuplicateKey(&mysql.MySQLError{Number: 1062, Message: "dup"}) {
		t.Fatalf("IsDuplicateKey should detect ER_DUP_ENTRY")
	}
	if d.IsDuplicateKey(&mysql.MySQLError{Number: 9999, Message: "other"}) {
		t.Fatalf("IsDuplicateKey should not match other codes")
	}

	if !d.IsDuplicateColumn(&mysql.MySQLError{Number: 1060, Message: "dup col"}) {
		t.Fatalf("IsDuplicateColumn should detect ER_DUP_FIELDNAME")
	}
	if d.IsDuplicateColumn(&mysql.MySQLError{Number: 9999, Message: "other"}) {
		t.Fatalf("IsDuplicateColumn should not match other codes")
	}

	if !d.IsDuplicateIndex(&mysql.MySQLError{Number: 1061, Message: "dup idx"}) {
		t.Fatalf("IsDuplicateIndex should detect ER_DUP_KEYNAME")
	}
	if d.IsDuplicateIndex(&mysql.MySQLError{Number: 9999, Message: "other"}) {
		t.Fatalf("IsDuplicateIndex should not match other codes")
	}
}

func TestPostgresDialect_DuplicateDetectors(t *testing.T) {
	d := PostgresDialect{}

	if !d.IsDuplicateKey(&pgconn.PgError{Code: "23505"}) {
		t.Fatalf("IsDuplicateKey should detect unique_violation")
	}
	if d.IsDuplicateKey(&pgconn.PgError{Code: "99999"}) {
		t.Fatalf("IsDuplicateKey should not match other codes")
	}

	if !d.IsDuplicateColumn(&pgconn.PgError{Code: "42701"}) {
		t.Fatalf("IsDuplicateColumn should detect duplicate_column")
	}
	if d.IsDuplicateColumn(&pgconn.PgError{Code: "99999"}) {
		t.Fatalf("IsDuplicateColumn should not match other codes")
	}

	if !d.IsDuplicateIndex(&pgconn.PgError{Code: "42P07"}) {
		t.Fatalf("IsDuplicateIndex should accept duplicate_table for already-exists cases")
	}
	if !d.IsDuplicateIndex(&pgconn.PgError{Code: "42710"}) {
		t.Fatalf("IsDuplicateIndex should accept duplicate_object for already-exists cases")
	}
	if d.IsDuplicateIndex(&pgconn.PgError{Code: "99999"}) {
		t.Fatalf("IsDuplicateIndex should not match other codes")
	}
}
