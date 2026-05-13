//go:build integration

package repository

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestMySQLConnection tests basic MySQL database connectivity
func TestMySQLConnection(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN environment variable not set, skipping MySQL integration test")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to MySQL database: %v", err)
	}

	// Test basic connectivity by executing a simple query
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying SQL DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping MySQL database: %v", err)
	}

	t.Log("MySQL connection established successfully")
}

// TestMySQLBasicOperations tests basic database operations on MySQL
func TestMySQLBasicOperations(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN environment variable not set, skipping MySQL integration test")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to MySQL database: %v", err)
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

	t.Log("MySQL basic operations completed successfully")
}

// TestMySQLAdapter tests the GORMAdapter with MySQL
func TestMySQLAdapter(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN environment variable not set, skipping MySQL integration test")
	}

	adapter, err := NewGORMAdapter("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to create GORMAdapter for MySQL: %v", err)
	}

	if adapter.DB == nil {
		t.Fatal("Expected non-nil DB in adapter")
	}

	// Test AutoMigrate
	err = adapter.AutoMigrate()
	if err != nil {
		t.Fatalf("Failed to auto migrate with adapter: %v", err)
	}

	t.Log("MySQL adapter test completed successfully")
}
