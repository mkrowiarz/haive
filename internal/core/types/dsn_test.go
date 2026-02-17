package types

import (
	"strings"
	"testing"
)

func TestDSNStringMySQLRoundTrip(t *testing.T) {
	dsn := &DSN{
		Engine:   "mysql",
		User:     "root",
		Password: "secret",
		Host:     "localhost",
		Port:     "3306",
		Database: "myapp",
	}

	result := dsn.String()

	expected := "mysql://root:secret@localhost:3306/myapp"
	if result != expected {
		t.Errorf("DSN.String() = %q, want %q", result, expected)
	}
}

func TestDSNStringPostgreSQLRoundTrip(t *testing.T) {
	dsn := &DSN{
		Engine:   "postgres",
		User:     "postgres",
		Password: "password",
		Host:     "database",
		Port:     "5432",
		Database: "myapp",
	}

	result := dsn.String()

	expected := "postgres://postgres:password@database:5432/myapp"
	if result != expected {
		t.Errorf("DSN.String() = %q, want %q", result, expected)
	}
}

func TestDSNStringPostgresqlSchemeNormalization(t *testing.T) {
	// postgresql scheme should be normalized to postgres
	dsn := &DSN{
		Engine:   "postgresql",
		User:     "user",
		Password: "pass",
		Host:     "host",
		Port:     "5432",
		Database: "db",
	}

	result := dsn.String()

	// Should use "postgres" scheme, not "postgresql"
	if strings.HasPrefix(result, "postgresql://") {
		t.Errorf("DSN.String() should normalize postgresql to postgres, got %q", result)
	}
	if !strings.HasPrefix(result, "postgres://") {
		t.Errorf("DSN.String() should start with postgres://, got %q", result)
	}
}

func TestDSNStringPasswordless(t *testing.T) {
	// DSN without password
	dsn := &DSN{
		Engine:   "mysql",
		User:     "root",
		Host:     "localhost",
		Port:     "3306",
		Database: "myapp",
	}

	result := dsn.String()

	// URL includes colon even with empty password (Go's url.UserPassword behavior)
	expected := "mysql://root:@localhost:3306/myapp"
	if result != expected {
		t.Errorf("DSN.String() = %q, want %q", result, expected)
	}
}

func TestDSNStringNoUser(t *testing.T) {
	// DSN without user or password
	dsn := &DSN{
		Engine:   "mysql",
		Host:     "localhost",
		Port:     "3306",
		Database: "myapp",
	}

	result := dsn.String()

	expected := "mysql://localhost:3306/myapp"
	if result != expected {
		t.Errorf("DSN.String() = %q, want %q", result, expected)
	}
}

func TestDSNStringWithServerVersion(t *testing.T) {
	dsn := &DSN{
		Engine:        "mysql",
		User:          "root",
		Password:      "secret",
		Host:          "localhost",
		Port:          "3306",
		Database:      "myapp",
		ServerVersion: "8.0",
	}

	result := dsn.String()

	// Should contain serverVersion query parameter
	if !strings.Contains(result, "serverVersion=8.0") {
		t.Errorf("DSN.String() should include serverVersion param, got %q", result)
	}
}

func TestDSNStringNoPort(t *testing.T) {
	dsn := &DSN{
		Engine:   "mysql",
		User:     "root",
		Password: "secret",
		Host:     "localhost",
		Database: "myapp",
	}

	result := dsn.String()

	expected := "mysql://root:secret@localhost/myapp"
	if result != expected {
		t.Errorf("DSN.String() = %q, want %q", result, expected)
	}
}

func TestDSNStringSpecialCharactersInPassword(t *testing.T) {
	dsn := &DSN{
		Engine:   "mysql",
		User:     "root",
		Password: "p@ss:w#rd!",
		Host:     "localhost",
		Port:     "3306",
		Database: "myapp",
	}

	result := dsn.String()

	// Special characters should be URL-encoded
	if !strings.Contains(result, "p%40ss%3Aw%23rd%21") {
		t.Errorf("DSN.String() should URL-encode special chars in password, got %q", result)
	}
}
