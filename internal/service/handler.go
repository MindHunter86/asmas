package service

import (
	"errors"
	"strings"
	"sync"

	"github.com/MindHunter86/asmas/internal/auth"
	"github.com/MindHunter86/asmas/internal/system"
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
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

//
//
//

const (
	RegistrationArgHostname = "hostname"
	RegistrationArgSign     = "sign"
)

// !!!! REQUEST VALIDATION
// Check Content-Type

// Base request validation
func middlewareAuthentification(c *fiber.Ctx) error {
	var hostname string
	if hostname = c.Query(RegistrationArgHostname); hostname == "" {
		rdebugf(c, "hostname : %s", hostname)

		rlog(c).Error().Msg("decline request wo hostname argument")
		return fiber.NewError(fiber.StatusBadRequest)
	}
	c.Locals(auth.LKeyHostname, hostname)

	var sign string
	if sign = c.Query(RegistrationArgSign); sign == "" {
		rdebugf(c, "sign : %s", sign)

		rlog(c).Error().Msg("decline request wo sign argument")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	aservice := c.UserContext().Value(utils.CKeyAuthService).(*auth.AuthService)

	var payload []byte
	payloadlen := len(c.IP()) + len(c.Request().String()) + len(hostname)
	if payload = aservice.PrepareHMACMessage(payloadlen, c.IP(), c.Path(), hostname); payload == nil {
		rdebugf(c, "chunks: %s | %s | %s", c.IP(), c.Path(), hostname)

		rlog(c).Error().Msg("unexpected result while preparing message for sign verification")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	if expect, ok := aservice.VerifyHMACSign(payload, futils.UnsafeBytes(sign)); !ok {
		rdebugf(c, "chunks : %s | %s | %s", c.IP(), c.Path(), hostname)
		rdebugf(c, "recevied sign %s, expect %s", sign, expect)

		rlog(c).Error().Msg("decline request with unverified hmac sign")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return c.Next()
}

// Variables authorization with Github config
func middlewareAuthorization(c *fiber.Ctx) error {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "name : %s", name)

		rlog(c).Error().Msg("decline request with invalid name param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	hostname := c.Locals(auth.LKeyHostname).(string)
	aservice := c.UserContext().Value(utils.CKeyAuthService).(*auth.AuthService)
	if ok, e := aservice.AuthorizeHostname(name, hostname); e != nil {
		rlog(c).Error().Msg(e.Error())
		return fiber.NewError(fiber.StatusInternalServerError)
	} else if !ok {
		rdebugf(c, "hostname : %s", hostname)

		rlog(c).Error().Msg("decline request from unauthorized hostname")
		return fiber.NewError(fiber.StatusForbidden)
	}

	return c.Next()

}

func handleGetCertificate(c *fiber.Ctx) error {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "hostname : %s", name)

		rlog(c).Error().Msg("decline request with invalid domain param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	aservice := c.UserContext().Value(utils.CKeyAuthService).(*auth.AuthService)

	cert, e := aservice.CertificateByName(name)
	if e != nil {
		rlog(c).Error().Msg("an error occurred while peeking certificate by domain " + e.Error())
		return fiber.NewError(fiber.StatusInternalServerError)
	} else if utils.IsEmpty(cert) {
		rlog(c).Error().Msg("an empty result received from auth service")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	sservice := c.UserContext().Value(utils.CKeySystem).(*system.System)
	buf := sservice.AcquireBuffer()
	defer sservice.ReleaseBuffer(buf)

	if e = sservice.PeekFile(system.PEM_CERTIFICATE, futils.UnsafeString(cert), buf); e != nil {
		rlog(c).Error().Msg("an error occurred while peeking certificate from system")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	///

	return c.Next()
}
