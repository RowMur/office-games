package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/RowMur/office-games/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userContextKey = contextKey("user")

func userFromContext(ctx context.Context) *database.User {
	user, ok := ctx.Value(userContextKey).(*database.User)
	if !ok {
		return nil
	}
	return user
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCookie, err := r.Cookie("auth")
		if err != nil {
			if err != http.ErrNoCookie {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		if authCookie.Value == "" {
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		secret := os.Getenv("JWT_SECRET")
		token, err := jwt.ParseWithClaims(authCookie.Value, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			http.Error(w, "invalid token, could not parse", http.StatusUnauthorized)
			return
		}

		tokenIssuer, err := token.Claims.GetIssuer()
		if err != nil {
			http.Error(w, "invalid token, could not parse", http.StatusUnauthorized)
			return
		}
		if tokenIssuer != issuer {
			http.Error(w, "invalid token, invalid issuer", http.StatusUnauthorized)
			return
		}

		tokenExp, err := token.Claims.GetExpirationTime()
		if err != nil {
			http.Error(w, "invalid token, could not parse", http.StatusUnauthorized)
			return
		}
		if tokenExp.Time.Unix() < time.Now().UTC().Unix() {
			w.Header().Set("Set-Cookie", "auth=; Max-Age=0")
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		userId, err := getUserIdFromToken(token)
		user := database.User{}
		result := database.GetDB().Where("ID = ?", userId).First(&user)
		if result.Error != nil {
			w.Header().Set("Set-Cookie", "auth=; Max-Age=0")
			http.Error(w, "invalid token, could not find user", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const issuer = "office-games-access"
const tokenDuration = time.Hour * 1440

func generateToken(userId int) (string, error) {
	timeNow := time.Now().UTC()
	expiryTime := timeNow.Add(tokenDuration)

	claims := &jwt.RegisteredClaims{
		Issuer: issuer,
		IssuedAt: &jwt.NumericDate{
			Time: timeNow,
		},
		ExpiresAt: &jwt.NumericDate{
			Time: expiryTime,
		},
		Subject: fmt.Sprint(userId),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable not set")
	}
	return token.SignedString([]byte(secret))
}

func getUserIdFromToken(token *jwt.Token) (int, error) {
	userId, err := token.Claims.GetSubject()
	if err != nil {
		return 0, errors.New("error parsing token")
	}

	intUserId, err := strconv.Atoi(userId)
	if err != nil {
		return 0, errors.New("error parsing token")
	}

	return intUserId, nil
}
