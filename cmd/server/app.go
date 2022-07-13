package main

import (
	"time"

	"github.com/mr-karan/arbok/internal/tunnel"
	"github.com/zerodha/logf"
)

// App is the global app context container that is passed
// around and injected everywhere.
type App struct {
	lo   logf.Logger
	tun  *tunnel.Tunnel
	opts Options
}

// Options represents config options required for app.
type Options struct {
	RefreshInterval time.Duration
}
