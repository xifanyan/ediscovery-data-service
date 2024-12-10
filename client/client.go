package client

import (
	"github.com/xifanyan/ediscovery-data-service/config"

	"github.com/xifanyan/adp"
)

func NewADPClient(cfg config.Config) *adp.Client {
	return adp.NewClientBuilder().
		WithDomain(cfg.ADP.Domain).
		WithPort(cfg.ADP.Port).
		WithUser(cfg.ADP.User).
		WithPassword(cfg.ADP.Password).
		Build()
}
