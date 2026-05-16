package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Structured logger that writes JSON to stdout and a log file.
type appLogger struct {
	file *os.File
}

func newAppLogger(path string) *appLogger {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file %s: %v", path, err)
	}
	return &appLogger{file: f}
}

func (l *appLogger) log(level, msg string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"level":   level,
		"message": msg,
		"_msg":    msg,
		"service": "backend",
	}
	for k, v := range fields {
		entry[k] = v
	}
	line, _ := json.Marshal(entry)
	// stdout for container logs
	fmt.Println(string(line))
	// file for Promtail scraping
	fmt.Fprintln(l.file, string(line))
}

func (l *appLogger) Info(msg string, fields map[string]interface{}) {
	l.log("info", msg, fields)
}

func (l *appLogger) Error(msg string, fields map[string]interface{}) {
	l.log("error", msg, fields)
}

// ---------- handlers ----------

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

type submitRequest struct {
	Name    string `form:"name" json:"name" binding:"required"`
	Email   string `form:"email" json:"email" binding:"required"`
	Message string `form:"message" json:"message" binding:"required"`
}

func submitHandler(l *appLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req submitRequest
		if err := c.ShouldBind(&req); err != nil {
			l.Error("validation failed", map[string]interface{}{"error": err.Error()})
			c.JSON(http.StatusBadRequest, gin.H{"error": "all fields are required: name, email, message"})
			return
		}

		var id int
		err := db.QueryRow(
			"INSERT INTO submissions (name, email, message) VALUES ($1, $2, $3) RETURNING id",
			req.Name, req.Email, req.Message,
		).Scan(&id)
		if err != nil {
			l.Error("db insert failed", map[string]interface{}{"error": err.Error(), "name": req.Name, "email": req.Email})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save submission"})
			return
		}

		l.Info("submission saved", map[string]interface{}{
			"id":    id,
			"name":  req.Name,
			"email": req.Email,
		})

		c.JSON(http.StatusCreated, gin.H{"status": "ok", "id": id})
	}
}

// ---------- main ----------

func main() {
	// Database connection
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "postgres"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "appuser"),
		getEnv("DB_PASSWORD", "apppass"),
		getEnv("DB_NAME", "appdb"),
	)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Retry DB connection up to 30s
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("waiting for postgres... (%d/30): %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("could not connect to postgres: %v", err)
	}
	log.Println("connected to postgres")

	// Logger
	logPath := getEnv("LOG_FILE", "/var/log/backend/app.log")
	// ensure directory exists
	if err := os.MkdirAll("/var/log/backend", 0755); err != nil {
		log.Fatalf("failed to create log dir: %v", err)
	}
	l := newAppLogger(logPath)
	defer l.file.Close()

	l.Info("backend starting", map[string]interface{}{
		"db_host": getEnv("DB_HOST", "postgres"),
		"port":    getEnv("PORT", "8080"),
	})

	// Router
	r := gin.Default()

	r.GET("/health", healthHandler)
	r.POST("/submit", submitHandler(l))

	port := getEnv("PORT", "8080")
	r.Run(":" + port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}