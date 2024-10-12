package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func mePageHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	mePageContent := views.MePage(*user, views.FormData{"email": user.Email, "username": user.Username}, nil)
	return render(c, http.StatusOK, views.Page(mePageContent, user))
}

func (s *Server) meUpdateHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	username := c.FormValue("username")
	email := c.FormValue("email")

	if username == "" || email == "" {
		data := views.FormData{"username": username, "email": email}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if email == "" {
			errs["email"] = "Email is required"
		}
		falseVar := false
		return render(c, http.StatusOK, views.UserDetails(data, errs, &falseVar))
	}

	user, updateErrs, err := s.db.UpdateUser(user.ID, map[string]interface{}{"username": username, "email": email})
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if updateErrs != nil {
		formErrs := views.FormErrors{
			"username": updateErrs["username"],
			"email":    updateErrs["email"],
		}
		return render(c, http.StatusOK, views.UserDetails(views.FormData{"username": username, "email": email}, formErrs, nil))
	}

	formData := views.FormData{"email": user.Email, "username": user.Username}
	truePtr := true
	return render(c, http.StatusOK, views.UserDetails(formData, views.FormErrors{}, &truePtr))
}
