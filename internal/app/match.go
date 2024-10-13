package app

import (
	"errors"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/elo"
	"gorm.io/gorm"
)

func (a *App) GetMatchById(id string) (*db.Match, error) {
	match := db.Match{}
	err := a.db.Preload("Game.Office").
		Preload("Participants.User").
		Preload("Creator").
		Preload("Approvals").
		First(&match, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &match, nil
}

func (a *App) LogMatch(creator *db.User, game *db.Game, note string, winners, losers []string) (*db.Match, error) {
	tx := a.db.Begin()

	match := db.Match{
		GameID:    game.ID,
		CreatorID: creator.ID,
		Note:      note,
	}
	if err := tx.Create(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	winnerRankings := []db.Ranking{}
	err := tx.Where("game_id = ? AND user_id IN (?)", game.ID, winners).Find(&winnerRankings).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(winnerRankings) != len(winners) {
		return nil, errors.New("winner not recognized")
	}

	loserRankings := []db.Ranking{}
	err = tx.Where("game_id = ? AND user_id IN (?)", game.ID, losers).Find(&loserRankings).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(loserRankings) != len(losers) {
		return nil, errors.New("loser not recognized")
	}

	participants := []db.MatchParticipant{}
	for _, ranking := range winnerRankings {
		participant := db.MatchParticipant{
			UserID:      ranking.UserID,
			MatchID:     match.ID,
			Result:      db.MatchResultWin,
			StartingElo: ranking.Points,
		}

		multiplier := 1.0
		if len(winners) > len(losers) {
			multiplier = float64(len(losers)) / float64(len(winners))
		}
		calcElo := elo.CalculatePointsGainLoss([]db.Ranking{ranking}, loserRankings, multiplier)
		if len(winners) > len(losers) {
			// When the multiplier is applied to a side, each player on that side gets slightly shortchanged due to the rounding
			// E.g. 10 points net gain/loss split in a game with 3 winners and 1 loser
			// The winners earn 3.33 points each (rounded to 3) and the loser loses 10 points
			// To avoid a system wide net loss of ELO just add one to each of the winners
			calcElo++
		}
		participant.CalculatedElo = calcElo
		participants = append(participants, participant)
	}
	for _, ranking := range loserRankings {
		participant := db.MatchParticipant{
			UserID:      ranking.UserID,
			MatchID:     match.ID,
			Result:      db.MatchResultLoss,
			StartingElo: ranking.Points,
		}

		multiplier := 1.0
		if len(losers) > len(winners) {
			multiplier = float64(len(winners)) / float64(len(losers))
		}
		calcElo := elo.CalculatePointsGainLoss(winnerRankings, []db.Ranking{ranking}, multiplier)
		participant.CalculatedElo = -calcElo
		participants = append(participants, participant)
	}

	err = tx.Create(&participants).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &match, nil
}

func (a *App) ApproveMatch(user *db.User, match *db.Match) error {
	if match.State != db.MatchStatePending {
		return errors.New("match is not pending")
	}

	var count int64
	err := a.db.Model(&db.MatchApproval{}).Where("match_id = ? AND user_id = ?", match.ID, user.ID).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("already approved")
	}

	tx := a.db.Begin()

	approval := db.MatchApproval{
		MatchID: match.ID,
		UserID:  user.ID,
	}
	err = tx.Create(&approval).Error
	if err != nil {
		tx.Rollback()
		return errors.New("error creating approval")
	}

	isApproved, err := a.IsMatchApproved(tx, match)
	if err != nil {
		tx.Rollback()
		return err
	}
	if !isApproved {
		tx.Commit()
		return nil
	}

	err = a.processApprovedMatch(tx, match)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (a *App) IsMatchApproved(tx *gorm.DB, match *db.Match) (bool, error) {
	err := tx.Preload("Game.Office").
		Preload("Participants").
		Preload("Approvals").
		Find(&match, "id = ?", match.ID).Error
	if err != nil {
		return false, err
	}

	return match.IsApproved(), nil
}

func (a *App) processApprovedMatch(tx *gorm.DB, match *db.Match) error {
	err := tx.Preload("Game.Office").
		Preload("Participants").
		Preload("Game.Rankings").
		Find(&match, "id = ?", match.ID).Error
	if err != nil {
		return err
	}

	err = tx.Model(&db.Match{}).Where("id = ?", match.ID).Update("State", db.MatchStateApproved).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update the rankings of the players
	const matchesWithDoublePoints = 20
	queryForGameMatches := tx.Select("id").Where("game_id = ?", match.GameID).Table("matches")

	for _, participant := range match.Participants {
		var matchesPlayed int64
		err := tx.Model(&db.MatchParticipant{}).Where("user_id = ? AND match_id IN (?)", participant.UserID, queryForGameMatches).Count(&matchesPlayed).Error
		if err != nil {
			tx.Rollback()
			return errors.New("error counting matches played")
		}

		var ranking db.Ranking
		for _, r := range match.Game.Rankings {
			if r.UserID == participant.UserID {
				ranking = r
				break
			}
		}
		if ranking.ID == 0 {
			tx.Rollback()
			return errors.New("player not recognised")
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
			return errors.New("error updating elo")
		}

		newElo := ranking.Points + appliedElo
		err = tx.Model(&ranking).Update("Points", newElo).Error
		if err != nil {
			tx.Rollback()
			return errors.New("error updating elo")
		}
	}

	return nil
}
