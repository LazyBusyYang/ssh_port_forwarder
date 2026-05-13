//go:build integration

package repository

import (
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSQLiteConnection tests basic SQLite database connectivity
func TestSQLiteConnection(t *testing.T) {
	dbPath := os.Getenv("TEST_DB_PATH")
	if dbPath == "" {
		t.Skip("TEST_DB_PATH environment variable not set, skipping SQLite integration test")
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	// Test basic connectivity by executing a simple query
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying SQL DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping SQLite database: %v", err)
	}

	t.Log("SQLite connection established successfully")
}

// TestSQLiteBasicOperations tests basic database operations on SQLite
func TestSQLiteBasicOperations(t *testing.T) {
	dbPath := os.Getenv("TEST_DB_PATH")
	if dbPath == "" {
		t.Skip("TEST_DB_PATH environment variable not set, skipping SQLite integration test")
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	// Test creating a table
	type TestTable struct {
		ID   uint
		Name string
	}

	err = db.AutoMigrate(&TestTable{})
	if err != nil {
		t.Fatalf("Failed to migrate test table: %v", err)
	}

	// Test insert
	testRecord := TestTable{Name: "test"}
	err = db.Create(&testRecord).Error
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	if testRecord.ID == 0 {
		t.Fatal("Expected non-zero ID after insert")
	}

	// Test query
	var retrieved TestTable
	err = db.First(&retrieved, testRecord.ID).Error
	if err != nil {
		t.Fatalf("Failed to query record: %v", err)
	}

	if retrieved.Name != "test" {
		t.Fatalf("Expected name 'test', got '%s'", retrieved.Name)
	}

	// Cleanup
	err = db.Migrator().DropTable(&TestTable{})
	if err != nil {
		t.Logf("Warning: failed to drop test table: %v", err)
	}

	t.Log("SQLite basic operations completed successfully")
}

// TestSQLiteAdapter tests the GORMAdapter with SQLite
func TestSQLiteAdapter(t *testing.T) {
	dbPath := os.Getenv("TEST_DB_PATH")
	if dbPath == "" {
		t.Skip("TEST_DB_PATH environment variable not set, skipping SQLite integration test")
	}

	adapter, err := NewGORMAdapter("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create GORMAdapter for SQLite: %v", err)
	}

	if adapter.DB == nil {
		t.Fatal("Expected non-nil DB in adapter")
	}

	// Test AutoMigrate
	err = adapter.AutoMigrate()
	if err != nil {
		t.Fatalf("Failed to auto migrate with adapter: %v", err)
	}

	t.Log("SQLite adapter test completed successfully")
}
