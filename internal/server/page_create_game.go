package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func createGameHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	office := db.Office{}
	if err := db.GetDB().Where("code = ?", officeCode).First(&office).Error; err != nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	pageContent := views.CreateGamePage(office)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func createGameFormHandler(c echo.Context) error {
	d := db.GetDB()

	officeCode := c.Param("code")
	office := db.Office{}
	if err := d.Where("code = ?", officeCode).First(&office).Error; err != nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	gameName := c.FormValue("game")
	if gameName == "" {
		errs := views.FormErrors{"game": "Name is required"}
		return render(c, http.StatusOK, views.CreateGameForm(views.FormData{}, errs, officeCode))
	}

	game := db.Game{
		Name:     gameName,
		OfficeID: office.ID,
	}
	if err := d.Create(&game).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create game")
	}

	c.Response().Header().Set("HX-Redirect", "/offices/"+officeCode)
	return c.NoContent(http.StatusOK)
}
