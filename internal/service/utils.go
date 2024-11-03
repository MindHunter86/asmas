package service

import (
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const HTTPAccessLogLevel = zerolog.InfoLevel

// TODO 2delete
// I think this block of code is not profitable
// so may be it must be reverted

var ferrPool = sync.Pool{
	New: func() interface{} {
		return &fiber.Error{}
	},
}

func AcquireFErr() *fiber.Error {
	return ferrPool.Get().(*fiber.Error)
}

func ReleaseFErr(e *fiber.Error) {
	ferrPool.Put(e)
}

func rlog(c *fiber.Ctx) *zerolog.Logger {
	return c.Locals("logger").(*zerolog.Logger)
}

func rdebugf(c *fiber.Ctx, format string, opts ...interface{}) {
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		return
	}

	rlog(c).Debug().Msgf(format, opts...)
}

func respondPlainWithStatus(c *fiber.Ctx, status int) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.SendStatus(status)
}
