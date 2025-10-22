package http

import (
	"github.com/labstack/echo/v4"
)

type CursorPagination struct {
	Prev         string `json:"prev" example:"abc"`
	Next         string `json:"next" example:"cba"`
	TotalEntries int    `json:"totalEntries" example:"100"`
}

type PaginateableContent[ModelOut any] interface {
	GetCursor() string
	ToModelResponse() ModelOut
}

func isForward(c echo.Context) bool {
	return c.QueryParam("nextCursor") != ""
}

func isBackward(c echo.Context) bool {
	return !isForward(c) && c.QueryParam("prevCursor") != ""
}

func NewCursorPagination[ModelOut any, S ~[]E, E PaginateableContent[ModelOut]](c echo.Context, collections S, hasMorePages bool, totalEntries int) CursorPagination {
	var prevCursor, nextCursor string
	if len(collections) > 0 {
		if isBackward(c) || hasMorePages {
			nextCursor = collections[len(collections)-1].GetCursor()
		}

		if isForward(c) || (hasMorePages && isBackward(c)) {
			prevCursor = collections[0].GetCursor()
		}
	}

	return CursorPagination{
		Prev:         prevCursor,
		Next:         nextCursor,
		TotalEntries: totalEntries,
	}
}
