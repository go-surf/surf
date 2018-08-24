package sqldb

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// EnsurePostgres connects to PostgreSQL instance, create database and return
// connection to it. If connection to Postres cannot be established, test is
// skipped.
//
// Unless option is provided, defaults are used:
//   * Database name: test_database_<creation time in unix ns>
//   * Host: localhost
//   * Port: 5432
//   * SSLMode: disable
//   * User: postgres
//
// Function connects to 'postgres' database first to create new database.
func EnsurePostgres(t *testing.T, o *DBOpts) (testdb *sql.DB, cleanup func()) {
	t.Helper()

	if o == nil {
		o = &DBOpts{}
	}
	assignDefaultOpts(o)

	rootDsn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' dbname='postgres' sslmode='%s'",
		o.Host, o.Port, o.User, o.SSLMode)
	rootdb, err := sql.Open("postgres", rootDsn)
	if err != nil {
		t.Skipf("cannot connect to postgres: %s", err)
	}
	if err := rootdb.Ping(); err != nil {
		t.Skipf("cannot ping postgres: %s", err)
	}
	if _, err := rootdb.Exec("CREATE DATABASE " + o.DBName); err != nil {
		t.Fatalf("cannot create database: %s", err)
		rootdb.Close()
	}

	testDsn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' dbname='%s' sslmode='%s'",
		o.Host, o.Port, o.User, o.DBName, o.SSLMode)
	testdb, err = sql.Open("postgres", testDsn)
	if err != nil {
		t.Fatalf("cannot connect to created database: %s", err)
	}
	if err := testdb.Ping(); err != nil {
		t.Fatalf("cannot ping test database: %s", err)
	}
	t.Logf("test database created: %s", o.DBName)

	cleanup = func() {
		testdb.Close()
		if _, err := rootdb.Exec("DROP DATABASE " + o.DBName); err != nil {
			t.Logf("cannot delete test database %q: %s", o.DBName, err)
		}
		rootdb.Close()
	}
	return testdb, cleanup
}

// DBOpts defines options for test database connections
type DBOpts struct {
	User    string
	Port    int
	Host    string
	SSLMode string
	DBName  string
}

func assignDefaultOpts(o *DBOpts) {
	if o.DBName == "" {
		o.DBName = fmt.Sprintf("test_database_%d", time.Now().UnixNano())
	}
	if o.Host == "" {
		o.Host = "localhost"
	}
	if o.Port == 0 {
		o.Port = 5432
	}
	if o.SSLMode == "" {
		o.SSLMode = "disable"
	}
	if o.User == "" {
		o.User = "postgres"
	}
}

// ClonePostgres creates clone of given database.
//
// While this may speedup tests that require bootstraping with a lot of
// fixtures, be aware that content layout on the hard drive may differ from
// origin and default ordering may differ from original database.
func ClonePostgres(t *testing.T, from string, o *DBOpts) (clonedb *sql.DB, cleanup func()) {
	t.Helper()

	if o == nil {
		o = &DBOpts{}
	}
	assignDefaultOpts(o)

	rootDsn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' dbname='%s' sslmode='%s'",
		o.Host, o.Port, o.User, from, o.SSLMode)
	rootdb, err := sql.Open("postgres", rootDsn)
	if err != nil {
		t.Skipf("cannot connect to postgres: %s", err)
	}
	if err := rootdb.Ping(); err != nil {
		t.Skipf("cannot ping postgres: %s", err)
	}
	query := fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s", o.DBName, from)
	if _, err := rootdb.Exec(query); err != nil {
		t.Fatalf("cannot clone %q database: %s", from, err)
	}

	clonedb, err = sql.Open("postgres", rootDsn)
	if err != nil {
		t.Fatalf("cannot connect to created database: %s", err)
	}
	if err := clonedb.Ping(); err != nil {
		t.Fatalf("cannot ping cloned database: %s", err)
	}
	t.Logf("test database cloned: %s (from %s)", o.DBName, from)

	cleanup = func() {
		clonedb.Close()
		if _, err := rootdb.Exec("DROP DATABASE " + o.DBName); err != nil {
			t.Logf("cannot delete test database %q: %s", o.DBName, err)
		}
		rootdb.Close()
	}
	return clonedb, cleanup
}
