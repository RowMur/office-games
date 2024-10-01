package db

import (
	"math/rand"

	"gorm.io/gorm"
)

var models = []interface{}{
	&User{},
	&Office{},
	&Game{},
	&Ranking{},
	&Match{},
	&MatchApproval{},
}

type User struct {
	gorm.Model
	Username        string `gorm:"unique"`
	Email           string `gorm:"unique"`
	Password        string
	Offices         []Office `gorm:"many2many:user_offices;"`
	Rankings        []Ranking
	MatchesAsWinner []Match `gorm:"many2many:match_winners;"`
	MatchesAsLoser  []Match `gorm:"many2many:match_losers;"`
	Approvals       []MatchApproval
}

type Office struct {
	gorm.Model
	Name       string
	Code       string
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
	tx.Where("id = ?", o.AdminRefer).First(&user)
	tx.Model(&o).Association("Players").Append(&user)
	// Generate a code for the office
	tx.Model(&o).Update("Code", generateCode())
	// Create the default game
	tx.Model(&o).Association("Games").Append(&Game{Name: "Default Game"})
	return
}

type Game struct {
	gorm.Model
	Name     string
	OfficeID uint
	Office   Office
	Rankings []Ranking
	Matches  []Match
}

func (g *Game) AfterCreate(tx *gorm.DB) (err error) {
	// Create a ranking for each player in the office
	office := Office{}
	tx.Where("id = ?", g.OfficeID).Preload("Players").First(&office)

	var initPlayerRankings []Ranking
	for _, user := range office.Players {
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

type Match struct {
	gorm.Model
	GameID        uint
	Game          Game
	CreatorID     uint
	Creator       User
	Winners       []User `gorm:"many2many:match_winners;"`
	Losers        []User `gorm:"many2many:match_losers;"`
	PointsValue   int
	ExpectedScore float64
	State         string `gorm:"default:'pending'"`
	Approvals     []MatchApproval
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
	for _, winner := range m.Winners {
		if m.IsApprovedByUser(winner.ID) {
			return true
		}
	}
	return false
}

func (m *Match) IsApprovedByLosers() bool {
	for _, loser := range m.Losers {
		if m.IsApprovedByUser(loser.ID) {
			return true
		}
	}
	return false
}

func (m *Match) IsApproved() bool {
	return m.IsApprovedByWinners() && m.IsApprovedByLosers()
}

type MatchApproval struct {
	gorm.Model
	MatchID uint
	Match   Match `gorm:"foreignKey:MatchID"`
	UserID  uint
	User    User
}
