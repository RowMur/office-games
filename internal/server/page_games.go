package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) gamesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	playerWinLosses := map[uint]views.WinLosses{}
	for _, match := range game.Matches {
		for _, participant := range match.Participants {
			winCount := playerWinLosses[participant.UserID].Wins
			lossCount := playerWinLosses[participant.UserID].Losses

			if participant.Result == db.MatchResultWin {
				winCount++
			} else if participant.Result == db.MatchResultLoss {
				lossCount++
			}

			playerWinLosses[participant.UserID] = views.WinLosses{
				Wins:   winCount,
				Losses: lossCount,
			}
		}
	}

	pageContent := views.GamePage(*game, game.Office, playerWinLosses, *user)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func (s *Server) gamesPlayPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	game, err := s.app.GetGameById(gameId, false)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	endpoint := fmt.Sprintf("/offices/%s/games/%s/play", game.Office.Code, gameId)
	pageContent := views.PlayGamePage(*game, game.Office, game.Office.Players, endpoint, *user)
	return render(c, http.StatusOK, views.Page(pageContent, user))
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

	err = views.ValidatePlayMatchForm(game, views.PlayMatchFormData{
		Note:    note,
		Winners: winners,
		Losers:  losers,
	})
	if err != nil {
		return render(c, http.StatusOK, views.PlayMatchFormErrors(views.FormErrors{
			"submit": err.Error(),
		}))
	}

	match, err := s.app.LogMatch(user, game, note, winners, losers)
	if err != nil {
		return render(c, http.StatusOK, views.PlayMatchFormErrors(views.FormErrors{
			"submit": err.Error(),
		}))
	}

	// Not the end of the world if the auto approve doesnt work
	_ = s.app.ApproveMatch(user, match)
	if match.IsApproved() {
		gameHome := fmt.Sprintf("/offices/%s/games/%s", game.Office.Code, gameId)
		c.Response().Header().Set("HX-Redirect", gameHome)
		return c.NoContent(http.StatusOK)
	}

	gameHome := fmt.Sprintf("/offices/%s/games/%s/pending/%d", game.Office.Code, gameId, match.ID)
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

	pageContent := views.PendingMatchesPage(*game, game.Office, game.Matches, *user)
	return render(c, http.StatusOK, views.Page(pageContent, user))
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

	pageContent := views.PendingMatchPage(match.Game, match.Game.Office, *match)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func (s *Server) pendingMatchApproveHandler(c echo.Context) error {
	user := userFromContext(c)
	officeCode := c.Param("code")
	gameId := c.Param("id")

	match, err := s.app.GetMatchById(c.Param("matchId"))
	if err != nil {
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}

	err = s.app.ApproveMatch(user, match)
	if err != nil {
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
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
	return render(c, http.StatusOK, views.Page(views.GameAdminPage(*game, game.Office, *user), user))
}

func (s *Server) deleteGameHandler(c echo.Context) error {
	gameId := c.Param("id")
	office := c.Param("code")

	err := s.db.C.Delete(&db.Game{}, gameId).Error
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

	formData := views.FormData{
		"name":             newName,
		"min-participants": newMinParticipants,
		"max-participants": newMaxParticipants,
		"game-type":        gameType,
	}

	if newName == "" {
		errs := views.FormErrors{"name": "Name is required"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, *game))
	}

	minParticipants, err := strconv.Atoi(newMinParticipants)
	if err != nil {
		errs := views.FormErrors{"min-participants": "Min participants must be a number"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, *game))
	}

	maxParticipants, err := strconv.Atoi(newMaxParticipants)
	if err != nil {
		errs := views.FormErrors{"max-participants": "Max participants must be a number"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, *game))
	}

	if minParticipants > maxParticipants {
		errs := views.FormErrors{"min-participants": "Min participants must be less than max participants"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, *game))
	}

	for i, gt := range db.GameTypes {
		if gt.Value == gameType {
			break
		}

		if i == len(db.GameTypes)-1 {
			errs := views.FormErrors{"game-type": "Invalid game type"}
			return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, *game))
		}
	}

	err = s.db.C.Model(&game).Where("id = ?", game.ID).Updates(map[string]interface{}{"name": newName, "min_participants": minParticipants, "max_participants": maxParticipants, "game_type": gameType}).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}
