package service

import (
	"github.com/xifanyan/ediscovery-data-service/client"
	"github.com/xifanyan/ediscovery-data-service/config"

	adp "github.com/xifanyan/adp"
	// searchwebapi "github.com/xifanyan/kiota-searchwebapi"
)

type Service struct {
	ADPsvc *adp.Service
	// SWAClient *searchwebapi.Client
}

func NewService(config config.Config) *Service {
	return &Service{
		ADPsvc: &adp.Service{ADPClient: client.NewADPClient(config)},
		// SWAClient: searchwebapi.NewClient(config.SearchWebAPI.Domain, config.SearchWebAPI.Port, config.SearchWebAPI.Endpoint),
	}
}
