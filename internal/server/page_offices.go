package server

import (
	"fmt"
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func (s *Server) officeHandler(c echo.Context) error {
	officeCode := c.Param("code")
	office, err := s.app.GetOfficeByCode(officeCode)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if office == nil {
		return c.String(http.StatusNotFound, "Office not found")
	}

	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officePageContent := views.OfficePage(*office, user)
	return render(c, http.StatusOK, views.Page(officePageContent, user))
}

func (s *Server) joinOfficeHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officeCode := c.FormValue("office")
	userErr, err := s.app.JoinOffice(user, officeCode)
	if userErr != nil {
		formData := views.FormData{"office": officeCode}
		errs := views.FormErrors{"office": userErr.Error()}
		return render(c, http.StatusOK, views.JoinOfficeForm(formData, errs))
	}
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", fmt.Sprintf("/offices/%s", officeCode))
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) createOfficeHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
	}

	officeName := c.FormValue("office")
	if officeName == "" {
		errs := views.FormErrors{"office": "Office name is required"}
		return render(c, http.StatusOK, views.CreateOfficeForm(views.FormData{}, errs))
	}

	_, err := s.app.CreateOffice(user, officeName)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("HX-Redirect", "/")
	return c.NoContent(http.StatusNoContent)
}
