package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

func flagsInitialization(expertMode bool) []cli.Flag {
	return []cli.Flag{
		// common settings
		&cli.StringFlag{
			Name:    "log-level",
			Value:   "info",
			Usage:   "levels: trace, debug, info, warn, err, panic, disabled",
			Aliases: []string{"l"},
			EnvVars: []string{"LOG_LEVEL"},
		},
		&cli.BoolFlag{
			Name:               "expert-mode",
			Usage:              "show hidden flags",
			DisableDefaultText: true,
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
			Value:    "tcp",
			EnvVars:  []string{"SYSLOG_PROTO"},
		},
		&cli.StringFlag{
			Name:     "syslog-tag",
			Category: "Syslog settings",
			Usage:    "optional setting; more information in syslog RFC",
			Value:    "",
			Hidden:   expertMode,
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
		},
		&cli.BoolFlag{
			Name:     "http-prefork",
			Category: "HTTP server settings",
			Usage: `enables use of the SO_REUSEPORT socket option;
			if enabled, the application will need to be ran
			through a shell because prefork mode sets environment variables;
			EXPERIMENTAL! USE CAREFULLY!`,
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.DurationFlag{
			Name:     "http-read-timeout",
			Category: "HTTP server settings",
			Value:    10 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-write-timeout",
			Category: "HTTP server settings",
			Value:    5 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "http-idle-timeout",
			Category: "HTTP server settings",
			Value:    10 * time.Minute,
		},
		&cli.BoolFlag{
			Name:               "http-pprof-enable",
			Category:           "HTTP server settings",
			Usage:              "enable golang http-pprof methods",
			DisableDefaultText: true,
		},
		&cli.StringFlag{
			Name:     "http-pprof-prefix",
			Category: "HTTP server settings",
			Usage:    "it should start with (but not end with) a slash. Example: '/test'",
			EnvVars:  []string{"PPROF_PREFIX"},
		},
		&cli.StringFlag{
			Name:     "http-pprof-secret",
			Category: "HTTP server settings",
			Usage:    "define static secret in x-pprof-secret header for avoiding unauthorized access",
			EnvVars:  []string{"PPROF_SECRET"},
		},

		// limiter settings
		&cli.BoolFlag{
			Name:               "limiter-enable",
			Category:           "Limiter settings",
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.BoolFlag{
			Name:               "limiter-use-bbolt",
			Category:           "Limiter settings",
			Usage:              "use bbolt key\value file database instead of memory database",
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.BoolFlag{
			Name:               "limiter-bbolt-reset",
			Category:           "Limiter settings",
			Usage:              "if bbolt used as storage, reset all limited IPs on startup",
			Hidden:             expertMode,
			DisableDefaultText: true,
		},
		&cli.IntFlag{
			Name:     "limiter-max-req",
			Category: "Limiter settings",
			Hidden:   expertMode,
			Value:    200,
		},
		&cli.DurationFlag{
			Name:     "limiter-records-duration",
			Category: "Limiter settings",
			Hidden:   expertMode,
			Value:    5 * time.Minute,
		},
	}
}
