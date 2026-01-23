package handlers

import (
	"database/sql"
	"go-foundations/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Register(c *gin.Context) {
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

	row := h.db.QueryRow(`INSERT INTO
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

func (h *Handler) Login(c *gin.Context) {
	request := &models.User{}
	err := c.ShouldBindBodyWithJSON(request)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	row := h.db.QueryRow(`
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

	tokenString, err := token.SignedString([]byte(h.jwtKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "internal error"})
		return
	}

	c.JSON(http.StatusOK, models.Token{Token: tokenString})
}
