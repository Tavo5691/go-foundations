package handlers

import (
	"database/sql"

	"github.com/google/uuid"
)

type Handler struct {
	db     *sql.DB
	jwtKey string
}

func New(db *sql.DB, jwtKey string) *Handler {
	return &Handler{db: db, jwtKey: jwtKey}
}

func parseId(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}
