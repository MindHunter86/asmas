package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

func flagsInitialization(expertmode bool) []cli.Flag {
	return []cli.Flag{
		// common settings
		&cli.StringFlag{
			Name:     "log-level",
			Category: "Common settings",
			Value:    "info",
			Usage:    "levels: trace, debug, info, warn, err, panic, disabled",
			Aliases:  []string{"l"},
			EnvVars:  []string{"LOG_LEVEL"},
		},
		&cli.BoolFlag{
			Name:               "expert-mode",
			Category:           "Common settings",
			Usage:              "show hidden flags",
			DisableDefaultText: true,
		},

		// debug feature-flags
		&cli.BoolFlag{
			Name:     "debug-skip-github-connect",
			Category: "Debug options",
			Hidden:   expertmode,
		},

		// common settings : syslog
		&cli.StringFlag{
			Name:     "syslog-server",
			Category: "Syslog settings",
			Usage:    "syslog server (optional); syslog sender is not used if value is empty",
			Value:    "",
			EnvVars:  []string{"SYSLOG_ADDRESS"},
		},
		&cli.StringFlag{
			Name:     "syslog-proto",
			Category: "Syslog settings",
			Usage:    "syslog protocol (optional); tcp or udp is possible",
			Value:    "udp",
			EnvVars:  []string{"SYSLOG_PROTO"},
		},
		&cli.StringFlag{
			Name:     "syslog-tag",
			Category: "Syslog settings",
			Usage:    "optional setting; more information in syslog RFC",
			Value:    "",
			Hidden:   expertmode,
		},

		// fiber-server settings
		&cli.StringFlag{
			Name:     "http-listen-addr",
			Category: "HTTP server settings",
			Usage:    "format - 127.0.0.1:8080, :8080",
			Value:    "127.0.0.1:8080",
		},
		&cli.StringFlag{
			Name:     "http-trusted-proxies",
			Category: "HTTP server settings",
			Usage:    "format - 192.168.0.0/16; can be separated by comma",
		},
		&cli.StringFlag{
			Name:     "http-realip-header",
			Category: "HTTP server settings",
			Value:    "X-Real-Ip",
			Hidden:   expertmode,
		},
		&cli.DurationFlag{
			Name:     "http-timeout-read",
			Category: "HTTP server settings",
			Value:    10 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-timeout-write",
			Category: "HTTP server settings",
			Value:    5 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-timeout-idle",
			Category: "HTTP server settings",
			Value:    10 * time.Minute,
		},
		&cli.BoolFlag{
			Name:               "http-pprof-enable",
			Category:           "HTTP server settings",
			Usage:              "enable golang http-pprof methods",
			DisableDefaultText: true,
			Hidden:             expertmode,
		},
		&cli.StringFlag{
			Name:     "http-pprof-prefix",
			Category: "HTTP server settings",
			Usage:    "it should start with (but not end with) a slash. Example: '/test'",
			Value:    "/internal",
			EnvVars:  []string{"PPROF_PREFIX"},
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "http-pprof-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in x-pprof-secret header for avoiding unauthorized access",
			Value:    "changemeplease",
			EnvVars:  []string{"PPROF_SECRET"},
			Hidden:   expertmode,
		},

		// auth service settings
		&cli.StringFlag{
			Name:     "auth-sign-token",
			Category: "Auth service settings",
			Value:    "changemeplease12345",
		},
		&cli.StringFlag{
			Name:     "auth-github-repo",
			Category: "Auth service settings",
			Value:    "MindHunter86/asmas-test",
		},
		&cli.StringFlag{
			Name:     "auth-github-path",
			Category: "Auth service settings",
			Value:    "config.yaml.asc",
		},
		&cli.StringFlag{
			Name:     "auth-github-branch",
			Category: "Auth service settings",
			Value:    "master",
		},
		&cli.DurationFlag{
			Name:     "auth-github-pull-interval",
			Category: "Auth service settings",
			Value:    5 * time.Minute,
		},
		&cli.DurationFlag{
			Name:     "auth-github-pull-error-delay",
			Category: "Auth service settings",
			Value:    1 * time.Minute,
		},

		// github http client settings
		&cli.StringFlag{
			Name:     "github-api-addr",
			Category: "Github client settings",
			Value:    "api.github.com:443",
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "github-api-version",
			Category: "Github client settings",
			Usage:    "https://docs.github.com/en/rest/about-the-rest-api/api-versions?apiVersion=2022-11-28#about-api-versioning",
			Value:    "2022-11-28",
			Hidden:   expertmode,
		},
		&cli.BoolFlag{
			Name:     "github-ssl-insecure",
			Category: "Github client settings",
			Hidden:   expertmode,
		},
		&cli.IntFlag{
			Name:     "github-max-conns",
			Category: "Github client settings",
			Value:    32,
			Hidden:   expertmode,
		},
		&cli.DurationFlag{
			Name:     "github-timeout-read",
			Category: "Github client settings",
			Value:    3 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "github-timeout-write",
			Category: "Github client settings",
			Value:    3 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "github-timeout-idle",
			Category: "Github client settings",
			Usage:    "idle keep-alive connections are closed after this duration",
			Value:    5 * time.Minute,
		},
		&cli.DurationFlag{
			Name:     "github-timeout-conn",
			Category: "Github client settings",
			Usage:    "keep-alive connections are closed after this duration",
			Value:    1 * time.Second,
		},
		&cli.IntFlag{
			Name:     "github-tcpdial-concurr",
			Category: "Github client settings",
			Usage:    "concurrency controls the maximum number of concurrent Dials that can be performed using this object. Setting this to 0 means unlimited",
			Value:    0,
			Hidden:   expertmode,
		},
		&cli.DurationFlag{
			Name:     "github-dnscache-dur",
			Category: "Github client settings",
			Usage:    "this may be used to override the default DNS cache duration",
			Value:    1 * time.Minute,
			Hidden:   expertmode,
		},

		// Certbot settings
		&cli.BoolFlag{
			Name:     "certbot-args-reuse-key",
			Category: "Certbot settings",
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "certbot-args-key-type",
			Category: "Certbot settings",
			Value:    "ecdsa",
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "certbot-args-elliptic-curve",
			Category: "Certbot settings",
			Value:    "secp256r1",
			Hidden:   expertmode,
		},
		&cli.IntFlag{
			Name:     "certbot-args-http-01-port",
			Category: "Certbot settings",
			Value:    8079,
		},
		&cli.StringFlag{
			Name:     "certbot-args-certs-path",
			Category: "Certbot settings",
			Value:    "/etc/letsencrypt/live/",
		},
		&cli.StringFlag{
			Name:     "certbot-args-account-email",
			Category: "Certbot settings",
			Value:    "root@example.com",
		},

		// system settings
		&cli.StringFlag{
			Name:     "system-cert-path",
			Category: "System settings",
			Usage:    "should be like certbot-args-certs-path",
			Value:    "/etc/letsencrypt/live/",
		},
		&cli.Int64Flag{
			Name:     "system-pem-size-limit",
			Category: "System settings",
			Usage:    "certificate in pem format file size limit in kilobytes",
			Value:    10,
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "system-pem-pubname",
			Category: "System settings",
			Usage:    "",
			Value:    "fullchain.pem",
			Hidden:   expertmode,
		},
		&cli.StringFlag{
			Name:     "system-pem-keyname",
			Category: "System settings",
			Usage:    "",
			Value:    "privkey.pem",
			Hidden:   expertmode,
		},
	}
}
