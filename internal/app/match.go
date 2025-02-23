package app

import (
	"errors"
	"strconv"

	"github.com/RowMur/office-table-tennis/internal/db"
	"gorm.io/gorm"
)

func (a *App) GetMatchById(id string) (*db.Match, error) {
	match := db.Match{}
	err := a.db.C.Preload("Office").
		Preload("Participants.User").
		Preload("Creator").
		Preload("Approvals").
		First(&match, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &match, nil
}

func (a *App) LogMatch(creator *db.User, office *db.Office, note string, winners, losers []string, isHandicap bool) (*db.Match, error) {
	tx := a.db.C.Begin()

	match := db.Match{
		OfficeID:   office.ID,
		CreatorID:  creator.ID,
		Note:       note,
		IsHandicap: isHandicap,
	}
	if err := tx.Create(&match).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	participants := []db.MatchParticipant{}
	for _, winner := range winners {
		userId, err := strconv.Atoi(winner)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		participants = append(participants, db.MatchParticipant{
			UserID:  uint(userId),
			MatchID: match.ID,
			Result:  db.MatchResultWin,
		})
	}
	for _, loser := range losers {
		userId, err := strconv.Atoi(loser)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		participants = append(participants, db.MatchParticipant{
			UserID:  uint(userId),
			MatchID: match.ID,
			Result:  db.MatchResultLoss,
		})
	}

	err := tx.Create(&participants).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &match, nil
}

func (a *App) ScheduleMatch(tx *gorm.DB, creator *db.User, office db.Office, tournament *db.Tournament, firstSideParticipants, secondSideParticipants []uint, nextMatch *db.Match) (*db.Match, error) {
	match := db.Match{
		CreatorID:  creator.ID,
		State:      db.MatchStateScheduled,
		Office:     office,
		Tournament: tournament,
		NextMatch:  nextMatch,
	}
	err := tx.Create(&match).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	participants := []db.MatchParticipant{}
	for _, firstSideParticipant := range firstSideParticipants {
		participants = append(participants, db.MatchParticipant{
			UserID:  firstSideParticipant,
			MatchID: match.ID,
			Result:  db.MatchResultWin,
		})
	}
	for _, secondSideParticipant := range secondSideParticipants {
		participants = append(participants, db.MatchParticipant{
			UserID:  secondSideParticipant,
			MatchID: match.ID,
			Result:  db.MatchResultLoss,
		})
	}

	if len(participants) == 0 {
		return &match, nil
	}

	err = tx.Create(&participants).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return &match, nil
}

func (a *App) ApproveMatch(user *db.User, match *db.Match) error {
	if match.State != db.MatchStatePending {
		return errors.New("match is not pending")
	}

	var count int64
	err := a.db.C.Model(&db.MatchApproval{}).Where("match_id = ? AND user_id = ?", match.ID, user.ID).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("already approved")
	}

	tx := a.db.C.Begin()

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
	err := tx.Preload("Office").
		Preload("Participants").
		Preload("Approvals").
		Find(&match, "id = ?", match.ID).Error
	if err != nil {
		return false, err
	}

	return match.IsApproved(), nil
}

func (a *App) processApprovedMatch(tx *gorm.DB, match *db.Match) error {
	err := tx.Preload("Office").Find(&match, "id = ?", match.ID).Error
	if err != nil {
		return err
	}

	err = tx.Model(&db.Match{}).Where("id = ?", match.ID).Update("State", db.MatchStateApproved).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	a.op.InvalidateOfficeCache(match.OfficeID)
	return nil
}
