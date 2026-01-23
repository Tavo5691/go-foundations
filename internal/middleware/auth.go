package middleware

import (
	"fmt"
	"go-foundations/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Auth(jwtKey string) gin.HandlerFunc {
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
			return []byte(jwtKey), nil
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
