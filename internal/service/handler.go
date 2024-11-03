package service

import (
	"errors"
	"strings"

	"github.com/MindHunter86/asmas/internal/auth"
	"github.com/MindHunter86/asmas/internal/system"
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
)

func (*Service) fiberDefaultErrorHandler(c *fiber.Ctx, err error) error {
	// reject invalid requests
	if strings.TrimSpace(c.Hostname()) == "" {
		gLog.Warn().Msg("invalid request from " + c.Context().RemoteIP().String())
		gLog.Debug().Msgf("invalid request: %+v ; error - %+v", c, err)
		return c.Context().Conn().Close()
	}

	// ? not profitable
	ferr := AcquireFErr()
	defer ReleaseFErr(ferr)

	// parse fiber error
	if !errors.As(err, &ferr) {
		rdebugf(c, "undefined error caught %+v", ferr)
		ferr.Code, ferr.Message = fiber.StatusInternalServerError, err.Error()
		return ferr
	}

	rdebugf(c, "fiber error caught %+v", ferr)
	return ferr
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
	if payload = aservice.PrepareHMACMessage(c.Path(), hostname); payload == nil {
		rdebugf(c, "chunks: %s | %s", c.Path(), hostname)

		rlog(c).Error().Msg("unexpected result while preparing message for sign verification")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	if expect, ok := aservice.SignWithVerifyHMACMessage(payload, futils.UnsafeBytes(sign)); !ok {
		rdebugf(c, "chunks : %s | %s", c.Path(), hostname)
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

func handleGetCertificates(c *fiber.Ctx) (e error) {
	hostname := c.Locals(auth.LKeyHostname).(string)
	aservice := c.UserContext().Value(utils.CKeyAuthService).(*auth.AuthService)

	var authorizations []string
	if authorizations, e = aservice.GetAvailableDomains(hostname); e != nil {
		return fiber.NewError(fiber.StatusInternalServerError, e.Error())
	} else if len(authorizations) == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}

	c.WriteString(strings.Join(authorizations, "|"))
	return respondPlainWithStatus(c, fiber.StatusOK)
}

func handleGetCertificate(c *fiber.Ctx) (e error) {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "hostname : %s", name)

		rlog(c).Error().Msg("decline request with invalid domain param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	sservice := c.UserContext().Value(utils.CKeySystem).(*system.System)
	if _, e = sservice.WritePemTo(name, system.PEM_CERTIFICATE, c); e != nil {
		rlog(c).Error().Msg("an error occurred while peeking certificate from system, " + e.Error())
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return respondPlainWithStatus(c, fiber.StatusOK)
}

func handleGetPrivate(c *fiber.Ctx) (e error) {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "hostname : %s", name)

		rlog(c).Error().Msg("decline request with invalid domain param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	sservice := c.UserContext().Value(utils.CKeySystem).(*system.System)
	if _, e = sservice.WritePemTo(name, system.PEM_PRIVATEKEY, c); e != nil {
		rlog(c).Error().Msg("an error occurred while peeking certificate from system, " + e.Error())
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	return respondPlainWithStatus(c, fiber.StatusOK)
}
