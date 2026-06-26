package server

import (
	"admin-stats/api"
	"admin-stats/config"
	"admin-stats/db"
	mongorepo "admin-stats/repository/mongo"
	"admin-stats/server/logger"
	"admin-stats/server/middleware"
	"admin-stats/service"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
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
	Services   *service.Services
	API        *api.API
}

func (s *Server) InitLoggers() {
	var cw, fw io.Writer
	if s.Config.LoggerConfig.EnableFileLogger {
		fw = logger.NewFileWriter(
			s.Config.LoggerConfig.FileLoggerConfig.FileName,
			s.Config.LoggerConfig.FileLoggerConfig.Path,
		)
	}
	if s.Config.LoggerConfig.EnableConsoleLogger {
		cw = logger.NewZeroLogConsoleWriter(logger.NewStandardConsoleWriter())
	}
	s.Log = logger.NewLogger(cw, fw)
	s.ForceLog = logger.NewForceLogger(fw)
}

func (s *Server) InitDBs() {
	s.initMongoDB()
	s.initRedis()
}

func (s *Server) InitServices() {
	repo := mongorepo.NewTransactionRepository(s.MongoDB.DB)
	s.Services = service.NewServices(&service.ServicesOptions{
		TransactionRepo: repo,
		Log:             s.Log,
	})
}

func (s *Server) initMongoDB() {
	mongoDB := db.NewMongoDB(s.Config.MongoConfig)
	if err := mongoDB.Connect(context.Background()); err != nil {
		s.ForceLog.Error(err)
		log.Fatalf("failed to connect to mongodb: %v", err)
	}
	s.MongoDB = mongoDB
}

func (s *Server) initRedis() {
	redisDB := db.NewRedisDB(s.Config.RedisConfig)
	if err := redisDB.Connect(context.Background()); err != nil {
		s.ForceLog.Error(err)
		log.Fatalf("failed to connect to redis: %v", err)
	}
	s.Redis = redisDB
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

	server.API = api.New(&api.Options{
		Router:                 r,
		Log:                    server.Log,
		ForceLog:               server.ForceLog,
		Validator:              validator.New(),
		Services:               server.Services,
		InternalAuthMiddleware: middleware.InternalAuth(server.Log, server.Config.AuthConfig.InternalToken),
	})

	return server
}
