package testutil

import "testing"

func TestDatabaseName(t *testing.T) {
	name, err := databaseName("postgres://postgres:postgres@localhost:55440/test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if name != "test" {
		t.Fatalf("database name = %q, want test", name)
	}
}

func TestDatabaseNameRejectsMissingName(t *testing.T) {
	if _, err := databaseName("postgres://postgres:postgres@localhost:55440/"); err == nil {
		t.Fatal("expected an error for a URL without a database name")
	}
}
