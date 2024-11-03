package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MindHunter86/asmas/internal/auth"
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	futils "github.com/gofiber/fiber/v2/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
)

type HttpServiceClient struct {
	*fasthttp.HostClient

	auth *auth.AuthService

	hostname string
	asmasuri *fasthttp.URI

	log *zerolog.Logger
}

func newHttpServiceClient(c context.Context, cc *cli.Context) *HttpServiceClient {
	hostname, e := os.Hostname()
	if e != nil {
		return nil
	}

	rri := fasthttp.AcquireURI()
	if e = rri.Parse(nil, futils.UnsafeBytes(cc.String("asmas-api-url"))); e != nil {
		return nil
	}

	return &HttpServiceClient{
		HostClient: &fasthttp.HostClient{
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent#crawler_and_bot_ua_strings
			Name: fmt.Sprintf("Mozilla/5.0 (compatible; %s/%s; +mailto:%s)",
				cc.App.Name, cc.App.Version, cc.App.Authors[0].Email),

			// Addr:  cc.String("asmas-server-addr"),
			IsTLS: bytes.Equal(rri.Scheme(), []byte("https")),

			TLSConfig: &tls.Config{
				InsecureSkipVerify: cc.Bool("asmas-ssl-insecure"), // skipcq: GSC-G402 false-positive
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS13,
			},

			MaxConns: cc.Int("asmas-max-conns"),

			ReadTimeout:         cc.Duration("asmas-timeout-read"),
			WriteTimeout:        cc.Duration("asmas-timeout-write"),
			MaxIdleConnDuration: cc.Duration("asmas-timeout-idle"),
			MaxConnDuration:     cc.Duration("asmas-timeout-conn"),

			DisableHeaderNamesNormalizing: false,
			DisablePathNormalizing:        false,
			NoDefaultUserAgentHeader:      false,

			Dial: (&fasthttp.TCPDialer{
				Concurrency:      cc.Int("asmas-tcpdial-concurr"),
				DNSCacheDuration: cc.Duration("asmas-dnscache-dur"),
			}).Dial,

			// !
			// ? DialTimeout
		},

		auth: auth.NewAuthClient(c, cc),

		hostname: hostname,
		asmasuri: rri,
	}
}

func (m *HttpServiceClient) bootstrap() {
	return
}

func (m *HttpServiceClient) acquireSignedRequest(apimethod string, payload ...string) (req *fasthttp.Request) {
	req = fasthttp.AcquireRequest()

	req.SetURI(m.asmasuri)
	req.URI().SetPath(fmt.Sprintf(apimethod, payload))

	var sign []byte
	if sign = m.auth.PrepareHMACMessage(futils.UnsafeString(req.URI().Path()), m.hostname); sign == nil {
		return nil
	}

	req.URI().QueryArgs().Add("hostname", m.hostname)
	req.URI().QueryArgs().AddBytesV("sign", sign)

	req.Header.SetHost(strings.Split(m.Addr, ":")[0])
	req.UseHostHeader = true

	req.Header.Set(fasthttp.HeaderAccept, fiber.MIMEApplicationJSONCharsetUTF8)
	req.Header.Set(fasthttp.HeaderUserAgent, m.Name)
	req.Header.Set(fasthttp.HeaderKeepAlive, "timeout=5, max=1000")
	req.Header.Set(fasthttp.HeaderConnection, "keep-alive")
	req.Header.Set(fasthttp.HeaderCacheControl, "no-cache")
	req.Header.Set(fasthttp.HeaderPragma, "no-cache")

	return
}

const (
	ApiMethodCertificates       = "/v1/certificates"
	ApiMethodCertificatePublic  = "/v1/certificates/%s/public"
	ApiMethodCertificatePrivate = "/v1/certificates/%s/private"
)

func (*HttpServiceClient) releaseApiResponse(rsp *fasthttp.Response) {
	fasthttp.ReleaseResponse(rsp)
}

func (m *HttpServiceClient) doApiRequest(apimethod string, payload ...string) (_ []byte, e error) {
	req, rsp :=
		m.acquireSignedRequest(apimethod, payload...),
		fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)

	// !!!!!
	// !!!!! releaseApiResponse
	// defer m.releaseRequestResponse(req, rsp)

	// ?
	// ? maybe some limits??
	// if !m.ratereset.IsZero() && m.noRequestsInLimitWindow() {
	// 	return nil, errors.New("could not call github request because of limits, retry after " + m.ratereset.String())
	// }
	// defer m.updateGithubRateLimits(&rsp.Header)

	if e = m.Do(req, rsp); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		m.log.Trace().Msg(req.String())
		m.log.Trace().Msg(rsp.String())
	}

	status, body := rsp.StatusCode(), rsp.Body()
	if status < fasthttp.StatusOK || status >= fasthttp.StatusInternalServerError {
		return nil, errors.New("asmas api respond with an unexpected status, is asmas down?")
	} else if status >= fasthttp.StatusBadRequest && status < fasthttp.StatusInternalServerError {
		return nil, errors.New("asmas api respond with a 4XX error, seems request builder had been failed")
	}

	if utils.IsEmpty(body) {
		return nil, errors.New("asmas api respond with an empty body, unexpected result")
	}

	return body, e
}
