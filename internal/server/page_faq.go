package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func faqPageHandler(c echo.Context) error {
	user := userFromContext(c)
	pageContent := views.FaqPage()
	return render(c, http.StatusOK, views.Page(pageContent, user))
}
