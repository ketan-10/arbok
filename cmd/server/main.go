package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mr-karan/arbok/internal/tunnel"
)

var (
	// Version of the build. This is injected at build-time.
	buildString = "unknown"
)

func main() {
	// load config.
	ko := initConfig("config.sample.toml", "ARBOK_SERVER")

	// Create a new context which is cancelled when `SIGINT`/`SIGTERM` is received.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	app := &App{
		lo: initLogger(ko),
	}
	app.lo.Info("booting arbok server", "version", buildString)

	tun, err := tunnel.New(tunnel.PeerOpts{
		Logger:     app.lo,
		Verbose:    ko.Bool("app.verbose"),
		CIDR:       ko.String("server.cidr"),
		ListenPort: ko.Int("server.listen_port"),
		PrivateKey: ko.MustString("server.private_key"),
	})
	if err != nil {
		app.lo.Fatal("error initialising wg tunnel", "error", err)
	}

	app.tun = tun

	var wg sync.WaitGroup
	// Start the http server in background.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.tun.Up(ctx); err != nil {
			app.lo.Fatal("error starting wg device", "error", err)
		}
	}()

	// Listen on the close channel indefinitely until a
	// `SIGINT` or `SIGTERM` is received.
	<-ctx.Done()
	// Cancel the context to gracefully shutdown and perform
	// any cleanup tasks.
	cancel()
	wg.Wait()
}
