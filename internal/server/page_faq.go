package server

import (
	"net/http"

	"github.com/RowMur/office-table-tennis/internal/views"
	"github.com/labstack/echo/v4"
)

func faqPageHandler(c echo.Context) error {
	user := userFromContext(c)
	return render(c, http.StatusOK, views.FaqPage(user))
}
