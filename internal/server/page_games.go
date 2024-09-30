package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/elo"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func gamesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	d := db.GetDB()
	game := db.Game{}
	if err := d.Where("id = ?", gameId).Preload("Office").Preload("Rankings.User").First(&game).Error; err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	pageContent := views.GamePage(game, game.Office, nil)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func gamesPlayPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	d := db.GetDB()
	game := db.Game{}
	if err := d.Where("id = ?", gameId).Preload("Office.Players").First(&game).Error; err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	endpoint := fmt.Sprintf("/offices/%s/games/%s/play", game.Office.Code, gameId)
	pageContent := views.PlayGamePage(game, game.Office, game.Office.Players, endpoint)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func gamesPlayFormHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	d := db.GetDB()
	game := db.Game{}
	if err := d.Where("id = ?", gameId).Preload("Office.Players").First(&game).Error; err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	c.Request().ParseForm()
	winners := c.Request().Form["Winners"]
	losers := c.Request().Form["Losers"]

	if len(winners) == 0 || len(losers) == 0 {
		errs := views.FormErrors{
			"Winners": "",
			"Losers":  "",
		}
		if len(winners) == 0 {
			errs["Winners"] = "Winners must be selected"
		}
		if len(losers) == 0 {
			errs["Losers"] = "Losers must be selected"
		}
		return render(c, http.StatusOK, views.PlayMatchFormErrors(errs))
	}

	if len(winners) != len(losers) {
		errs := views.FormErrors{
			"Winners": "",
			"Losers":  "",
			"submit":  "Winners and Losers must be of equal number",
		}
		return render(c, http.StatusOK, views.PlayMatchFormErrors(errs))
	}

	playerMap := map[string]string{}
	for _, winner := range winners {
		playerMap[winner] = "winner"
	}
	for _, loser := range losers {
		_, ok := playerMap[loser]
		if ok {
			errs := views.FormErrors{
				"Winners": "",
				"Losers":  "",
				"submit":  "Player cannot be both winner and loser",
			}
			return render(c, http.StatusOK, views.PlayMatchFormErrors(errs))
		}

		playerMap[loser] = "loser"
	}

	dbc := db.GetDB()
	tx := dbc.Begin()

	match := db.Match{
		GameID:    game.ID,
		CreatorID: user.ID,
	}
	if err := tx.Create(&match).Error; err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	winningUsers := []db.User{}
	err := tx.Model(&db.User{}).Where("id IN (?)", winners).Find(&winningUsers).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}
	err = tx.Model(&match).Association("Winners").Append(winningUsers)
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	losingUsers := []db.User{}
	err = tx.Model(&db.User{}).Where("id IN (?)", losers).Find(&losingUsers).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}
	err = tx.Model(&match).Association("Losers").Append(losingUsers)
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var winnersRankings, losersRankings []db.Ranking
	err = tx.Where("game_id = ? AND user_id IN (?)", game.ID, winners).First(&winnersRankings).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}
	err = tx.Where("game_id = ? AND user_id IN (?)", game.ID, losers).First(&losersRankings).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	points, expectedScore := elo.CalculatePointsGainLoss(winnersRankings, losersRankings)
	tx.Model(&match).Update("PointsValue", points).Update("ExpectedScore", expectedScore)

	tx.Commit()
	gameHome := fmt.Sprintf("/offices/%s/games/%s", game.Office.Code, gameId)
	c.Response().Header().Set("HX-Redirect", gameHome)
	return c.NoContent(http.StatusOK)
}

func gamePendingMatchesPage(c echo.Context) error {
	user := userFromContext(c)
	d := db.GetDB()

	gameId := c.Param("id")
	game := db.Game{}
	err := d.Where("id = ?", gameId).
		Preload("Office").
		Preload("Matches", "state IN (?)", "pending").
		Preload("Matches.Creator").
		Preload("Matches.Winners").
		Preload("Matches.Losers").
		Preload("Matches.Approvals").
		First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	pageContent := views.PendingMatchesPage(game, game.Office, game.Matches)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func pendingMatchPage(c echo.Context) error {
	user := userFromContext(c)
	d := db.GetDB()

	matchId := c.Param("matchId")
	match := db.Match{}
	err := d.Where("id = ?", matchId).
		Preload("Game").
		Preload("Game.Office").
		Preload("Creator").
		Preload("Winners").
		Preload("Losers").
		Preload("Approvals").
		First(&match).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	pageContent := views.PendingMatchPage(match.Game, match.Game.Office, match)
	return render(c, http.StatusOK, views.Page(pageContent, user))
}

func pendingMatchApproveHandler(c echo.Context) error {
	user := userFromContext(c)
	d := db.GetDB()

	matchId, err := strconv.Atoi(c.Param("matchId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid match ID")
	}

	approval := db.MatchApproval{
		MatchID: uint(matchId),
		UserID:  user.ID,
	}

	var count int64
	err = d.Model(approval).Where("match_id = ? AND user_id = ?", matchId, user.ID).Count(&count).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if count > 0 {
		return c.String(http.StatusBadRequest, "You have already approved this match")
	}

	err = d.Create(&approval).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	err = d.Preload("Match.Game.Office").Preload("Match.Winners").Preload("Match.Losers").Preload("Match.Approvals").Find(&approval, "id = ?", approval.ID).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if !approval.Match.IsApproved() {
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	err = d.Model(&db.Match{}).Where("id = ?", approval.Match.ID).Update("State", "approved").Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/games/%d/pending", approval.Match.Game.Office.Code, approval.Match.GameID))
	return c.NoContent(http.StatusOK)
}
