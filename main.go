package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	    "github.com/gofiber/fiber/v2/middleware/cors"
        	"github.com/gofiber/contrib/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Quiz represents the structure of a quiz in the database
type Quiz struct {
    ID          int    `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    CreatedAt   string `json:"created_at"`
}

func createQuiz(db *sql.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        log.Printf("Received POST request to /api/quizes")
        // Parse the request body into a Quiz struct
        var quiz Quiz
        if err := c.BodyParser(&quiz); err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid request body",
            })
        }

        // Insert the new quiz into the database and fetch the ID
        var id int
        err := db.QueryRow(`
            INSERT INTO quizzes (title, description)
            VALUES ($1, $2)
            RETURNING id
        `, quiz.Title, quiz.Description).Scan(&id)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to create quiz",
            })
        }

        // Return the created quiz
        return c.Status(fiber.StatusCreated).JSON(Quiz{
            ID:          id,
            Title:       quiz.Title,
            Description: quiz.Description,
            CreatedAt:   time.Now().Format(time.RFC3339),
        })
    }
}
// InsertQuiz inserts a new quiz into the database
func InsertQuiz(db *sql.DB, title string, description string) error {
    query := `INSERT INTO quizzes (title, description) VALUES ($1, $2)`
    _, err := db.Exec(query, title, description)
    if err != nil {
        return fmt.Errorf("failed to insert quiz: %w", err)
    }
    return nil
}

func main() {
    // Load environment variables from .env file
    if err := godotenv.Load(); err != nil {
        log.Printf("Warning: Error loading .env file: %v", err)
    }

    // Connect to the PostgreSQL database
    db, err := connectToDB()
    if err != nil {
        log.Fatalf("Failed to connect to the database: %v", err)
    }
    defer db.Close()

    // Run database migrations
    if err := runMigrations(db); err != nil {
        log.Fatalf("Failed to run database migrations: %v", err)
    }

    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
    })

		    // Add CORS middleware
    app.Use(cors.New(cors.Config{
        AllowOrigins: "http://localhost:5173",
        AllowHeaders: "Origin, Content-Type, Accept",
    }))


    // Route to check server status
    app.Get("/", index)

    // Route to get all quizzes
    app.Get("/api/quizes", getQuizzes(db))

    app.Get("/ws", websocket.New(func(c *websocket.Conn) {

		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", msg)

			if err = c.WriteMessage(mt, msg); err != nil {
				log.Println("write:", err)
				break
			}
		}

	}))

    // Route to create a new quiz
app.Post("/api/quizes", createQuiz(db))




    // Get port from environment variable or use default
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    addr := fmt.Sprintf(":%s", port)
    log.Printf("Starting server on port %s", port)
    if err := app.Listen(addr); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

// Function to handle migrations
func runMigrations(db *sql.DB) error {
    log.Println("Running database migrations...")

    // Create quizzes table
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS quizzes (
            id SERIAL PRIMARY KEY,
            title VARCHAR(255) NOT NULL,
            description TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return fmt.Errorf("failed to create quizzes table: %w", err)
    }

    // Insert some sample data if the table is empty
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM quizzes").Scan(&count)
    if err != nil {
        return fmt.Errorf("failed to check if quizzes table is empty: %w", err)
    }

    if count == 0 {
        log.Println("Inserting sample data...")
        _, err = db.Exec(`
            INSERT INTO quizzes (title, description) VALUES
            ('General Knowledge Quiz', 'Test your knowledge across various topics'),
            ('Science Quiz', 'Explore the wonders of science'),
            ('History Quiz', 'Journey through time with historical facts')
        `)
        if err != nil {
            return fmt.Errorf("failed to insert sample data: %w", err)
        }
    }

    log.Println("Database migrations completed successfully")
    return nil
}

// Handler for the root route
func index(c *fiber.Ctx) error {
    return c.SendString("Hello, World!")
}

// Handler to retrieve all quizzes
func getQuizzes(db *sql.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Query all quizzes from the database
        rows, err := db.Query(`
            SELECT id, title, description, created_at 
            FROM quizzes 
            ORDER BY created_at DESC
        `)
        if err != nil {
            log.Printf("Error querying quizzes: %v", err)
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to fetch quizzes",
            })
        }
        defer rows.Close()

        // Slice to store all quizzes
        var quizzes []Quiz

        // Iterate through the rows and scan into Quiz structs
        for rows.Next() {
            var quiz Quiz
            err := rows.Scan(
                &quiz.ID,
                &quiz.Title,
                &quiz.Description,
                &quiz.CreatedAt,
            )
            if err != nil {
                log.Printf("Error scanning quiz row: %v", err)
                return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                    "error": "Failed to process quiz data",
                })
            }
            quizzes = append(quizzes, quiz)
        }

        // Check for errors from iterating over rows
        if err = rows.Err(); err != nil {
            log.Printf("Error iterating quiz rows: %v", err)
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to retrieve all quizzes",
            })
        }

        log.Printf("Returning %d quizzes", len(quizzes))
        return c.JSON(quizzes)
    }
}

// Function to connect to the database
func connectToDB() (*sql.DB, error) {
    connStr := os.Getenv("DATABASE_URL")
    if connStr == "" {
        return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
    }

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to open database connection: %w", err)
    }

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping the database: %w", err)
    }

    return db, nil
}