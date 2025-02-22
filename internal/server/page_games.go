package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views/games"
	"github.com/labstack/echo/v4"
)

func (s *Server) gamesPlayPageHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	endpoint := office.Link() + "/play"
	return render(c, http.StatusOK, games.PlayGamePage(*office, office.Players, endpoint, user))
}

func (s *Server) gamesPlayFormHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Request().ParseForm()
	winners := c.Request().Form["Winners"]
	losers := c.Request().Form["Losers"]
	note := c.FormValue("note")
	isHandicap := c.FormValue("isHandicap") == "on"

	err = games.ValidatePlayMatchForm(games.PlayMatchFormData{
		Note:    note,
		Winners: winners,
		Losers:  losers,
	})
	if err != nil {
		return render(c, http.StatusOK, games.PlayMatchFormErrors(err))
	}

	match, err := s.app.LogMatch(user, office, note, winners, losers, isHandicap)
	if err != nil {
		return render(c, http.StatusOK, games.PlayMatchFormErrors(err))
	}

	// Not the end of the world if the auto approve doesnt work
	_ = s.app.ApproveMatch(user, match)
	if match.IsApproved() {
		c.Response().Header().Set("HX-Redirect", office.Link())
		return c.NoContent(http.StatusOK)
	}

	gameHome := office.Link() + fmt.Sprintf("/pending/%d", match.ID)
	c.Response().Header().Set("HX-Redirect", gameHome)
	return c.NoContent(http.StatusOK)
}

func (s *Server) gamePendingMatchesPage(c echo.Context) error {
	user := userFromContext(c)

	officeCode := c.Param("code")
	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var pendingMatches []db.Match
	err = s.db.C.
		Where("office_id = ? AND state = ?", office.ID, db.MatchStatePending).
		Order("created_at DESC").
		Preload("Participants.User").
		Preload("Creator").
		Preload("Approvals").
		Find(&pendingMatches).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, http.StatusOK, games.PendingMatchesPage(*office, pendingMatches, user))
}

func (s *Server) pendingMatchPage(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	matchId := c.Param("matchId")
	match, err := s.app.GetMatchById(matchId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Check if this match is still pending
	if match.State != db.MatchStatePending {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/offices/%s", officeCode))
	}

	return render(c, http.StatusOK, games.PendingMatchPage(match.Office, *match, user))
}

func (s *Server) pendingMatchApproveHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	match, err := s.app.GetMatchById(c.Param("matchId"))
	if err != nil {
		return render(c, http.StatusOK, games.MatchApproveError(err.Error()))
	}

	err = s.app.ApproveMatch(user, match)
	if err != nil {
		return render(c, http.StatusOK, games.MatchApproveError(err.Error()))
	}

	isMatchApproved, _ := s.app.IsMatchApproved(s.db.C, match)
	if !isMatchApproved {
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/pending", officeCode))
	return c.NoContent(http.StatusOK)
}

func (s *Server) pendingMatchDeleteHandler(c echo.Context) error {
	user := userFromContext(c)

	matchId := c.Param("matchId")
	officeCode := c.Param("code")

	match := &db.Match{}
	err := s.db.C.First(match, matchId).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if user.ID != match.CreatorID {
		return c.String(http.StatusForbidden, "You do not have permission to delete this match")
	}

	if match.State != db.MatchStatePending {
		return c.String(http.StatusForbidden, "You can only delete pending matches")
	}

	err = s.db.C.Delete(match).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/pending", officeCode))
	return c.NoContent(http.StatusOK)
}

const (
	matchesPerPage = 10
)

func (s *Server) matchesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	page := c.QueryParam("page")
	if page == "" {
		page = "0"
	}

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 0 {
		return c.String(http.StatusBadRequest, "Invalid page number")
	}

	startingIndex := pageInt * matchesPerPage
	if startingIndex > len(office.Matches)-1 {
		return c.String(http.StatusNotFound, "Page not found")
	}

	endingIndex := min(startingIndex+matchesPerPage, len(office.Matches))
	matchesToReturn := office.Matches[startingIndex:endingIndex]

	hasNextPage := len(office.Matches) > endingIndex
	nextPage := ""
	if hasNextPage {
		nextPage = strconv.Itoa(pageInt + 1)
	}

	processedGame, err := s.gp.Process(office.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if pageInt < 1 {
		// full page
		return render(c, http.StatusOK, games.MatchesPage(
			games.MatchesPageProps{
				User:          user,
				Matches:       matchesToReturn,
				Office:        *office,
				NextPage:      nextPage,
				ProcessedGame: processedGame,
			},
		))
	}

	// partial page
	return render(c, http.StatusOK, games.Matches(games.MatchesProps{Matches: matchesToReturn, NextPage: nextPage, ProcessedGame: processedGame, Office: *office}))
}

func (s *Server) gameStatsPageHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	processedGame, err := s.gp.Process(office.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, http.StatusOK, games.StatsPage(*office, user, *processedGame))
}

func (s *Server) gamePlayerStatsPostHandler(c echo.Context) error {
	officeCode := c.Param("code")

	stringPlayerId := c.FormValue("player")
	playerId, err := strconv.Atoi(stringPlayerId)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid player ID")
	}

	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	processedGame, err := s.gp.Process(office.ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	player := processedGame.GetPlayer(uint(playerId))
	if player == nil {
		return render(c, http.StatusOK, games.PlayerHasntPlayedYet())
	}

	return render(c, http.StatusOK, games.PlayerStats(*processedGame, *player))
}
