package db

import (
	"errors"
	"fmt"
	"math/rand"

	"gorm.io/gorm"
)

var Models = []interface{}{
	&User{},
	&Office{},
	&Match{},
	&MatchApproval{},
	&MatchParticipant{},
}

type Office struct {
	gorm.Model
	Name        string
	Code        string `gorm:"unique"`
	AdminRefer  uint
	Admin       User   `gorm:"foreignKey:AdminRefer"`
	Players     []User `gorm:"many2many:user_offices;"`
	Matches     []Match
	Tournaments []Tournament
}

func (o *Office) Link() string {
	return fmt.Sprintf("/offices/%s", o.Code)
}

func generateCode() string {
	lengthOfCode := 6
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	code := make([]rune, lengthOfCode)

	for i := range lengthOfCode {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func (o *Office) AfterCreate(tx *gorm.DB) (err error) {
	// Add the admin to the office players
	var user User
	err = tx.Where("id = ?", o.AdminRefer).First(&user).Error
	if err != nil {
		return
	}
	err = tx.Model(&o).Association("Players").Append(&user)
	if err != nil {
		return
	}

	if o.Code == "" {
		// Generate a code for the office
		err = tx.Model(&o).Update("Code", generateCode()).Error
		if err != nil {
			return
		}
	}

	return
}

const (
	MatchResultWin  = "win"
	MatchResultLoss = "loss"
)

type MatchParticipant struct {
	gorm.Model
	UserID  uint
	User    User
	MatchID uint
	Match   Match
	Result  string
}

func (mp *MatchParticipant) AfterCreate(tx *gorm.DB) (err error) {
	user := User{}
	err = tx.Where("id = ?", mp.UserID).First(&user).Error
	if err != nil {
		return
	}

	if user.NonPlayer {
		err = errors.New("Non-players cannot play matches")
		return
	}

	return
}

const (
	MatchStatePending   = "pending"
	MatchStateApproved  = "approved"
	MatchStateScheduled = "scheduled"
)

type Match struct {
	gorm.Model
	OfficeID     uint
	Office       Office
	CreatorID    uint
	Creator      User
	Participants []MatchParticipant
	State        string `gorm:"default:'pending'"`
	Approvals    []MatchApproval
	Note         string
	IsHandicap   bool
	TournamentID *uint
	Tournament   *Tournament
	NextMatchID  *uint
	NextMatch    *Match
}

func (m *Match) BeforeDelete(tx *gorm.DB) (err error) {
	if m.ID == 0 {
		return errors.New("Match ID is 0")
	}
	err = tx.Where("match_id = ?", m.ID).Delete(&MatchParticipant{}).Error
	if err != nil {
		return
	}
	err = tx.Where("match_id = ?", m.ID).Delete(&MatchApproval{}).Error
	if err != nil {
		return
	}
	return
}

func (m *Match) IsApprovedByUser(userID uint) bool {
	for _, approval := range m.Approvals {
		if approval.UserID == userID {
			return true
		}
	}
	return false
}

func (m *Match) IsApprovedByWinners() bool {
	for _, participant := range m.Participants {
		if participant.Result == MatchResultWin && m.IsApprovedByUser(participant.UserID) {
			return true
		}
	}
	return false
}

func (m *Match) IsApprovedByLosers() bool {
	for _, participant := range m.Participants {
		if participant.Result == MatchResultLoss && m.IsApprovedByUser(participant.UserID) {
			return true
		}
	}
	return false
}

func (m *Match) IsAdminApproved() bool {
	adminUserId := m.Office.AdminRefer

	// // Don't allow admin's to "super" approve their own matches
	// for _, participant := range m.Participants {
	// 	if participant.UserID == adminUserId {
	// 		return false
	// 	}
	// }

	for _, approval := range m.Approvals {
		if approval.UserID == adminUserId {
			return true
		}
	}

	return false
}

func (m *Match) IsApproved() bool {
	return (m.IsApprovedByWinners() && m.IsApprovedByLosers()) || m.IsAdminApproved()
}

func (m *Match) Winners() []MatchParticipant {
	var winners []MatchParticipant
	for _, participant := range m.Participants {
		if participant.Result == MatchResultWin {
			winners = append(winners, participant)
		}
	}
	return winners
}

func (m *Match) Losers() []MatchParticipant {
	var losers []MatchParticipant
	for _, participant := range m.Participants {
		if participant.Result == MatchResultLoss {
			losers = append(losers, participant)
		}
	}
	return losers
}

type MatchApproval struct {
	gorm.Model
	MatchID uint
	Match   Match `gorm:"foreignKey:MatchID"`
	UserID  uint
	User    User
}

type Tournament struct {
	gorm.Model
	Name         string
	OfficeID     uint
	Office       Office
	Participants []User `gorm:"many2many:user_tournaments;"`
	Matches      []Match
}
