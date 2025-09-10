package service

import (
	"github.com/rs/zerolog/log"
	"github.com/xifanyan/ediscovery-data-service/client"
	"github.com/xifanyan/ediscovery-data-service/config"

	"github.com/labstack/echo/v4"
	adp "github.com/xifanyan/adp"
	// searchwebapi "github.com/xifanyan/kiota-searchwebapi"
)

type Service struct {
	cfg    config.Config
	ADPsvc *adp.Service
	// SWAClient *searchwebapi.Client
}

func NewService(config config.Config) *Service {
	return &Service{
		cfg:    config,
		ADPsvc: &adp.Service{ADPClient: client.NewADPClient(config)},
		// SWAClient: searchwebapi.NewClient(config.SearchWebAPI.Domain, config.SearchWebAPI.Port, config.SearchWebAPI.Endpoint),
	}
}

func (s *Service) ResetADPServiceWithContextCredential(c echo.Context) *adp.Service {
	// Extract user and password from context
	user, userOk := c.Get("adp_user").(string)
	password, passwordOk := c.Get("adp_password").(string)

	// If both credentials are provided and valid, reset them
	if userOk && user != "" && passwordOk && password != "" {
		log.Debug().Msgf("Resetting ADP credentials: user=%s", user)
		s.ADPsvc.ADPClient.ResetCredentials(user, password)
	}

	return s.ADPsvc
}
