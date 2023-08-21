package main

import (
	"fmt"

	"github.com/draganm/scotty/k8sexec"
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

			ke, err := k8sexec.NewK8SExecutor()
			if err != nil {
				return fmt.Errorf("could not create new k8s executor: %w", err)
			}

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

					sizeEvents := make(chan k8sexec.WindowSize, 1)
					sizeEvents <- k8sexec.WindowSize{
						Width:  pt.Window.Height,
						Height: pt.Window.Width,
					}

					if hasPty {
						go func() {
							for ctx.Err() == nil {
								select {
								case <-ctx.Done():
									log.Info("context done")
									return
								case ev := <-evs:
									// log.Info("event", "window", ev)
									// s.Write([]byte("event!\n"))
									sizeEvents <- k8sexec.WindowSize{
										Width:  ev.Width,
										Height: ev.Height,
									}
								}
							}

						}()
					}

					err = ke.RunOnPod(
						ctx,
						"hook-hub",
						"hook-hub-0",
						"hook-hub",
						[]string{"/bin/sh"},
						s,
						s,
						s.Stderr(),
						hasPty,
						sizeEvents,
					)

					if err != nil {
						s.Exit(1)
					} else {
						s.Close()
					}
				},
			)
			return nil
		},
	}
	app.RunAndExitOnError()
}
