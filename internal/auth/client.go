package auth

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	futils "github.com/gofiber/fiber/v2/utils"
	"github.com/mailru/easyjson"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
)

type (
	HttpClient struct {
		*fasthttp.HostClient

		githuburi    *fasthttp.URI
		githubapiver string

		rateremain int
		ratereset  time.Time

		log *zerolog.Logger
	}

	//easyjson:json
	GithubResponse struct {
		// OK response
		Name    string `json:",omitempty"`
		Sha     string `json:",omitempty"`
		Size    int    `json:",omitempty"`
		Type    string `json:",omitempty"`
		Content []byte `json:",omitempty"`

		// Error response
		Message string `json:",omitempty"`
		Status  int    `json:",omitempty"`
	}
)

func NewHttpClient(cc *cli.Context, log *zerolog.Logger) *HttpClient {
	rri, apihost := fasthttp.AcquireURI(), []byte(cc.String("github-api-addr"))
	apiurl := fmt.Sprintf("https://%s/repos/%s/contents/%s?ref=%s",
		cc.String("github-api-addr"),
		cc.String("auth-github-repo"),
		cc.String("auth-github-path"),
		cc.String("auth-github-branch"))

	if e := rri.Parse([]byte(apihost), []byte(apiurl)); e != nil {
		return nil
	}

	return &HttpClient{
		HostClient: &fasthttp.HostClient{
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent#crawler_and_bot_ua_strings
			Name: fmt.Sprintf("Mozilla/5.0 (compatible; %s/%s; +https://anilibria.top/support)",
				cc.App.Name, cc.App.Version),

			Addr:  cc.String("github-api-addr"),
			IsTLS: bytes.Equal(rri.Scheme(), []byte("https")),

			TLSConfig: &tls.Config{
				InsecureSkipVerify: cc.Bool("github-ssl-insecure"), // skipcq: GSC-G402 false-positive
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS13,
			},

			MaxConns: cc.Int("github-max-conns"),

			ReadTimeout:         cc.Duration("github-timeout-read"),
			WriteTimeout:        cc.Duration("github-timeout-write"),
			MaxIdleConnDuration: cc.Duration("github-timeout-idle"),
			MaxConnDuration:     cc.Duration("github-timeout-conn"),

			DisableHeaderNamesNormalizing: false,
			DisablePathNormalizing:        false,
			NoDefaultUserAgentHeader:      false,

			Dial: (&fasthttp.TCPDialer{
				Concurrency:      cc.Int("github-tcpdial-concurr"),
				DNSCacheDuration: cc.Duration("github-dnscache-dur"),
			}).Dial,

			// !!!
			// ? DialTimeout
		},

		githuburi:    rri,
		githubapiver: cc.String("github-api-version"),

		log: log,
	}
}

//
//
//

func (m *HttpClient) acquireRequestResponse() (req *fasthttp.Request, rsp *fasthttp.Response) {
	req, rsp = fasthttp.AcquireRequest(), fasthttp.AcquireResponse()

	req.SetURI(m.githuburi)

	req.Header.Set(fasthttp.HeaderAccept, "application/vnd.github+json; charset=utf-8")
	req.Header.Set(fasthttp.HeaderUserAgent, m.Name)
	req.Header.Set(fasthttp.HeaderKeepAlive, "timeout=5, max=1000")
	req.Header.Set(fasthttp.HeaderConnection, "keep-alive")
	req.Header.Set(fasthttp.HeaderCacheControl, "no-cache")
	req.Header.Set(fasthttp.HeaderPragma, "no-cache")

	req.Header.Set("X-GitHub-Api-Version", m.githubapiver)

	req.Header.SetHost(strings.Split(m.Addr, ":")[0])
	req.UseHostHeader = true

	return
}

func (*HttpClient) releaseRequestResponse(req *fasthttp.Request, rsp *fasthttp.Response) {
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(rsp)
}

func (m *HttpClient) fetchConfigFromGithub() (_ *GithubResponse, e error) {
	req, rsp := m.acquireRequestResponse()
	defer m.releaseRequestResponse(req, rsp)

	if !m.ratereset.IsZero() && !m.hasRequestsInLimitWindow() {
		return nil, errors.New("could not call github request because of limits, retry after " + m.ratereset.String())
	}
	defer m.updateGithubRateLimits(&rsp.Header)

	if e = m.Do(req, rsp); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		m.log.Trace().Msg(req.String())
		m.log.Trace().Msg(rsp.String())
	}

	status, body := rsp.StatusCode(), rsp.Body()
	if status < fasthttp.StatusOK || status >= fasthttp.StatusInternalServerError {
		return nil, errors.New("github api respond with an unexpected status, is github down?")
	} else if status >= fasthttp.StatusBadRequest && status < fasthttp.StatusInternalServerError {
		return nil, errors.New("github api respond with a 4XX error, seems request builder had been failed")
	}

	if IsEmpty(body) {
		return nil, errors.New("github api respond with an empty body, unexpected result")
	}

	response := &GithubResponse{}
	return response, easyjson.Unmarshal(rsp.Body(), response)
}

func (m *HttpClient) hasRequestsInLimitWindow() bool {
	return m.rateremain > 0 && !time.Now().Before(m.ratereset)
}

func (m *HttpClient) updateGithubRateLimits(headers *fasthttp.ResponseHeader) (e error) {
	var remaining, reset []byte
	if remaining = headers.Peek("X-RateLimit-Remaining"); IsEmpty(remaining) {
		return errors.New("ratelimit headers (remaining) are empty")
	}

	if reset = headers.Peek("X-RateLimit-Reset"); IsEmpty(reset) {
		return errors.New("ratelimit headers (reset) are empty")
	}

	var remainingbuf int
	if remainingbuf, e = strconv.Atoi(futils.UnsafeString(remaining)); e != nil {
		return
	}

	var resetbuf int64
	if resetbuf, e = strconv.ParseInt(futils.UnsafeString(reset), 10, 64); e != nil {
		return
	}

	m.rateremain = remainingbuf
	m.ratereset = time.Unix(resetbuf, 0)
	return
}

func (m *HttpClient) validateGithubResponse(response *GithubResponse) error {
	if response == nil {
		return errors.New("BUG! given gihub response is nil")
	}

	if response.Message != "" || response.Status != 0 {
		m.log.Trace().Msgf("status: %d; message: %s", response.Status, response.Message)
		return errors.New("response has message/status field, seems github respond with an error")
	}

	if clen := len(response.Content); clen != response.Size {
		m.log.Trace().Msgf("content len(): %d; reponse.size: %d", clen, response.Size)
		return errors.New("response content length is not matches with responded size")
	}

	if response.Type != "file" {
		m.log.Trace().Msgf("response type: %s; expecting 'file'", response.Type)
		return errors.New("unexpected response object type received")
	}

	m.log.Info().Msgf("downloaded and validated file %s with hash %s and length %d",
		response.Name, response.Sha, response.Size)
	return nil
}
