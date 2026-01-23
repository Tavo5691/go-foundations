package main

import (
	"database/sql"
	"fmt"
	"go-foundations/internal/models"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/lib/pq"
)

type HealthResponse struct {
	Status string `json:"status,omitempty"`
}

func parseId(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
			return
		}
		tokenString := authHeader[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(os.Getenv("JWT_KEY")), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
			return
		}

		c.Next()
	}
}

func main() {
	// load env variables
	_ = godotenv.Load()

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatal("JWT_KEY environment variable is required")
	}

	// open DB connection
	db, err := sql.Open("postgres", os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to DB")
	defer db.Close()

	//health
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	}

	// users
	registerHandler := func(c *gin.Context) {
		request := &models.User{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		row := db.QueryRow(`INSERT INTO
			users(id, email, password, created_at)
			VALUES($1, $2, $3, $4)
			RETURNING id, email, created_at`,
			uuid.New(), request.Email, string(hashed), time.Now())

		user := models.User{}
		err = row.Scan(&user.ID, &user.Email, &user.CreatedAt)
		if err != nil {
			if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
				c.JSON(http.StatusConflict, models.ErrorResponse{Error: "email already registered"})
				return
			}

			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database error"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}

	loginHandler := func(c *gin.Context) {
		request := &models.User{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
			return
		}

		row := db.QueryRow(`
		SELECT id, email, password, created_at
		FROM users
		WHERE email = $1`, request.Email)

		user := models.User{}
		err = row.Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
			return
		}

		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.MapClaims{
				"sub": user.ID.String(),
				"exp": time.Now().Add(24 * time.Hour).Unix()})

		tokenString, err := token.SignedString([]byte(jwtKey))
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		c.JSON(http.StatusOK, models.Token{Token: tokenString})
	}

	// tasks
	getTasksHandler := func(c *gin.Context) {
		results := make([]models.Task, 0)

		rows, err := db.Query("SELECT id, title, description, completed, created_at FROM tasks")
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			task := models.Task{}
			err = rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
				return
			}

			results = append(results, task)
		}

		err = rows.Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		c.JSON(http.StatusOK, results)
	}

	postTaskHandler := func(c *gin.Context) {
		request := &models.Task{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
			return
		}

		row := db.QueryRow(`INSERT INTO
			tasks(id, title, description, completed, created_at)
			VALUES($1, $2, $3, $4, $5)
			RETURNING *`,
			uuid.New(), request.Title, request.Description, request.Completed, time.Now())

		task := models.Task{}
		err = row.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database error"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		c.JSON(http.StatusCreated, task)
	}

	getTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
			return
		}

		row := db.QueryRow("SELECT id, title, description, completed, created_at FROM tasks WHERE id = $1", taskId)

		task := models.Task{}
		err = row.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		c.JSON(http.StatusOK, task)
	}

	putTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
			return
		}

		request := &models.Task{}
		err = c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
			return
		}

		result, err := db.Exec(
			`UPDATE tasks
			SET title = $1, description = $2, completed = $3
			WHERE id = $4`,
			request.Title, request.Description, request.Completed, taskId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not found"})
			return
		}

		c.Status(http.StatusOK)
	}

	deleteTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
			return
		}

		result, err := db.Exec("DELETE FROM tasks WHERE id = $1", taskId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not found"})
			return
		}

		c.Status(http.StatusNoContent)
	}

	// routes
	router := gin.Default()
	protected := router.Group("/")
	protected.Use(authMiddleware())

	router.GET("/health", healthHandler)

	router.POST("/register", registerHandler)
	router.POST("/login", loginHandler)

	protected.GET("/tasks", getTasksHandler)
	protected.POST("/tasks", postTaskHandler)
	protected.GET("/tasks/:id", getTaskByIdHandler)
	protected.PUT("/tasks/:id", putTaskByIdHandler)
	protected.DELETE("/tasks/:id", deleteTaskByIdHandler)

	log.Fatal(router.Run())
}
