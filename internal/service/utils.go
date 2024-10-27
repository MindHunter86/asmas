package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func rlog(c *fiber.Ctx) *zerolog.Logger {
	return c.Locals("logger").(*zerolog.Logger)
}

func respond(status int, msg, desc string) {
	// !!!
	// !!!
	// !!!
	// !!!
	// !!!
	// if e := utils.RespondWithApiError(status, msg, desc, c); e != nil {
	// 	rlog(c).Error().Msg("could not respond with JSON error - " + e.Error())
	// }
}
