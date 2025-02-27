package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RowMur/office-table-tennis/internal/db"
	"github.com/RowMur/office-table-tennis/internal/email"
	"github.com/RowMur/office-table-tennis/internal/token"
	"github.com/labstack/echo/v4"
)

type contextWithUser struct {
	echo.Context
	user  *db.User
	token *token.Token
}

func userFromContext(c echo.Context) *db.User {
	cc, ok := c.(*contextWithUser)
	if !ok {
		return nil
	}
	return cc.user
}

func usersTokenFromContext(c echo.Context) *token.Token {
	cc, ok := c.(*contextWithUser)
	if !ok {
		return nil
	}
	return cc.token
}

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		startTime := time.Now()
		defer func() {
			fmt.Printf("Req: %s | Auth middleware: %s\n", c.Request().URL.Path, time.Now().Sub(startTime))
		}()

		authCookie, err := c.Request().Cookie("auth")
		if err != nil && err != http.ErrNoCookie {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if authCookie == nil {
			return next(c)
		}

		if authCookie.Value == "" {
			return next(c)
		}

		token, err := token.ParseToken(authCookie.Value)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		if token.HasExpired {
			return signOut(c)
		}

		user, err := s.db.GetUserById(token.UserId)
		if user == nil || err != nil {
			return signOut(c)
		}

		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		cc := &contextWithUser{c, user, token}
		return next(cc)
	}
}

func enforceSignedIn(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if userFromContext(c) == nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
		}
		return next(c)
	}
}

func enforceSignedOut(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := userFromContext(c)
		if user != nil {
			if c.Request().Header.Get("Content-Type") == "application/json" {
				token := usersTokenFromContext(c)
				return c.JSON(http.StatusOK, map[string]string{"token": token.String})

			}
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}
		return next(c)
	}
}

func (s *Server) enforceMember(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := userFromContext(c)
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		officeCode := c.Param("code")
		for _, o := range user.Offices {
			if o.Code == officeCode {
				return next(c)
			}
		}

		return echo.NewHTTPError(http.StatusForbidden, "You are not a member of this office")
	}
}

func sendForgotPasswordEmail(c echo.Context, user *db.User) error {
	token, err := token.GenerateToken(user.ID, token.ForgotPasswordToken)
	if err != nil {
		return err
	}

	host := c.Request().Host
	emailBody := fmt.Sprintf("Click <a href=\"http://%s/reset-password?token=%s\">here</a> to reset your password", host, token)

	return email.SendEmail([]string{user.Email}, "Office Table Tennis - Password Recovery", emailBody)
}
