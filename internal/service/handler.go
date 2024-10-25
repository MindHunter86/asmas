package service

import (
	"errors"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func (*Service) fiberDefaultErrorHandler(c *fiber.Ctx, err error) error {
	// reject invalid requests
	if strings.TrimSpace(c.Hostname()) == "" {
		gLog.Warn().Msg("invalid request from " + c.Context().RemoteIP().String())
		gLog.Debug().Msgf("invalid request: %+v ; error - %+v", c, err)
		return c.Context().Conn().Close()
	}

	// apiv1 error style:
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	// `rspcode` - apiv1 legacy hardcode
	// if u have 4XX or 5XX in service, u must respond with 200
	rspcode := fiber.StatusOK

	// ? not profitable
	// TODO too much allocations here:
	ferr := AcquireFErr()
	defer ReleaseFErr(ferr)

	errdesc := "error provided by " + c.App().Server().Name + " service"

	// parse fiber error
	if !errors.As(err, &ferr) {
		respond(fiber.StatusInternalServerError, err.Error(), errdesc)
		return c.SendStatus(rspcode)
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		rlog(c).Debug().Msgf("%+v", err)
	}

	respond(ferr.Code, ferr.Error(), errdesc)
	return c.SendStatus(rspcode)
}

// TODO 2delete
// I think this block of code is not profitable
// so may be it must be reverted

var ferrPool = sync.Pool{
	New: func() interface{} {
		return new(fiber.Error)
	},
}

func AcquireFErr() *fiber.Error {
	return ferrPool.Get().(*fiber.Error)
}

func ReleaseFErr(e *fiber.Error) {
	// ? is it required
	e.Code, e.Message = 0, ""
	ferrPool.Put(e)
}
