package server

import (
	"net/http"

	"github.com/RowMur/office-table-tennis/internal/views"
	"github.com/labstack/echo/v4"
)

func pageHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return render(c, http.StatusOK, views.LoggedOutHomepage())
	}

	userHasOffices := len(user.Offices) > 0

	return render(c, http.StatusOK, views.MainPage(user, userHasOffices, user.Offices))
}
