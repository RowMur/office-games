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
		Preload("Matches", "state NOT IN (?)", db.MatchStatePending, func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Preload("Matches.Participants.User").
		Preload("Matches.Creator").
		First(&game).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	playerWinLosses := map[uint]views.WinLosses{}
	for _, match := range game.Matches {
		for _, participant := range match.Participants {
			winCount := playerWinLosses[participant.UserID].Wins
			lossCount := playerWinLosses[participant.UserID].Losses

			if participant.Result == "win" {
				winCount++
			} else {
				lossCount++
			}

			playerWinLosses[participant.UserID] = views.WinLosses{
				Wins:   winCount,
				Losses: lossCount,
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

	participantCount := len(winners) + len(losers)
	if participantCount < game.MinParticipants || participantCount > game.MaxParticipants {
		errs := views.FormErrors{
			"Winners": "",
			"Losers":  "",
			"submit":  fmt.Sprintf("There must be between %d and %d players selected", game.MinParticipants, game.MaxParticipants),
		}
		return render(c, http.StatusOK, views.PlayMatchFormErrors(errs))
	}

	if game.GameType == db.GameTypeHeadToHead && len(winners) != len(losers) {
		errs := views.FormErrors{
			"Winners": "",
			"Losers":  "",
			"submit":  "Winners and Losers must be of equal number",
		}
		return render(c, http.StatusOK, views.PlayMatchFormErrors(errs))
	}

	playerMap := map[string]string{}
	for _, winner := range winners {
		playerMap[winner] = "win"
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

		playerMap[loser] = "loss"
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

	participantUserIds := []string{}
	participantUserIds = append(participantUserIds, winners...)
	participantUserIds = append(participantUserIds, losers...)

	participantRankings := []db.Ranking{}
	err := tx.Where("game_id = ? AND user_id IN (?)", game.ID, participantUserIds).Find(&participantRankings).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

	winnerRankings := []db.Ranking{}
	loserRankings := []db.Ranking{}
	for _, ranking := range participantRankings {
		stringUserId := strconv.Itoa(int(ranking.UserID))
		if playerMap[stringUserId] == "win" {
			winnerRankings = append(winnerRankings, ranking)
		} else {
			loserRankings = append(loserRankings, ranking)
		}
	}

	participants := []db.MatchParticipant{}
	for _, participantUserId := range participantUserIds {
		intUserId, err := strconv.Atoi(participantUserId)
		if err != nil {
			tx.Rollback()
			return c.String(http.StatusInternalServerError, err.Error())
		}
		participant := db.MatchParticipant{
			UserID:  uint(intUserId),
			MatchID: match.ID,
			Result:  playerMap[participantUserId],
		}

		participantRanking := db.Ranking{}
		for _, ranking := range participantRankings {
			if ranking.UserID == participant.UserID {
				participantRanking = ranking
				break
			}
		}
		participant.StartingElo = participantRanking.Points

		if participant.Result == "win" {
			multiplier := 1.0
			if len(winners) > len(losers) {
				multiplier = float64(len(losers)) / float64(len(winners))
			}
			calcElo := elo.CalculatePointsGainLoss([]db.Ranking{participantRanking}, loserRankings, multiplier)
			if len(winners) > len(losers) {
				// When the multiplier is applied to a side, each player on that side gets slightly shortchanged due to the rounding
				// E.g. 10 points net gain/loss split in a game with 3 winners and 1 loser
				// The winners earn 3.33 points each (rounded to 3) and the loser loses 10 points
				// To avoid a system wide net loss of ELO just add one to each of the winners
				calcElo++
			}
			participant.CalculatedElo = calcElo
		} else {
			multiplier := 1.0
			if len(losers) > len(winners) {
				multiplier = float64(len(winners)) / float64(len(losers))
			}
			calcElo := elo.CalculatePointsGainLoss(winnerRankings, []db.Ranking{participantRanking}, multiplier)
			participant.CalculatedElo = -calcElo
		}

		participants = append(participants, participant)
	}

	err = tx.Create(&participants).Error
	if err != nil {
		tx.Rollback()
		return c.String(http.StatusInternalServerError, err.Error())
	}

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
		Preload("Matches", "state IN (?)", db.MatchStatePending, func(db *gorm.DB) *gorm.DB {
			return db.Order("Matches.created_at DESC")
		}).
		Preload("Matches.Creator").
		Preload("Matches.Participants.User").
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
		Preload("Participants.User").
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
		Preload("Match.Participants").
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

	err = tx.Model(&db.Match{}).Where("id = ?", approval.Match.ID).Update("State", db.MatchStateApproved).Error
	if err != nil {
		tx.Rollback()
		return render(c, http.StatusOK, views.MatchApproveError(err.Error()))
	}

	// Update the rankings of the players
	const matchesWithDoublePoints = 20
	queryForGameMatches := tx.Select("id").Where("game_id = ?", approval.Match.GameID).Table("matches")

	for _, participant := range approval.Match.Participants {
		var matchesPlayed int64
		err := tx.Model(&db.MatchParticipant{}).Where("user_id = ? AND match_id IN (?)", participant.UserID, queryForGameMatches).Count(&matchesPlayed).Error
		if err != nil {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Error counting matches"))
		}

		var ranking db.Ranking
		for _, r := range approval.Match.Game.Rankings {
			if r.UserID == participant.UserID {
				ranking = r
				break
			}
		}
		if ranking.ID == 0 {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Player not recognised"))
		}

		appliedElo := participant.CalculatedElo
		if matchesPlayed <= matchesWithDoublePoints {
			appliedElo *= 2
		}
		if ranking.Points+appliedElo < 200 {
			appliedElo = ranking.Points - 200
		}

		err = tx.Model(&participant).Update("AppliedElo", appliedElo).Error
		if err != nil {
			tx.Rollback()
			return render(c, http.StatusOK, views.MatchApproveError("Error updating match participant"))
		}

		newElo := ranking.Points + appliedElo
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
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, game))
	}

	minParticipants, err := strconv.Atoi(newMinParticipants)
	if err != nil {
		errs := views.FormErrors{"min-participants": "Min participants must be a number"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, game))
	}

	maxParticipants, err := strconv.Atoi(newMaxParticipants)
	if err != nil {
		errs := views.FormErrors{"max-participants": "Max participants must be a number"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, game))
	}

	if minParticipants > maxParticipants {
		errs := views.FormErrors{"min-participants": "Min participants must be less than max participants"}
		return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, game))
	}

	for i, gt := range db.GameTypes {
		if gt.Value == gameType {
			break
		}

		if i == len(db.GameTypes)-1 {
			errs := views.FormErrors{"game-type": "Invalid game type"}
			return render(c, http.StatusOK, views.EditGameForm(formData, errs, office, game))
		}
	}

	err = d.Model(&game).Where("id = ?", game.ID).Updates(map[string]interface{}{"name": newName, "min_participants": minParticipants, "max_participants": maxParticipants, "game_type": gameType}).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Refresh", "true")
	return c.NoContent(http.StatusOK)
}
