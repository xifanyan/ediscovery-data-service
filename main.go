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

// setupLogWriter sets up the log writer with the provided configuration.
//
// This function first ensures the directory containing the log file exists
// and then opens the log file in append mode. If the log file cannot be
// opened, it returns an error.
//
// If the configuration specifies that logs should also be written to the
// console, it returns an io.MultiWriter that writes to both the log file
// and the console.
//
// Parameters:
//   cfg (config.Config) - The configuration containing log settings.
//
// Returns:
//   io.Writer - The writer to use for logging.
//   error - An error if the log file cannot be opened.

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

// setupGlobalLogger sets up the global logger with the provided configuration.
//
// This function configures the zerolog global settings, sets up the log writer,
// and applies the log level from the configuration.
//
// Parameters:
//
//	cfg (config.Config) - The configuration containing log settings.
//
// Returns:
func setupGlobalLogger(cfg config.Config) {
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

// setupMiddleware configures middleware for the Echo instance.
//
// This function adds middleware for user authentication and request logging.
// The user authentication middleware ensures that requests are authenticated
// based on the provided configuration. The request logging middleware logs
// the URI and status of each request using the zerolog logger.
//
// Parameters:
//   e (echo.Echo) - The Echo instance to configure middleware for.
//   cfg (config.Config) - The configuration containing settings for authentication.

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
