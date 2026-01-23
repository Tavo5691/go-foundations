package handlers

import (
	"database/sql"
	"go-foundations/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetTasks(c *gin.Context) {
	results := make([]models.Task, 0)

	rows, err := h.db.Query("SELECT id, title, description, completed, created_at FROM tasks")
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

func (h *Handler) CreateTask(c *gin.Context) {
	request := &models.Task{}
	err := c.ShouldBindBodyWithJSON(request)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	row := h.db.QueryRow(`INSERT INTO
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

func (h *Handler) GetTask(c *gin.Context) {
	taskId, err := parseId(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	row := h.db.QueryRow("SELECT id, title, description, completed, created_at FROM tasks WHERE id = $1", taskId)

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

func (h *Handler) UpdateTask(c *gin.Context) {
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

	result, err := h.db.Exec(
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

func (h *Handler) DeleteTask(c *gin.Context) {
	taskId, err := parseId(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	result, err := h.db.Exec("DELETE FROM tasks WHERE id = $1", taskId)
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
