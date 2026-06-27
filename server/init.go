package server

import (
	"admin-stats/db"
	"admin-stats/repository"
	mongorepo "admin-stats/repository/mongo"
	"admin-stats/server/logger"
	"admin-stats/service"
	"context"
	"io"
	"log"
	"time"
)

// ---- Loggers ----------------------------------------------------------------

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
	s.Log = logger.NewZeroLogger(cw, fw)
	s.ForceLog = logger.NewForceLogger(fw)
}

// ---- Databases --------------------------------------------------------------

func (s *Server) InitDBs() {
	s.initMongoDB()
	s.initRedis()
	s.initRepos()
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

// ---- Repositories -----------------------------------------------------------

func (s *Server) initRepos() {
	s.Repos = &repository.Repos{
		Transaction: s.initTransactionRepo(),
		DailyStats:  s.initDailyStatsRepo(),
	}
}

func (s *Server) initTransactionRepo() *mongorepo.TransactionRepository {
	repo := mongorepo.NewTransactionRepository(s.MongoDB.DB)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		s.ForceLog.Error(err)
		log.Fatalf("failed to ensure transaction indexes: %v", err)
	}
	return repo
}

func (s *Server) initDailyStatsRepo() *mongorepo.DailyStatsRepository {
	repo := mongorepo.NewDailyStatsRepository(s.MongoDB.DB)
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		s.ForceLog.Error(err)
		log.Fatalf("failed to ensure daily_stats indexes: %v", err)
	}
	return repo
}

// ---- Services ---------------------------------------------------------------

func (s *Server) InitServices() {
	ttlHours := s.Config.RedisConfig.StatsCacheTTLHours
	cacheTTL := time.Duration(ttlHours) * time.Hour // 0h → NewStatsService defaults to 24h

	s.Services = service.NewServices(&service.ServicesOptions{
		Repos:    s.Repos,
		Log:      s.Log,
		Redis:    s.Redis.Client,
		CacheTTL: cacheTTL,
	})
}

// ---- Cron -------------------------------------------------------------------

func (s *Server) StartCron(ctx context.Context) {
	cron := NewCron(s.Services.Stats, s.Config.CronConfig.StatsRecomputeDays, s.Log)
	cron.Start(ctx)
}
