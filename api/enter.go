package api

import (
	"go-star/api/log_api"
	"go-star/api/site_api"
)

type Api struct {
	SiteApi site_api.SiteApi
	LogApi  log_api.LogApi
}

var App = Api{}
