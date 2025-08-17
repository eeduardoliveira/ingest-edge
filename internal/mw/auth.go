// internal/mw/auth.go
package mw

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"strings"
)

type contextKey string

var DriverIDCtxKey = contextKey("driver_id")

func AuthMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_HS256_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		driverID, _ := claims["driver_id"].(string)
		if driverID == "" {
			http.Error(w, "driver_id claim required", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), CtxDriverKey{}, driverID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
