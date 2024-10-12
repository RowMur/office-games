package server

import (
	"net/http"

	t "github.com/RowMur/office-games/internal/token"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func resetPasswordPage(c echo.Context) error {
	token := tokenFromContext(c)
	pageContent := views.ResetPasswordPage(token.String)
	return render(c, 200, views.Page(pageContent, nil))
}

func (s *Server) resetPasswordFormHandler(c echo.Context) error {
	token := tokenFromContext(c)

	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm")

	errs := s.us.ResetPassword(token.UserId, password, confirmPassword)
	if errs != nil {
		if errs.Error != nil {
			formErrs := views.FormErrors{"submit": "Failed to reset password"}
			return render(c, http.StatusOK, views.ResetPasswordForm(views.FormData{}, formErrs, token.String))
		}

		formErrs := views.FormErrors{
			"password": errs.Password,
			"confirm":  errs.Confirm,
		}
		return render(c, http.StatusOK, views.ResetPasswordForm(views.FormData{}, formErrs, token.String))
	}

	return render(c, 200, views.ResetPasswordSuccess())
}

type contextWithToken struct {
	echo.Context
	token *t.Token
}

func tokenFromContext(c echo.Context) *t.Token {
	cc, ok := c.(*contextWithToken)
	if !ok {
		return nil
	}
	return cc.token
}

func resetPasswordTokenMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenString := c.QueryParam("token")
		if tokenString == "" {
			return c.String(400, "Token is required")
		}

		token, err := t.ParseToken(tokenString)
		if err != nil {
			return c.String(400, "Invalid token")
		}
		if token.HasExpired {
			return c.String(400, "Link has expired. Please request a new one.")
		}

		cc := &contextWithToken{c, token}
		return next(cc)
	}
}
