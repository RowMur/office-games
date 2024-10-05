package server

import (
	"fmt"
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm/clause"
)

func officeHandler(c echo.Context) error {
	officeCode := c.Param("code")

	office := &db.Office{}
	err := db.GetDB().Where("code = ?", officeCode).
		Preload(clause.Associations).
		First(office).Error
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			return c.String(http.StatusNotFound, "Office not found")
		}

		return c.String(http.StatusInternalServerError, err.Error())
	}

	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	selectedGame := office.Games[0]

	officePageContent := views.OfficePage(*office, user, selectedGame)
	return render(c, http.StatusOK, views.Page(officePageContent, user))
}

func joinOfficeHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officeCode := c.FormValue("office")
	if officeCode == "" {
		errs := views.FormErrors{"office": "Office code is required"}
		return render(c, http.StatusBadRequest, views.JoinOfficeForm(views.FormData{}, errs))
	}

	office := &db.Office{}
	err := db.GetDB().Where("code = ?", officeCode).Preload("Players").Preload("Games").First(office).Error
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			data := views.FormData{"office": officeCode}
			errs := views.FormErrors{"office": "Office not found"}
			return render(c, http.StatusNotFound, views.JoinOfficeForm(data, errs))
		}

		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Check if user is already in the office
	for _, player := range office.Players {
		if player.ID == user.ID {
			data := views.FormData{"office": officeCode}
			errs := views.FormErrors{"office": "You are already in this office"}
			return render(c, http.StatusBadRequest, views.JoinOfficeForm(data, errs))
		}
	}

	err = db.GetDB().Model(office).Association("Players").Append(user)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var initRankingsForEachOfficeGame []db.Ranking
	for _, game := range office.Games {
		initRankingsForEachOfficeGame = append(initRankingsForEachOfficeGame, db.Ranking{UserID: user.ID, GameID: game.ID})
	}
	if len(initRankingsForEachOfficeGame) > 0 {
		err = db.GetDB().Model(user).Association("Rankings").Append(initRankingsForEachOfficeGame)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s", office.Code))
	return c.NoContent(http.StatusNoContent)
}

func createOfficeHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officeName := c.FormValue("office")
	if officeName == "" {
		errs := views.FormErrors{"office": "Office name is required"}
		return render(c, http.StatusBadRequest, views.CreateOfficeForm(views.FormData{}, errs))
	}

	newOffice := &db.Office{Name: officeName, AdminRefer: user.ID}
	err := db.GetDB().Create(newOffice).Error
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", "/")
	return c.NoContent(http.StatusNoContent)
}
