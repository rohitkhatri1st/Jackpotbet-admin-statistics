package server

import (
	"admin-stats/api"
	"admin-stats/config"
	"admin-stats/db"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"admin-stats/server/middleware"
	"admin-stats/server/validator"
	"admin-stats/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Server struct {
	httpServer *http.Server
	Router     *mux.Router
	Log        logger.Logger
	ForceLog   logger.Logger
	Config     config.Config
	MongoDB    *db.MongoDB
	Redis      *db.RedisDB
	Repos      *repository.Repos
	Services   *service.Services
	API        *api.API
}

func (s *Server) StartServer() {
	s.Router.Use(
		gorillahandlers.RecoveryHandler(gorillahandlers.PrintRecoveryStack(true)),
		middleware.CORS(s.Config.CORSConfig),
	)

	if s.Config.ServerConfig.EnableRequestLogging {
		s.Router.Use(middleware.RequestLogger(s.Log.With("type", "request")))
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Config.ServerConfig.Port),
		Handler:      s.Router,
		ReadTimeout:  time.Duration(s.Config.ServerConfig.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.ServerConfig.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.ServerConfig.IdleTimeout) * time.Second,
	}

	go func() {
		s.ForceLog.Info("msg", "Server Listening", "port", s.Config.ServerConfig.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.ForceLog.Error(err)
			log.Fatalf("server stopped unexpectedly: %v", err)
		}
	}()
}

func (s *Server) StopServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.ForceLog.Error(err)
	}
	if err := s.MongoDB.Disconnect(context.Background()); err != nil {
		s.ForceLog.Error(err)
	}
	if err := s.Redis.Disconnect(context.Background()); err != nil {
		s.ForceLog.Error(err)
	}
}

func NewServer() *Server {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := mux.NewRouter()
	server := &Server{
		httpServer: &http.Server{},
		Router:     r,
		Config:     cfg,
	}

	server.InitLoggers()
	server.InitDBs()
	server.InitServices()
	server.validate()

	server.API = api.New(&api.Options{
		Router:                 r,
		Log:                    server.Log,
		ForceLog:               server.ForceLog,
		Validator:              validator.NewValidator(),
		Services:               server.Services,
		InternalAuthMiddleware: middleware.InternalAuth(server.Log, server.Config.AuthConfig.InternalToken),
	})

	return server
}

// validate fatals on startup if any field in Repos or Services was added but not initialized.
// This catches the "added a new repo/service field but forgot to wire it" mistake immediately.
func (s *Server) validate() {
	checkNilFields(s.Repos, "Repos")
	checkNilFields(s.Services, "Services")
}

func checkNilFields(v any, structName string) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			log.Fatalf("%s is not initialized", structName)
		}
		rv = rv.Elem()
	}
	rt := rv.Type()
	for i := range rv.NumField() {
		field := rv.Field(i)
		fieldName := rt.Field(i).Name
		switch field.Kind() {
		case reflect.Ptr:
			if field.IsNil() {
				log.Fatalf("%s.%s is not initialized — did you forget to wire it in init.go?", structName, fieldName)
			}
		case reflect.Interface:
			if field.IsNil() {
				log.Fatalf("%s.%s is not initialized — did you forget to wire it in init.go?", structName, fieldName)
			}
			// Guard against typed nils: interface is non-nil but wraps a nil pointer.
			// e.g. var r repository.TransactionRepository = (*mongo.TransactionRepository)(nil)
			if elem := field.Elem(); elem.Kind() == reflect.Ptr && elem.IsNil() {
				log.Fatalf("%s.%s is a typed nil — its init function returned nil instead of a real value", structName, fieldName)
			}
		}
	}
}
