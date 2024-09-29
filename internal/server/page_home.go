package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func pageHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusUnauthorized, "/sign-in")
	}

	userHasOffices := len(user.Offices) > 0

	mainPageContent := views.MainPage(*user, userHasOffices, user.Offices)
	return render(c, http.StatusOK, views.Page(mainPageContent, user))
}
