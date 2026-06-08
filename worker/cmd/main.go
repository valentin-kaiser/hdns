package main

import (
	"fmt"
	"os"
	"path/filepath"

	kardianos "github.com/kardianos/service"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/interruption"
	"github.com/valentin-kaiser/go-core/logging"
	"github.com/valentin-kaiser/go-core/logging/log"
	goservice "github.com/valentin-kaiser/go-core/service"
	"github.com/valentin-kaiser/go-core/web"
	"github.com/valentin-kaiser/hdns-worker/pkg/config"
	"github.com/valentin-kaiser/hdns-worker/pkg/handler"
)

const (
	svcName        = "HDNSWorker"
	svcDisplayName = "HDNS Webhook Worker"
	svcDescription = "Receives HTTP webhook calls from HDNS and executes configured tasks (certificate deployment, service restarts, custom commands)."
)

func init() {
	defer interruption.Catch()
	interruption.Write = true

	logging.Anonymous(true)
	apperror.Anonymous(true)
	apperror.ErrorHandler = func(err error, msg string) {
		log.Error().Err(err).Msg(msg)
	}

	logging.SetGlobalAdapter(logging.
		NewZerologAdapter().
		WithConsole().
		WithFileRotation(filepath.Join(flag.Path, "logs", "worker.log"), 10, 30, 30, true).
		SetLevel(logging.InfoLevel))

	// config.Init() calls flag.Init() internally, reads hdns-worker.yaml, and
	// validates the config. Fatal errors are logged and exit the process.
	config.Init()

	logging.GetGlobalAdapterInterface().SetLevel(logging.Level(config.Get().LogLevel))
}

func main() {
	defer interruption.Catch()

	if flag.Help {
		flag.PrintHelp()
		fmt.Fprintln(os.Stderr, "\nService management commands:")
		fmt.Fprintln(os.Stderr, "  install    Install as a system service")
		fmt.Fprintln(os.Stderr, "  uninstall  Remove the system service")
		fmt.Fprintln(os.Stderr, "  start      Start the installed service")
		fmt.Fprintln(os.Stderr, "  stop       Stop the running service")
		fmt.Fprintln(os.Stderr, "  restart    Restart the running service")
		return
	}

	svcConfig := &goservice.Config{
		Name:        svcName,
		DisplayName: svcDisplayName,
		Description: svcDescription,
	}

	// Non-flag positional arguments are service management commands.
	if args := flag.Arguments(); len(args) > 0 {
		action := args[0]
		switch action {
		case "install", "uninstall", "start", "stop", "restart":
			noop := &noopProgram{}
			svc, err := kardianos.New(noop, svcConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create service handle: %v\n", err)
				os.Exit(1)
			}
			if err := kardianos.Control(svc, action); err != nil {
				fmt.Fprintf(os.Stderr, "service action %q failed: %v\n", action, err)
				os.Exit(1)
			}
			fmt.Printf("service %q: %s completed successfully\n", svcName, action)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n", action)
			fmt.Fprintln(os.Stderr, "valid commands: install, uninstall, start, stop, restart")
			os.Exit(1)
		}
	}

	start := func(s *goservice.Service) error {
		cfg := config.Get()
		log.Info().Msgf("[WORKER] starting on port %d", cfg.Port)
		log.Debug().Msgf("[WORKER] data path: %s", flag.Path)
		log.Debug().Msgf("[WORKER] tasks registered: %d", len(cfg.Tasks))

		done := make(chan error, 1)

		srv := web.New().
			WithHTTPPort(cfg.Port).
			WithLog()

		for _, t := range cfg.Tasks {
			path := "/" + t.Name + "/"
			srv.WithHandlerFunc(path, handler.Task(&t))
			log.Debug().Msgf("[WORKER] registered route POST %s", path)
		}

		srv.StartAsync(done)

		if err := web.Instance().Error; err != nil {
			return err
		}

		log.Info().Msgf("[WORKER] listening on :%d", cfg.Port)
		return <-done
	}

	stop := func(s *goservice.Service) error {
		log.Info().Msg("[WORKER] service stop requested")
		return web.Instance().Shutdown()
	}

	if err := goservice.Run(svcConfig, start, stop); err != nil {
		log.Error().Err(err).Msg("[WORKER] fatal error")
		os.Exit(1)
	}
}

// noopProgram implements kardianos/service.Interface with no-op operations.
// It is only used for service management commands (install, uninstall, etc.)
// that manage the service registration without running the program logic.
type noopProgram struct{}

func (n *noopProgram) Start(_ kardianos.Service) error { return nil }
func (n *noopProgram) Stop(_ kardianos.Service) error  { return nil }
