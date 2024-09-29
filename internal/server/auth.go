package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/RowMur/office-games/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type contextWithUser struct {
	echo.Context
	user *db.User
}

func userFromContext(c echo.Context) *db.User {
	cc, ok := c.(*contextWithUser)
	if !ok {
		return nil
	}
	return cc.user
}

func authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authCookie, err := c.Request().Cookie("auth")
		if err != nil && err != http.ErrNoCookie {
			fmt.Println("authMiddleware error", err.Error())
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if authCookie == nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
		}

		if authCookie.Value == "" {
			return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
		}

		secret := os.Getenv("JWT_SECRET")
		token, err := jwt.ParseWithClaims(authCookie.Value, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token, could not parse")
		}

		tokenIssuer, err := token.Claims.GetIssuer()
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token, could not parse")
		}
		if tokenIssuer != issuer {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token, invalid issuer")
		}

		tokenExp, err := token.Claims.GetExpirationTime()
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token, could not parse")
		}
		if tokenExp.Time.Unix() < time.Now().UTC().Unix() {
			return signOut(c)
		}

		userId, err := getUserIdFromToken(token)
		user := db.User{}
		result := db.GetDB().Where("ID = ?", userId).Preload("Offices").First(&user)
		if result.Error != nil {
			return signOut(c)
		}
		cc := &contextWithUser{c, &user}
		return next(cc)
	}
}

func enforceSignedOut(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authCookie, err := c.Request().Cookie("auth")
		if err != nil && err != http.ErrNoCookie {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		if authCookie != nil {
			return c.Redirect(http.StatusFound, "/")
		}
		return next(c)
	}
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
