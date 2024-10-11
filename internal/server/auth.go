package server

import (
	"fmt"
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/email"
	"github.com/RowMur/office-games/internal/token"
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

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
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
		cc := &contextWithUser{c, user}
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
		if userFromContext(c) != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}
		return next(c)
	}
}

func enforceMember(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := userFromContext(c)
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		officeCode := c.Param("code")
		office := &db.Office{}
		err := db.GetDB().Where("code = ?", officeCode).Preload("Players").First(office).Error
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, u := range office.Players {
			if u.ID == user.ID {
				return next(c)
			}
		}

		return echo.NewHTTPError(http.StatusForbidden, "You are not a member of this office")
	}
}

func enforceAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := userFromContext(c)
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		officeCode := c.Param("code")
		office := &db.Office{}
		err := db.GetDB().Where("code = ?", officeCode).First(office).Error
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		if office.AdminRefer != user.ID {
			return echo.NewHTTPError(http.StatusForbidden, "You are not the admin of this office")
		}

		return next(c)
	}
}

func sendForgotPasswordEmail(c echo.Context, user *db.User) error {
	token, err := token.GenerateToken(user.ID, token.ForgotPasswordToken)
	if err != nil {
		return err
	}

	host := c.Request().Host
	emailBody := fmt.Sprintf("Click <a href=\"http://%s/reset-password?token=%s\">here</a> to reset your password", host, token)

	return email.SendEmail([]string{user.Email}, "Office Games - Password Recovery", emailBody)
}
