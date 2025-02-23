package server

import (
	"fmt"
	"net/http"
	"strconv"

	officeViews "github.com/RowMur/office-table-tennis/internal/views/office"
	"github.com/labstack/echo/v4"
)

func (s *Server) createTournamentFormHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officeCode := c.Param("code")
	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if office == nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	c.Request().ParseForm()
	participants := c.Request().Form["participants"]
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name is required")
	}

	participantUintIds := []uint{}
	for _, participant := range participants {
		id, err := strconv.ParseInt(participant, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid participant")
		}

		participantUintIds = append(participantUintIds, uint(id))
	}

	tournament, err := s.app.CreateTournament(user, name, *office, participantUintIds)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/tournament/%d", tournament.ID))
	return c.String(http.StatusOK, fmt.Sprintf("Tournament created: %d", tournament.ID))
}

func (s *Server) tournamentPageHandler(c echo.Context) error {
	officeCode := c.Param("code")
	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if office == nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	processedOffice, err := s.op.Process(office.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	stringTournamentId := c.Param("id")
	tournamentId, err := strconv.Atoi(stringTournamentId)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid tournament ID")
	}
	tournament := processedOffice.GetTournament(uint(tournamentId))
	if tournament == nil {
		return c.String(http.StatusNotFound, "Tournament not found")
	}

	return render(c, http.StatusOK, officeViews.TournamentPage(officeViews.TournamentPageProps{
		User:       user,
		Office:     *office,
		Tournament: *tournament,
	}))
}
