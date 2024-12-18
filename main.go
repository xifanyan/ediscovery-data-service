package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/xifanyan/ediscovery-data-service/auth"
	"github.com/xifanyan/ediscovery-data-service/config"
	"github.com/xifanyan/ediscovery-data-service/handler"
	"github.com/xifanyan/ediscovery-data-service/service"
)

var (
	configFile = flag.String("config", "config.json", "config file")
)

func setupLogWriter(cfg config.Config) (io.Writer, error) {

	if err := os.MkdirAll(filepath.Dir(cfg.Log.Path), os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	logFile, err := os.OpenFile(cfg.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}

	if cfg.Log.Console {
		return io.MultiWriter(logFile, os.Stdout), nil
	}
	return logFile, nil
}

func setupGlobalLogger(cfg config.Config) {
	// Configure global settings
	zerolog.TimeFieldFormat = time.RFC3339

	w, err := setupLogWriter(cfg)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to setup log writer")
	}

	switch cfg.Log.Level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Logger = zerolog.New(w).With().Timestamp().Logger()
}

func setupMiddleware(e *echo.Echo, cfg config.Config) {
	e.Use(auth.UserAuthMiddleware(cfg))

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")

			return nil
		},
	}))
}

func main() {
	// Set the global logging level to Info
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// zerolog.SetGlobalLevel(zerolog.TraceLevel)

	// Load the configuration from the specified file
	var cfg config.Config
	var err error
	if cfg, err = config.LoadConfig(*configFile); err != nil {
		// If the configuration file can't be loaded, panic
		panic("failed to load config file")
	}

	setupGlobalLogger(cfg)

	// Create the service object, passing the loaded configuration
	svc := service.NewService(cfg)

	// Create the handler object, passing the created service object
	h := handler.NewHandler(svc)

	// Create a new Echo instance
	e := echo.New()

	setupMiddleware(e, cfg)

	// Set up the routes for the Echo instance using the handler object
	h.SetupRouter(e)

	// Start the Echo server, using the address specified in the configuration
	// Any errors will be logged
	e.Logger.Fatal(e.Start(cfg.EchoAddress()))
}
