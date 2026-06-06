package middleware

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var multiSlash = regexp.MustCompile(`/+`)

// NormalizePath fixes double slashes from reverse proxy (e.g. //health → /health).
func NormalizePath() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := string(c.Request().URI().Path())
		if !strings.Contains(path, "//") {
			return c.Next()
		}

		normalized := multiSlash.ReplaceAllString(path, "/")
		if normalized != path {
			c.Request().URI().SetPath(normalized)
		}
		return c.Next()
	}
}
