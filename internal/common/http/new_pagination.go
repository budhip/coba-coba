package http

import (
	"github.com/gofiber/fiber/v2"
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

func isForward(c *fiber.Ctx) bool {
	return c.Query("nextCursor") != ""
}

func isBackward(c *fiber.Ctx) bool {
	return !isForward(c) && c.Query("prevCursor") != ""
}

func NewCursorPagination[ModelOut any, S ~[]E, E PaginateableContent[ModelOut]](c *fiber.Ctx, collections S, hasMorePages bool, totalEntries int) CursorPagination {
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
