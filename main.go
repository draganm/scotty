package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/draganm/scotty/k8sexec"
	"github.com/draganm/scotty/k8sutil"
	"github.com/draganm/scotty/tui"
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

			cl, err := k8sutil.NewClient()
			if err != nil {
				return fmt.Errorf("could not create new k8s client: %w", err)
			}

			// middlewares :=

			s, err := wish.NewServer(
				wish.WithAddress(c.String(`addr`)),
				wish.WithHostKeyPath(".ssh/term_info_ed25519"),
				wish.WithMiddleware(
					teaHandler(cl, ke),
				),
			)

			if err != nil {
				return fmt.Errorf("could not create ssh server: %w", err)
			}

			return s.ListenAndServe()

			return nil
		},
	}
	app.RunAndExitOnError()
}

func teaHandler(cl *k8sutil.Client, ke *k8sexec.K8SExecutor) wish.Middleware {

	return func(h ssh.Handler) ssh.Handler {
		cs := &tui.CurrentSelection{}

		m1 := func(h ssh.Handler) ssh.Handler {

			return func(s ssh.Session) {

				// wish.Printf(s, "ns: %s, pod: %s\n", ns, pod)

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

				err := ke.RunOnPod(
					ctx,
					cs.Namespace,
					cs.Pod,
					cs.Container,
					[]string{cs.Command},
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

			}

		}

		m2 := bm.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
			kl := &k8sLister{ctx: s.Context(), kc: cl}
			return tui.SelectContainerAndCommand(cs, kl), []tea.ProgramOption{tea.WithAltScreen()}
		})

		return m2(m1(h))

	}

}
