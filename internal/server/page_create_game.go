package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views/games"
	"github.com/labstack/echo/v4"
)

func (s *Server) createGameHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if office == nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	return render(c, http.StatusOK, games.CreateGamePage(user, *office))
}

func (s *Server) createGameFormHandler(c echo.Context) error {
	officeCode := c.Param("code")
	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if office == nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	gameName := c.FormValue("game")
	if gameName == "" {
		errs := games.CreateGameFormErrors{
			Game: "Name is required",
		}
		return render(c, http.StatusOK, games.CreateGameForm(games.CreateGameFormData{}, errs, officeCode))
	}

	game := db.Game{
		Name:     gameName,
		OfficeID: office.ID,
	}
	if err := s.db.C.Create(&game).Error; err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create game")
	}

	c.Response().Header().Set("HX-Redirect", "/offices/"+officeCode)
	return c.NoContent(http.StatusOK)
}
