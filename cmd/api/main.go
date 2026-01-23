package main

import (
	"database/sql"
	"go-foundations/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

type HealthResponse struct {
	Status string `json:"status,omitempty"`
}

func parseId(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
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

	router.GET("/health", healthHandler)
	router.GET("/tasks", getTasksHandler)
	router.POST("/tasks", postTaskHandler)
	router.GET("/tasks/:id", getTaskByIdHandler)
	router.PUT("/tasks/:id", putTaskByIdHandler)
	router.DELETE("/tasks/:id", deleteTaskByIdHandler)

	log.Fatal(router.Run())
}
