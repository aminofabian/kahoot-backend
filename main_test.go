package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
    // Load environment variables from .env file
    if err := godotenv.Load(".env"); err != nil {
        log.Fatalf("Error loading .env file: %v", err)
    }

    // Connect to the PostgreSQL database
    db, err := connectToDB()
    if err != nil {
        log.Fatalf("Failed to connect to the database: %v", err)
    }
    testDB = db

    // Try executing a simple query
    rows, err := testDB.Query("SELECT 1")
    if err != nil {
        log.Fatalf("Failed to execute query: %v", err)
    }
    rows.Close()

    // Run the tests
    os.Exit(m.Run())
}
func TestGetQuizzes(t *testing.T) {
    // Create a new Fiber app and request context
    app := fiber.New()
    app.Get("/api/quizes", getQuizzes(testDB))
    req := httptest.NewRequest("GET", "/api/quizes", nil)
    resp, err := app.Test(req, -1)
    assert.NoError(t, err)

    // Assert the response
    assert.Equal(t, 200, resp.StatusCode)

    // Decode the response body
    var responseData []map[string]any
    err = json.NewDecoder(resp.Body).Decode(&responseData)
    assert.NoError(t, err)

    // Assert the response data
    assert.Len(t, responseData, 1)
    assert.Equal(t, float64(123), responseData[0]["test"])

    // Try executing a simple query
    rows, err := testDB.Query("SELECT 1")
    assert.NoError(t, err)
    assert.True(t, rows.Next())
    rows.Close()
}

