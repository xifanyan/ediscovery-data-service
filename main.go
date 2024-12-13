package main

import (
	"flag"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/xifanyan/ediscovery-data-service/auth"
	"github.com/xifanyan/ediscovery-data-service/config"
	"github.com/xifanyan/ediscovery-data-service/handler"
	"github.com/xifanyan/ediscovery-data-service/service"
)

var (
	configFile = flag.String("config", "config.json", "config file")
)

func setupMiddleware(e *echo.Echo, cfg config.Config) {
	logger := zerolog.New(os.Stdout)
	e.Use(middleware.RequestLoggerWithConfig(
		middleware.RequestLoggerConfig{
			LogURI:    true,
			LogStatus: true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				logger.Info().
					Str("URI", v.URI).
					Int("status", v.Status).
					Msg("request")

				return nil
			},
		},
	))
	e.Use(auth.UserAuthMiddleware(cfg))
}

func main() {
	// Set the global logging level to Info
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	// Load the configuration from the specified file
	var cfg config.Config
	var err error
	if cfg, err = config.LoadConfig(*configFile); err != nil {
		// If the configuration file can't be loaded, panic
		panic("failed to load config file")
	}

	// Create the service object, passing the loaded configuration
	svc := service.NewService(cfg)

	// Create the handler object, passing the created service object
	h := handler.NewHandler(svc)

	// Create a new Echo instance
	e := echo.New()

	// Set up middleware for the Echo instance
	setupMiddleware(e, cfg)

	// Set up the routes for the Echo instance using the handler object
	h.SetupRouter(e)

	// Start the Echo server, using the address specified in the configuration
	// Any errors will be logged
	e.Logger.Fatal(e.Start(cfg.EchoAddress()))
}
