package api

import (
	"admin-stats/server/logger"
	"admin-stats/service"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

// API holds the router and all dependencies needed by handlers.
type API struct {
	router *mux.Router
	log    logger.Logger
	// forceLog is a logger that always logs to console, regardless of the configuration.
	forceLog               logger.Logger
	validator              *validator.Validate
	services               *service.Services
	internalAuthMiddleware mux.MiddlewareFunc
}

type Options struct {
	Router                 *mux.Router
	Log                    logger.Logger
	ForceLog               logger.Logger
	Validator              *validator.Validate
	Services               *service.Services
	InternalAuthMiddleware mux.MiddlewareFunc
}

func New(opts *Options) *API {
	a := &API{
		router:                 opts.Router,
		log:                    opts.Log,
		forceLog:               opts.ForceLog,
		validator:              opts.Validator,
		services:               opts.Services,
		internalAuthMiddleware: opts.InternalAuthMiddleware,
	}
	a.registerRoutes()
	return a
}
