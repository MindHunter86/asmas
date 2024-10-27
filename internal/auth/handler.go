package auth

import (
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	RegistrationArgHostname = "hostname"
	RegistrationArgSign     = "sign"
)

// !!!! REQUEST VALIDATION
// Check Content-Type

// Base request validation
func MiddlewareAuthentification(c *fiber.Ctx) error {
	// var sign []byte
	// if sign = c.Request().PostArgs().Peek(RegistrationArgSign); IsEmpty(sign) {
	// 	rdebugf(c, "sign : %s", futils.UnsafeString(sign))

	// 	rlog(c).Error().Msg("decline request wo sign argument")
	// 	return fiber.NewError(fiber.StatusBadRequest)
	// }

	var hostname string
	if hostname = c.Query(RegistrationArgHostname); hostname == "" {
		rdebugf(c, "hostname : %s", hostname)

		rlog(c).Error().Msg("decline request wo hostname argument")
		return fiber.NewError(fiber.StatusBadRequest)
	}
	c.Locals(LKeyHostname, hostname)

	// !!!
	//! Check SIGN!!!
	// !!!

	return c.Next()
}

// Variables authorization with Github config
func MiddlewareAuthorization(c *fiber.Ctx) error {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "name : %s", name)

		rlog(c).Error().Msg("decline request with invalid name param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	hostname := c.Locals(LKeyHostname).(string)
	aservice := c.UserContext().Value(utils.CKeyAuthService).(*AuthService)
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

func HandleGetCertificate(c *fiber.Ctx) error {
	var name string
	if name = c.Params("name"); name == "" {
		rdebugf(c, "hostname : %s", name)

		rlog(c).Error().Msg("decline request with invalid domain param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	aservice := c.UserContext().Value(utils.CKeyAuthService).(*AuthService)

	cert, e := aservice.CertificateByName(name)
	if e != nil {
		rlog(c).Error().Msg("an error occurred while peeking certificate by domain " + e.Error())
		return fiber.NewError(fiber.StatusInternalServerError)
	} else if IsEmpty(cert) {
		rlog(c).Error().Msg("an empty result received from auth service")
		return fiber.NewError(fiber.StatusInternalServerError)
	}

	///

	return c.Next()
}

func HandlerInvitation(c *fiber.Ctx) error {
	// get ip, ua; gen TimeNano; concate and sign with secret

	// get time, add 10 seconds and hmac it, return hash and time (time means expire_in)
	// also cache cache in map and reuse it
	return c.Next()
}

func HandlerRegistration(c *fiber.Ctx) error {
	// var auth []byte
	// if auth = c.Request().Header.Peek(fiber.HeaderAuthorization); len(auth) == 0 {
	// 	rlog(c).Error().Msg("decline request wo Authorization header")
	// 	return fiber.NewError(fiber.StatusBadRequest)
	// }

	// aservice := c.UserContext().Value(utils.CKeyAuthService).(*AuthService)
	// if !bytes.Equal(auth, aservice.RegistrationToken()) {
	// 	rlog(c).Error().Msg("decline request with incorrect registration token")
	// 	return fiber.NewError(fiber.StatusForbidden)
	// }

	// hostname, sign :=
	// 	c.Request().PostArgs().Peek(RegistrationArgHostname),
	// 	c.Request().PostArgs().Peek(RegistrationArgSign)

	// if len(hostname) == 0 || len(sign) == 0 {
	// 	rdebugf(c, "hostname:invite:sign %s:%s:%s",
	// 		futils.UnsafeString(hostname), futils.UnsafeString(sign), futils.UnsafeString(invitation))

	// 	rlog(c).Error().Msg("decline request with invalid args in body")
	// 	return fiber.NewError(fiber.StatusBadRequest)
	// }

	return c.Next()
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
