package server

import (
	"fmt"
	"strconv"

	"github.com/RowMur/office-table-tennis/internal/db"
	"github.com/labstack/echo/v4"
)

func (s *Server) eloPageHandler(c echo.Context) error {
	gameId := c.Request().URL.Query().Get("game")
	if gameId == "" {
		return c.HTML(400, "game is required")
	}

	gameIdint, err := strconv.Atoi(gameId)
	if err != nil {
		return c.HTML(400, "Bad game ID")
	}

	rankings, err := s.gp.Process(uint(gameIdint))
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			return c.HTML(400, "Unrecognised game ID")
		}

		return c.HTML(500, "Something went wrong")
	}

	return c.String(200, fmt.Sprintf("%+v", rankings))
}
