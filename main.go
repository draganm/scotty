package main

import (
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/go-logr/zapr"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger, _ := zap.Config{
		Encoding:    "json",
		Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths: []string{"stdout"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			// CallerKey:    "caller",
			// EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}.Build()

	defer logger.Sync()

	app := &cli.App{
		Name: "scotty",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				EnvVars: []string{"ADDR"},
				Value:   ":2222",
				Usage:   "address where the Scotty will listen for SSH connections",
			},
		},
		Action: func(c *cli.Context) error {
			log := zapr.NewLogger(logger)
			log.Info("started")
			ssh.ListenAndServe(
				c.String(`addr`),
				func(s ssh.Session) {
					pt, evs, hasPty := s.Pty()
					log.Info(
						"new connection",
						"remote", s.RemoteAddr().String(),
						"user", s.User(),
						"pty", hasPty,
						"term", pt.Term,
						"termWindow", pt.Window,
					)

					ctx := s.Context()

					if hasPty {
						go func() {
							for ctx.Err() == nil {
								select {
								case <-ctx.Done():
									log.Info("context done")
									return
								case ev := <-evs:
									log.Info("event", "window", ev)

								}
							}
						}()
					}

					time.Sleep(20 * time.Second)
					s.Close()
				},
			)
			return nil
		},
	}
	app.RunAndExitOnError()
}
