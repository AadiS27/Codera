package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/db"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/platform/logger"
	"github.com/codera/code-executor/internal/queue"
	"github.com/codera/code-executor/internal/sandbox/docker"
	"github.com/codera/code-executor/internal/server"
	"github.com/codera/code-executor/internal/worker"
	"github.com/codera/code-executor/internal/recovery"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/language/java"
	"github.com/codera/code-executor/internal/language/python"
	"github.com/codera/code-executor/internal/language/go"
	"github.com/codera/code-executor/internal/language/cpp"
	"github.com/codera/code-executor/internal/sandbox"
	"github.com/codera/code-executor/internal/repository"
	"github.com/codera/code-executor/internal/judge"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Cannot use slog yet, use standard print for bootstrap failure
		os.Stdout.WriteString("Failed to load configuration: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize structured logging
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	// Initialize Sandbox
	sb := docker.NewRuntime(cfg)

	// Initialize Language Registry
	langRegistry := language.NewMapRegistry()
	
	// Load language profiles
	javaProfile, _ := sandbox.GetProfileForLanguage("java")
	pythonProfile, _ := sandbox.GetProfileForLanguage("python")
	goProfile, _ := sandbox.GetProfileForLanguage("go")
	cppProfile, _ := sandbox.GetProfileForLanguage("cpp")
	
	// Register Language Executors
	langRegistry.Register(java.NewExecutor(javaProfile))
	langRegistry.Register(python.NewExecutor(pythonProfile))
	langRegistry.Register(golang.NewExecutor(goProfile))
	langRegistry.Register(cpp.NewExecutor(cppProfile))

	// Initialize Execution Service
	execService := execution.NewService(cfg, langRegistry, sb)

	// Connect to Database
	database, err := db.Connect(context.Background(), cfg, log)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run Database Migrations
	if err := db.RunMigrations(cfg.DatabaseURL, log); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisDB, err := db.ConnectRedis(context.Background(), cfg, log)
	if err != nil {
		log.Error("Failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisDB.Close()

	// Initialize Jobs layer with Postgres and Redis
	jobStore := jobs.NewPostgresJobStore(database.Pool)
	redisQueue := queue.NewRedisQueue(redisDB.Client, cfg.RedisStream, cfg.RedisConsumerGroup, cfg.RedisStreamMaxLen)
	
	// Create consumer group safely
	if err := redisQueue.EnsureGroupExists(context.Background()); err != nil {
		log.Error("Failed to initialize redis consumer group", "error", err)
		os.Exit(1)
	}

	jobService := jobs.NewService(jobStore, redisQueue, langRegistry)

	role := cfg.Role
	log.Info("Starting application", "role", role, "instance_id", cfg.InstanceID)

	// Initialize Repositories
	probRepo := repository.NewPostgresProblemRepository(database.Pool)
	testCaseRepo := repository.NewPostgresTestCaseRepository(database.Pool)
	subRepo := repository.NewPostgresSubmissionRepository(database.Pool)

	// Initialize Judge Engine
	compRegistry := judge.NewComparatorRegistry()
	compRegistry.Register(domain.ComparisonModeExact, &judge.ExactComparator{})
	compRegistry.Register(domain.ComparisonModeWhitespace, &judge.WhitespaceComparator{})
	
	judgeEngine := judge.NewEngine(cfg, langRegistry, sb, compRegistry, subRepo, probRepo, testCaseRepo)

	var workerPool *worker.Pool
	if role == "worker" || role == "all" {
		// Initialize and Start Worker Pool
		workerPool = worker.NewPool(log, redisQueue, jobStore, judgeEngine, execService, cfg.ExecutionWorkers, cfg.InstanceID, cfg.WorkerHeartbeatInterval, cfg.WorkerShutdownTimeout)
		workerPool.Start(context.Background())

		// Start Pending Message Recovery
		pendingRecovery := jobs.NewPendingRecovery(log, redisDB.Client, jobStore, redisQueue, cfg.RedisStream, cfg.RedisConsumerGroup, cfg.RedisPendingIdleTimeout, cfg.RedisPendingClaimBatchSize)
		go pendingRecovery.Start(context.Background())
	}

	var srv *server.Server
	serverErrors := make(chan error, 1)
	if role == "api" || role == "all" {
		// Start Background Recovery Service
		recoverySvc := recovery.NewService(database.Pool, log, redisQueue, cfg.RecoveryInterval, cfg.WorkerHeartbeatTimeout)
		go recoverySvc.Start(context.Background())

		// Initialize server handlers
		probHandler := server.NewProblemHandler(probRepo, testCaseRepo)
		subHandler := server.NewSubmissionHandler(subRepo, probRepo, redisQueue, langRegistry)
		runHandler := server.NewRunHandler(jobService, langRegistry)

		// Initialize server
		srv = server.New(cfg, log, jobService, probHandler, subHandler, runHandler, database, redisDB)

		// Start the service listening for requests.
		go func() {
			log.Info("Starting server", "port", cfg.Port, "env", cfg.Env)
			serverErrors <- srv.Start()
		}()
	}

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		log.Info("Graceful shutdown started", "signal", sig)
		defer log.Info("Graceful shutdown completed", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if srv != nil {
			if err := srv.Shutdown(ctx); err != nil {
				log.Error("Graceful shutdown error", "error", err)
			}
		}

		if workerPool != nil {
			workerPool.Stop(ctx)
		}
	}
}
