package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/interruption"
	"github.com/valentin-kaiser/go-core/logging"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/version"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/dns"
	"github.com/valentin-kaiser/hdns/pkg/web"
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
		WithFileRotation(filepath.Join(flag.Path, "logs", "hdns.log"), 10, 30, 30, true).
		SetLevel(logging.InfoLevel))

	config.Init()
	config.OnChange(func(o *config.App, n *config.App) error {
		log.Info().Msg("applying configuration changes")
		if o.LogLevel != n.LogLevel {
			adapter, _ := logging.GetGlobalAdapter[logging.Adapter]()
			adapter.SetLevel(logging.Level(n.LogLevel))
		}
		if o.WebPort != n.WebPort {
			log.Info().Msg("restarting web server due to address/port change")
			go web.Restart()
		}
		return nil
	})
}

func main() {
	defer interruption.Catch()

	apperror.WithDetails = flag.Debug
	logging.Debug(flag.Debug)
	logging.GetGlobalAdapterInterface().SetLevel(logging.Level(config.Get().LogLevel))

	if flag.Help {
		flag.PrintHelp()
		return
	}

	if flag.Version {
		fmt.Print(version.String())
		return
	}

	log.Info().Msgf("=== HDNS %s ===", version.String())
	if flag.Debug {
		log.Debug().Msg("running in debug mode")
		log.Debug().Msgf("data path: %s", flag.Path)
		log.Debug().Msgf("git tag: %s", version.GitTag)
		log.Debug().Msgf("git commit: %s", version.GitCommit)
		log.Debug().Msgf("git short: %s", version.GitShort)
		log.Debug().Msgf("build date: %s", version.BuildDate)
		log.Debug().Msgf("runtime version: %s %s", version.GoVersion, version.Platform)

		for _, mod := range version.Modules {
			log.Debug().Msgf("module => %s %s %s", mod.Path, mod.Version, mod.Sum)
		}
	}

	database.HDNS().RegisterOnConnectHandler(database.Setup)

	go database.Connect()
	database.HDNS().AwaitConnection()

	err := dns.Refresh(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to perform initial DNS refresh")
		return
	}

	err = dns.Start(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to start DNS service")
		return
	}

	go web.Start()

	signal := interruption.OnSignal([]func() error{
		func() error {
			err := web.Stop()
			if err != nil {
				return apperror.Wrap(err)
			}
			return nil
		},
		func() error {
			dns.Stop()
			return nil
		},
		func() error {
			err := database.HDNS().Disconnect()
			if err != nil {
				return apperror.Wrap(err)
			}
			return nil
		},
		func() error {
			log.Info().Msg("stopped gracefully")
			return nil
		},
	}, os.Interrupt, syscall.SIGTERM)
	defer interruption.WaitForShutdown(signal)
}
