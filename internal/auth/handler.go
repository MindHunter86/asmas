package auth

import (
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	futils "github.com/gofiber/fiber/v2/utils"
)

const (
	RegistrationArgHostname   = "hostname"
	RegistrationArgInvitation = "hostname"
	RegistrationArgSign       = "sign"
)

func MiddlewareAuthorization(c *fiber.Ctx) error {
	var sign []byte
	if sign = c.Request().PostArgs().Peek(RegistrationArgSign); IsEmpty(sign) {
		rdebug(c, "sign : %s", futils.UnsafeString(sign))

		rlog(c).Error().Msg("decline request wo sign argument")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	var hostname []byte
	if hostname = c.Request().PostArgs().Peek(RegistrationArgHostname); IsEmpty(hostname) {
		rdebug(c, "hostname : %s", futils.UnsafeString(hostname))

		rlog(c).Error().Msg("decline request wo hostname argument")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	aservice := c.UserContext().Value(utils.CKeyAuthService).(*AuthService)
	if !aservice.AuthorizeHostname(hostname) {
		rdebug(c, "hostname : %s", futils.UnsafeString(hostname))

		rlog(c).Error().Msg("decline request from unauthorized hostname")
		return fiber.NewError(fiber.StatusForbidden)
	}

	return c.Next()
}

func HandleGetCertificate(c *fiber.Ctx) error {
	var domain string
	if domain = c.Params("domain"); domain == "" {
		rdebug(c, "hostname : %s", domain)

		rlog(c).Error().Msg("decline request with invalid domain param")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	aservice := c.UserContext().Value(utils.CKeyAuthService).(*AuthService)

	cert, e := aservice.CertificateByDomain(domain)
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

	hostname, invitation, sign :=
		c.Request().PostArgs().Peek(RegistrationArgHostname),
		c.Request().PostArgs().Peek(RegistrationArgInvitation),
		c.Request().PostArgs().Peek(RegistrationArgSign)

	if len(hostname) == 0 || len(invitation) == 0 || len(sign) == 0 {
		rdebug(c, "hostname:invite:sign %s:%s:%s",
			futils.UnsafeString(hostname), futils.UnsafeString(sign), futils.UnsafeString(invitation))

		rlog(c).Error().Msg("decline request with invalid args in body")
		return fiber.NewError(fiber.StatusBadRequest)
	}

	return c.Next()
}

func rlog(c *fiber.Ctx) *zerolog.Logger {
	return nil
}

func rdebug(c *fiber.Ctx, format string, opts ...interface{}) {
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		return
	}

	rlog(c).Debug().Msgf(format, opts...)
}
