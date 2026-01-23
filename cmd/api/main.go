package main

import (
	"database/sql"
	"go-foundations/internal/config"
	"go-foundations/internal/handlers"
	"go-foundations/internal/middleware"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to DB")
	defer db.Close()

	router := gin.Default()
	protected := router.Group("/")
	protected.Use(middleware.Auth(cfg.JWTKey))

	h := handlers.New(db, cfg.JWTKey)

	router.GET("/health", h.Health)

	router.POST("/register", h.Register)
	router.POST("/login", h.Login)

	protected.GET("/tasks", h.GetTasks)
	protected.POST("/tasks", h.CreateTask)
	protected.GET("/tasks/:id", h.GetTask)
	protected.PUT("/tasks/:id", h.UpdateTask)
	protected.DELETE("/tasks/:id", h.DeleteTask)

	log.Fatal(router.Run())
}
