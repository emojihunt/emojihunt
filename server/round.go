package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/emojihunt/emojihunt/db"
	"github.com/labstack/echo/v4"
	"github.com/mattn/go-sqlite3"
	"github.com/rivo/uniseg"
)

func (s *Server) ListRounds(c echo.Context) error {
	rounds, err := s.db.ListRounds(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rounds)
}

func (s *Server) GetRound(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	round, err := s.db.GetRound(c.Request().Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return echo.NewHTTPError(http.StatusNotFound, "no such round")
	} else if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}

type CreateRoundParams struct {
	Name  string `form:"name"`
	Emoji string `form:"emoji"`
}

func (s *Server) CreateRound(c echo.Context) error {
	var params CreateRoundParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	if params.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	} else if params.Emoji == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "emoji is required")
	} else if uniseg.GraphemeClusterCount(params.Emoji) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "emoji must be a single grapheme cluster")
	} else if uniseg.StringWidth(params.Emoji) != 2 {
		// *almost* correct, see https://github.com/rivo/uniseg/issues/27
		return echo.NewHTTPError(http.StatusBadRequest, "emoji must be an emoji")
	}

	round, err := s.db.CreateRound(c.Request().Context(), params.Name, params.Emoji)
	if c := db.ErrorCode(err); c == sqlite3.ErrConstraintUnique {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}

func (s *Server) UpdateRound(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	round, err := s.db.GetRound(c.Request().Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return echo.NewHTTPError(http.StatusNotFound, "no such round")
	} else if err != nil {
		return err
	}

	params, err := c.FormParams()
	if err != nil {
		return err
	}
outer:
	for k, vs := range params {
		t := reflect.TypeOf(&round).Elem()
		v := reflect.ValueOf(&round).Elem()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).Tag.Get("json") == k {
				v.Field(i).Set(reflect.ValueOf(vs[0]))
				continue outer
			}
		}
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid paramter: %s", k))
	}

	err = s.db.UpdateRound(c.Request().Context(), round)
	if c := db.ErrorCode(err); c == sqlite3.ErrConstraintUnique {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}
