package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views/games"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (s *Server) gamesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	playerWinLosses := games.UserWinLosses{}
	for _, match := range game.Matches {
		for _, participant := range match.Participants {
			winCount := playerWinLosses[participant.UserID].Wins
			lossCount := playerWinLosses[participant.UserID].Losses

			if participant.Result == db.MatchResultWin {
				winCount++
			} else if participant.Result == db.MatchResultLoss {
				lossCount++
			}

			playerWinLosses[participant.UserID] = games.WinLosses{
				Wins:   winCount,
				Losses: lossCount,
			}
		}
	}

	gameWithPendingMatches, err := s.app.GetGameById(gameId, true)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, http.StatusOK, games.GamePage(games.GamePageProps{
		Game:              *game,
		Office:            game.Office,
		UserWinLosses:     playerWinLosses,
		User:              user,
		PendingMatchCount: len(gameWithPendingMatches.Matches),
	}))
}

func (s *Server) gamesPlayPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	endpoint := game.Link() + "/play"
	return render(c, http.StatusOK, games.PlayGamePage(*game, game.Office, game.Office.Players, endpoint, user))
}

func (s *Server) gamesPlayFormHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Request().ParseForm()
	winners := c.Request().Form["Winners"]
	losers := c.Request().Form["Losers"]
	note := c.FormValue("note")

	err = games.ValidatePlayMatchForm(game, games.PlayMatchFormData{
		Note:    note,
		Winners: winners,
		Losers:  losers,
	})
	if err != nil {
		return render(c, http.StatusOK, games.PlayMatchFormErrors(err))
	}

	match, err := s.app.LogMatch(user, game, note, winners, losers)
	if err != nil {
		return render(c, http.StatusOK, games.PlayMatchFormErrors(err))
	}

	// Not the end of the world if the auto approve doesnt work
	_ = s.app.ApproveMatch(user, match)
	if match.IsApproved() {
		c.Response().Header().Set("HX-Redirect", game.Link())
		return c.NoContent(http.StatusOK)
	}

	gameHome := game.Link() + fmt.Sprintf("/pending/%d", match.ID)
	c.Response().Header().Set("HX-Redirect", gameHome)
	return c.NoContent(http.StatusOK)
}

func (s *Server) gamePendingMatchesPage(c echo.Context) error {
	user := userFromContext(c)

	gameId := c.Param("id")
	game, err := s.app.GetGameById(gameId, true)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, http.StatusOK, games.PendingMatchesPage(*game, game.Office, game.Matches, user))
}

func (s *Server) pendingMatchPage(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")
	gameId := c.Param("id")

	matchId := c.Param("matchId")
	match, err := s.app.GetMatchById(matchId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Check if this match is still pending
	if match.State != db.MatchStatePending {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/offices/%s/games/%s", officeCode, gameId))
	}

	return render(c, http.StatusOK, games.PendingMatchPage(match.Game, match.Game.Office, *match, user))
}

func (s *Server) pendingMatchApproveHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")
	gameId := c.Param("id")

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

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/games/%s/pending", officeCode, gameId))
	return c.NoContent(http.StatusOK)
}

func (s *Server) gameAdminPage(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return render(c, http.StatusOK, games.GameAdminPage(*game, game.Office, user))
}

func (s *Server) deleteGameHandler(c echo.Context) error {
	gameIdString := c.Param("id")
	office := c.Param("code")

	gameId, err := strconv.Atoi(gameIdString)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid game ID")
	}

	game := &db.Game{
		Model: gorm.Model{
			ID: uint(gameId),
		},
	}
	err = s.db.C.Delete(game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s", office))
	return c.NoContent(http.StatusOK)
}

func (s *Server) editGameHandler(c echo.Context) error {
	gameId := c.Param("id")
	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	office := c.Param("code")
	newName := c.FormValue("name")
	newMinParticipants := c.FormValue("min-participants")
	newMaxParticipants := c.FormValue("max-participants")
	gameType := c.FormValue("game-type")

	formData := games.EditGameFormData{
		Name:            newName,
		MinParticipants: newMinParticipants,
		MaxParticipants: newMaxParticipants,
		GameType:        gameType,
	}

	if newName == "" {
		errs := games.EditGameFormErrors{Name: "Name is required"}
		return render(c, http.StatusOK, games.EditGameForm(formData, errs, office, *game))
	}

	minParticipants, err := strconv.Atoi(newMinParticipants)
	if err != nil {
		errs := games.EditGameFormErrors{MinParticipants: "Min participants must be a number"}
		return render(c, http.StatusOK, games.EditGameForm(formData, errs, office, *game))
	}

	maxParticipants, err := strconv.Atoi(newMaxParticipants)
	if err != nil {
		errs := games.EditGameFormErrors{MaxParticipants: "Max participants must be a number"}
		return render(c, http.StatusOK, games.EditGameForm(formData, errs, office, *game))
	}

	if minParticipants > maxParticipants {
		errs := games.EditGameFormErrors{MinParticipants: "Min participants must be less than max participants"}
		return render(c, http.StatusOK, games.EditGameForm(formData, errs, office, *game))
	}

	for i, gt := range db.GameTypes {
		if gt.Value == gameType {
			break
		}

		if i == len(db.GameTypes)-1 {
			errs := games.EditGameFormErrors{GameType: "Invalid game type"}
			return render(c, http.StatusOK, games.EditGameForm(formData, errs, office, *game))
		}
	}

	err = s.db.C.Model(&game).Where("id = ?", game.ID).Updates(map[string]interface{}{"name": newName, "min_participants": minParticipants, "max_participants": maxParticipants, "game_type": gameType}).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}

func (s *Server) pendingMatchDeleteHandler(c echo.Context) error {
	user := userFromContext(c)

	matchId := c.Param("matchId")
	gameId := c.Param("id")
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

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/games/%s/pending", officeCode, gameId))
	return c.NoContent(http.StatusOK)
}

func (s *Server) matchesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, http.StatusOK, games.MatchesPage(
		games.MatchesPageProps{
			User:    user,
			Matches: game.Matches,
			Office:  game.Office,
			Game:    *game,
		},
	))
}
