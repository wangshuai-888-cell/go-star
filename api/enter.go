package api

import "go-star/api/site_api"

type Api struct {
	SiteApi site_api.SiteApi
}

var App = Api{}
