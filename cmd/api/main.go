package main

import (
	"go-foundations/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var db = make([]models.Task, 0)

type HealthResponse struct {
	Status string `json:"status,omitempty"`
}

func parseId(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

func findTaskById(id string) (int, *models.Task, error) {
	uuid, err := parseId(id)
	if err != nil {
		return -1, nil, err
	}

	for index := range db {
		if uuid == db[index].ID {
			return index, &db[index], nil
		}
	}

	return -1, nil, nil
}

func main() {
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	}

	getTasksHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, db)
	}

	postTaskHandler := func(c *gin.Context) {
		newTask := &models.Task{}
		err := c.ShouldBindBodyWithJSON(newTask)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		newTask.ID = uuid.New()
		newTask.CreatedAt = time.Now()

		db = append(db, *newTask)

		c.JSON(http.StatusCreated, newTask)
	}

	getTaskByIdHandler := func(c *gin.Context) {
		_, task, err := findTaskById(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if task == nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusOK, task)
	}

	putTaskByIdHandler := func(c *gin.Context) {
		request := &models.Task{}
		err := c.ShouldBindBodyWithJSON(request)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		_, task, err := findTaskById(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if task == nil {
			c.Status(http.StatusNotFound)
			return
		}

		task.Title = request.Title
		task.Description = request.Description
		task.Completed = request.Completed

		c.JSON(http.StatusOK, task)
	}

	deleteTaskByIdHandler := func(c *gin.Context) {
		index, _, err := findTaskById(c.Param("id"))
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if index == -1 {
			c.Status(http.StatusNotFound)
			return
		}

		db = append(db[:index], db[index+1:]...)
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
