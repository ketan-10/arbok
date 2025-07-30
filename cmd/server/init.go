package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	flag "github.com/spf13/pflag"
)

// initLogger initializes slog logger instance.
func initLogger(ko *koanf.Koanf) *slog.Logger {
	// Parse log level from config
	var level slog.Level
	switch strings.ToLower(ko.String("app.log_level")) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // Enable caller information
	}

	// Create text handler for console output
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
}

// initConfig loads config to `ko` object.
func initConfig(cfgDefault string, envPrefix string) *koanf.Koanf {
	var (
		ko = koanf.New(".")
		f  = flag.NewFlagSet("front", flag.ContinueOnError)
	)

	// Configure Flags.
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register `--config` flag.
	cfgPath := f.String("config", cfgDefault, "Path to a config file to load.")

	// Parse and Load Flags.
	err := f.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Load the config files from the path provided.
	fmt.Printf("attempting to load config from file: %s\n", *cfgPath)

	err = ko.Load(file.Provider(*cfgPath), toml.Parser())
	if err != nil {
		// If the default config is not present, print a warning and continue reading the values from env.
		if *cfgPath == cfgDefault {
			fmt.Printf("unable to open sample config file: %v\n", err)
		} else {
			fmt.Printf("error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("attempting to read config from env vars\n")
	// Load environment variables if the key is given
	// and merge into the loaded config.
	if envPrefix != "" {
		err = ko.Load(env.Provider(envPrefix, ".", func(s string) string {
			return strings.Replace(strings.ToLower(
				strings.TrimPrefix(s, envPrefix)), "__", ".", -1)
		}), nil)
		if err != nil {
			fmt.Printf("error loading env config: %v\n", err)
			os.Exit(1)
		}
	}

	return ko
}
