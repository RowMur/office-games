package db

import (
	"math/rand"

	"github.com/RowMur/office-games/internal/elo"
	"gorm.io/gorm"
)

var models = []interface{}{
	&User{},
	&Office{},
	&Game{},
	&Ranking{},
	&Match{},
	&Team{},
	&TeamMatch{},
}

type User struct {
	gorm.Model
	Username        string `gorm:"unique"`
	Email           string `gorm:"unique"`
	Password        string
	Offices         []Office `gorm:"many2many:user_offices;"`
	Rankings        []Ranking
	MatchesAsWinner []Match `gorm:"foreignKey:WinnerID;references:ID;"`
	MatchesAsLoser  []Match `gorm:"foreignKey:LoserID;references:ID;"`
	Teams           []Team  `gorm:"many2many:team_players;"`
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
	GameID               uint
	Game                 Game
	WinnerID             uint
	Winner               User `gorm:"foreignKey:WinnerID"`
	WinnerStartingPoints int
	WinnerGainedPoints   int
	LoserID              uint
	Loser                User `gorm:"foreignKey:LoserID"`
	LoserStartingPoints  int
	LoserLostPoints      int
	ExpectedScore        float64
}

func (m *Match) AfterCreate(tx *gorm.DB) (err error) {
	// Update the rankings of the players
	game := Game{}
	tx.Where("id = ?", m.GameID).Preload("Rankings").First(&game)

	var winnerRanking, loserRanking Ranking
	tx.Where("game_id = ? AND user_id = ?", m.GameID, m.WinnerID).First(&winnerRanking)
	tx.Where("game_id = ? AND user_id = ?", m.GameID, m.LoserID).First(&loserRanking)

	// Calculate the new rankings
	points, expectedScore := elo.CalculatePointsGainLoss(winnerRanking.Points, loserRanking.Points)

	const matchesWithDoublePoints = 20
	var winnerMatchCount, loserMatchCount int64
	tx.Table("matches").Where("winner_id = ? OR loser_id = ?", m.WinnerID, m.WinnerID).Count(&winnerMatchCount)
	tx.Table("matches").Where("winner_id = ? OR loser_id = ?", m.LoserID, m.LoserID).Count(&loserMatchCount)

	var winnerNewPoints, loserNewPoints int

	if winnerMatchCount > matchesWithDoublePoints {
		winnerNewPoints = winnerRanking.Points + points
	} else {
		winnerNewPoints = winnerRanking.Points + (2 * points)
	}

	if loserMatchCount > matchesWithDoublePoints {
		loserNewPoints = loserRanking.Points - points
	} else {
		loserNewPoints = loserRanking.Points - (2 * points)
	}

	if loserNewPoints < 200 {
		loserNewPoints = 200
	}

	// Fill in the match point details
	tx.Model(&m).
		Update("WinnerStartingPoints", winnerRanking.Points).
		Update("WinnerGainedPoints", winnerNewPoints-winnerRanking.Points).
		Update("LoserStartingPoints", loserRanking.Points).
		Update("LoserLostPoints", loserRanking.Points-loserNewPoints).
		Update("ExpectedScore", expectedScore)

	// Update the rankings
	tx.Model(&winnerRanking).Update("Points", winnerNewPoints)
	tx.Model(&loserRanking).Update("Points", loserNewPoints)

	return
}

type Team struct {
	gorm.Model
	Players []User `gorm:"many2many:team_players;"`
}

type TeamMatch struct {
	gorm.Model
	GameID   uint
	Game     Game
	WinnerID uint
	Winner   Team `gorm:"foreignKey:WinnerID"`
	LoserID  uint
	Loser    Team `gorm:"foreignKey:LoserID"`
}
