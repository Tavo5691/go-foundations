package main

import (
	"database/sql"
	"fmt"
	"go-foundations/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		tokenString := authHeader[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte("key"), nil
		})
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}

func main() {
	db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/taskdb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to DB")
	defer db.Close()

	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	}

	registerHandler := func(c *gin.Context) {
		request := &models.User{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			c.Status(http.StatusInternalServerError)
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
				c.Status(http.StatusConflict)
				return
			}

			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusCreated, user)
	}

	loginHandler := func(c *gin.Context) {
		request := &models.User{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		row := db.QueryRow(`
		SELECT id, email, password, created_at
		FROM users
		WHERE email = $1`, request.Email)

		user := models.User{}
		err = row.Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
		if err == sql.ErrNoRows {
			c.Status(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
		if err != nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			jwt.MapClaims{
				"sub": user.ID.String(),
				"exp": time.Now().Add(24 * time.Hour).Unix()})

		tokenString, err := token.SignedString([]byte("key"))
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, models.Token{Token: tokenString})
	}

	getTasksHandler := func(c *gin.Context) {
		results := make([]models.Task, 0)

		rows, err := db.Query("SELECT id, title, description, completed, created_at FROM tasks")
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			task := models.Task{}
			err = rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			results = append(results, task)
		}

		err = rows.Err()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, results)
	}

	postTaskHandler := func(c *gin.Context) {
		request := &models.Task{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.Status(http.StatusBadRequest)
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
			c.Status(http.StatusNotFound)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusCreated, task)
	}

	getTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		row := db.QueryRow("SELECT id, title, description, completed, created_at FROM tasks WHERE id = $1", taskId)

		task := models.Task{}
		err = row.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt)
		if err == sql.ErrNoRows {
			c.Status(http.StatusNotFound)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, task)
	}

	putTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		request := &models.Task{}
		err = c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		result, err := db.Exec(
			`UPDATE tasks
			SET title = $1, description = $2, completed = $3
			WHERE id = $4`,
			request.Title, request.Description, request.Completed, taskId)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if rowsAffected == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		c.Status(http.StatusOK)
	}

	deleteTaskByIdHandler := func(c *gin.Context) {
		taskId, err := parseId(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		result, err := db.Exec("DELETE FROM tasks WHERE id = $1", taskId)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if rowsAffected == 0 {
			c.Status(http.StatusNotFound)
			return
		}

		c.Status(http.StatusNoContent)
	}

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
