package db

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"
)

var Models = []interface{}{
	&User{},
	&Office{},
	&Game{},
	&Ranking{},
	&Match{},
	&MatchApproval{},
	&MatchParticipant{},
}

type Office struct {
	gorm.Model
	Name       string
	Code       string `gorm:"unique"`
	AdminRefer uint
	Admin      User   `gorm:"foreignKey:AdminRefer"`
	Players    []User `gorm:"many2many:user_offices;"`
	Games      []Game
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
	// Generate a code for the office
	err = tx.Model(&o).Update("Code", generateCode()).Error
	if err != nil {
		return
	}
	// Create the default game
	err = tx.Model(&o).Association("Games").Append(&Game{Name: "Default Game"})
	if err != nil {
		return
	}

	return
}

const (
	GameTypeHeadToHead       = "head_to_head"
	GameTypeWinnersAndLosers = "winners_and_losers"
	// Need to add support for this
	// GameTypeOrderedResult = "ordered_result"
)

type GameType struct {
	Value   string
	Display string
}

var GameTypes = []GameType{
	{Value: GameTypeHeadToHead, Display: "Head to Head"},
	{Value: GameTypeWinnersAndLosers, Display: "Winners and Losers"},
}

type Game struct {
	gorm.Model
	Name            string
	OfficeID        uint
	Office          Office
	Rankings        []Ranking
	Matches         []Match
	GameType        string `gorm:"default:'head_to_head'"`
	MinParticipants int    `gorm:"default:2"`
	MaxParticipants int    `gorm:"default:4"`
}

func (g *Game) AfterCreate(tx *gorm.DB) (err error) {
	// Create a ranking for each player in the office
	office := Office{}
	tx.Where("id = ?", g.OfficeID).Preload("Players").First(&office)

	var initPlayerRankings []Ranking
	for _, user := range office.Players {
		if user.NonPlayer {
			continue
		}

		initPlayerRankings = append(initPlayerRankings, Ranking{UserID: user.ID})
	}
	err = tx.Model(&g).Association("Rankings").Append(initPlayerRankings)
	return
}

type Ranking struct {
	gorm.Model
	Points int `gorm:"default:400"`
	GameID uint
	Game   Game
	UserID uint
	User   User
}

func (r *Ranking) AfterCreate(tx *gorm.DB) (err error) {
	user := User{}
	err = tx.Where("id = ?", r.UserID).First(&user).Error
	if err != nil {
		return
	}

	if user.NonPlayer {
		err = errors.New("Non-players cannot be ranked")
		return
	}

	return
}

const (
	MatchResultWin  = "win"
	MatchResultLoss = "loss"
)

type MatchParticipant struct {
	gorm.Model
	UserID        uint
	User          User
	MatchID       uint
	Match         Match
	Result        string
	StartingElo   int
	CalculatedElo int
	AppliedElo    int
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
	MatchStatePending  = "pending"
	MatchStateApproved = "approved"
)

type Match struct {
	gorm.Model
	GameID       uint
	Game         Game
	CreatorID    uint
	Creator      User
	Participants []MatchParticipant
	State        string `gorm:"default:'pending'"`
	Approvals    []MatchApproval
	Note         string
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
	adminUserId := m.Game.Office.AdminRefer

	// Don't allow admin's to "super" approve their own matches
	for _, participant := range m.Participants {
		if participant.UserID == adminUserId {
			return false
		}
	}

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
