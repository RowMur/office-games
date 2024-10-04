package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/elo"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func gamesPageHandler(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	d := db.GetDB()
	game := db.Game{}
	err := d.Where("id = ?", gameId).
		Preload("Office").
		Preload("Rankings", func(db *gorm.DB) *gorm.DB {
			return db.Order("Rankings.points DESC")
		}).
		Preload("Rankings.User").
		Preload("Matches", "state NOT IN (?)", "pending", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Matches.Winners").
		Preload("Matches.Losers").
		First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	playerWinLosses := map[uint]views.WinLosses{}
	for _, match := range game.Matches {
		for _, winner := range match.Winners {
			playerWinLosses[winner.ID] = views.WinLosses{
				Wins:   playerWinLosses[winner.ID].Wins + 1,
				Losses: playerWinLosses[winner.ID].Losses,
			}
		}
		for _, loser := range match.Losers {
			playerWinLosses[loser.ID] = views.WinLosses{
				Wins:   playerWinLosses[loser.ID].Wins,
				Losses: playerWinLosses[loser.ID].Losses + 1,
			}
		}
	}

	pageContent := views.GamePage(game, game.Office, playerWinLosses, *user)
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
	pageContent := views.PlayGamePage(game, game.Office, game.Office.Players, endpoint, *user)
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

	approval := db.MatchApproval{
		MatchID: match.ID,
		UserID:  user.ID,
	}
	err = tx.Create(&approval).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	tx.Commit()
	gameHome := fmt.Sprintf("/offices/%s/games/%s/pending/%d", game.Office.Code, gameId, match.ID)
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
		Preload("Matches", "state IN (?)", "pending", func(db *gorm.DB) *gorm.DB {
			return db.Order("Matches.created_at DESC")
		}).
		Preload("Matches.Creator").
		Preload("Matches.Winners").
		Preload("Matches.Losers").
		Preload("Matches.Approvals").
		First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	pageContent := views.PendingMatchesPage(game, game.Office, game.Matches, *user)
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

	matchId, err := strconv.Atoi(c.Param("matchId"))
	if err != nil {
		return render(c, http.StatusOK, views.MatchApproveError("Invalid match ID"))
	}

	d := db.GetDB()
	tx := d.Begin()

	approval := db.MatchApproval{
		MatchID: uint(matchId),
		UserID:  user.ID,
	}

	var count int64
	err = tx.Model(approval).Where("match_id = ? AND user_id = ?", matchId, user.ID).Count(&count).Error
	if err != nil {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}
	if count > 0 {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError("You have already approved this match"))
	}

	err = tx.Create(&approval).Error
	if err != nil {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}

	err = tx.Preload("Match.Game.Office").
		Preload("Match.Game.Rankings").
		Preload("Match.Winners").
		Preload("Match.Losers").
		Preload("Match.Approvals").
		Find(&approval, "id = ?", approval.ID).Error
	if err != nil {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}

	if !approval.Match.IsApproved() {
		tx.Commit()
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}

	err = tx.Model(&db.Match{}).Where("id = ?", approval.Match.ID).Update("State", "approved").Error
	if err != nil {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}

	// Update the rankings of the players
	const matchesWithDoublePoints = 20
	queryForGameMatches := tx.Select("id").Where("game_id = ?", approval.Match.GameID).Table("matches")

	for _, winner := range approval.Match.Winners {
		var winCount, lossCount int64
		tx.Table("match_winners").Where("user_id = ? AND match_id IN (?)", winner.ID, queryForGameMatches).Count(&winCount)
		tx.Table("match_losers").Where("user_id = ? AND match_id IN (?)", winner.ID, queryForGameMatches).Count(&lossCount)

		matchesPlayed := winCount + lossCount

		var ranking db.Ranking
		for _, r := range approval.Match.Game.Rankings {
			if r.UserID == winner.ID {
				ranking = r
				break
			}
		}
		if ranking.ID == 0 {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Player not recognised"))
		}

		var newElo int
		if matchesPlayed >= matchesWithDoublePoints {
			newElo = ranking.Points + approval.Match.PointsValue
		} else {
			newElo = ranking.Points + (2 * approval.Match.PointsValue)
		}

		err = tx.Model(&ranking).Update("Points", newElo).Error
		if err != nil {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Error updating rankings"))
		}
	}

	for _, loser := range approval.Match.Losers {
		var winCount, lossCount int64
		tx.Table("match_winners").Where("user_id = ? AND match_id IN (?)", loser.ID, queryForGameMatches).Count(&winCount)
		tx.Table("match_losers").Where("user_id = ? AND match_id IN (?)", loser.ID, queryForGameMatches).Count(&lossCount)

		matchesPlayed := winCount + lossCount

		var ranking db.Ranking
		for _, r := range approval.Match.Game.Rankings {
			if r.UserID == loser.ID {
				ranking = r
				break
			}
		}
		if ranking.ID == 0 {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Player not recognised"))
		}

		var newElo int
		if matchesPlayed >= matchesWithDoublePoints {
			newElo = ranking.Points - approval.Match.PointsValue
		} else {
			newElo = ranking.Points - (2 * approval.Match.PointsValue)
		}

		if newElo < 200 {
			newElo = 200
		}

		err = tx.Model(&ranking).Update("Points", newElo).Error
		if err != nil {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Error updating rankings"))
		}
	}

	tx.Commit()
	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/games/%d/pending", approval.Match.Game.Office.Code, approval.Match.GameID))
	return c.NoContent(http.StatusOK)
}

func gameAdminPage(c echo.Context) error {
	user := userFromContext(c)
	gameId := c.Param("id")

	d := db.GetDB()
	game := db.Game{}
	err := d.Where("id = ?", gameId).
		Preload("Office").
		First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return render(c, http.StatusOK, views.Page(views.GameAdminPage(game, game.Office, *user), user))
}

func deleteGameHandler(c echo.Context) error {
	gameId := c.Param("id")
	office := c.Param("code")

	d := db.GetDB()
	err := d.Delete(&db.Game{}, gameId).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s", office))
	return c.NoContent(http.StatusOK)
}

func editGameHandler(c echo.Context) error {
	d := db.GetDB()

	gameId := c.Param("id")
	game := db.Game{}
	err := d.Where("id = ?", gameId).First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	office := c.Param("code")
	newName := c.FormValue("name")
	if newName == "" {
		errs := views.FormErrors{"name": "Name is required"}
		return render(c, http.StatusOK, views.EditGameForm(views.FormData{}, errs, office, game))
	}
	err = d.Model(&game).Where("id = ?", game.ID).Updates(map[string]interface{}{"name": newName}).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s/games/%s", office, gameId))
	return c.NoContent(http.StatusOK)
}
