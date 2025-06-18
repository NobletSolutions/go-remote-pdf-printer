package app

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/prolegis/form-ai/internal/aws"
	"gitlab.com/prolegis/form-ai/internal/config"
	controllers "gitlab.com/prolegis/form-ai/internal/controller/v1"
	"gitlab.com/prolegis/form-ai/internal/db/mariadb"
	docSvc "gitlab.com/prolegis/form-ai/internal/services/document"
	"gitlab.com/prolegis/form-ai/internal/services/document/job_handlers"
	formTypes "gitlab.com/prolegis/form-ai/internal/services/form-types"
	"gitlab.com/prolegis/form-ai/internal/services/form-types/interfaces"
	"gitlab.com/prolegis/form-ai/internal/services/job"
	"gitlab.com/prolegis/form-ai/pkg/httpserver"
	"gitlab.com/prolegis/form-ai/pkg/httpserver/middleware"
	"gitlab.com/prolegis/form-ai/pkg/logger"
)

func RunApi(cfg *config.Config) {
	logger := logger.NewLogger(cfg.LogConfig.Level, cfg.LogConfig.Directory+"/api.log")

	logger.Info().
		Str("version", cfg.AppConfig.Version).
		Str("log_level", cfg.LogConfig.Level).
		Bool("debug_requests", cfg.DebugRequests()).
		Msg("server started")

	var err error
	fmt.Printf("DB URL: %s\n", cfg.DbConfig.Url)
	dbConn := ConnectToDb(cfg.DbConfig.Url, logger)
	defer func(dbConn *gorm.DB) {
		err := dbConn.DB().Close()
		if err != nil {
			panic("unable to close db connection: " + err.Error())
		}
	}(dbConn)

	gin.DisableConsoleColor()
	ginHandler := gin.Default()

	// Build Services
	docRepo := mariadb.NewDocumentRepository(dbConn)
	jobRepo := mariadb.NewJobRepository(dbConn)

	fileStore := aws.NewS3Service(cfg.AwsConfig.S3Config, logger)
	textractSvc := aws.NewTextractService(cfg.AwsConfig.TextractConfig, jobRepo, docRepo, &cfg.LogConfig.Directory, logger, cfg.DebugRequests())
	documentSrv := docSvc.NewDocumentService(docRepo, fileStore, textractSvc, logger)
	jobService := job.NewJobService(jobRepo, docRepo, cfg.LogConfig.Directory, logger, cfg.DebugRequests())

	jobTextractHandler := job_handlers.NewTextractTextDetectionHandler(docRepo, textractSvc, cfg.AppConfig.RootDirectory, logger)
	jobService.RegisterJobHandler(jobTextractHandler)

	jobHandlers := []interfaces.FormTypeHandler{formTypes.NewRealEstateFormHandler(logger), formTypes.NewMortgageInstructionsFormHandler(logger)}
	jobTextAnalysisHandler := job_handlers.NewTextractTextAnalysisHandler(docRepo, textractSvc, jobHandlers, logger, cfg.AppConfig.RootDirectory)
	jobService.RegisterJobHandler(jobTextAnalysisHandler)

	jobService.RegisterJobHandler(job_handlers.NewDefaultFailedHandler(docRepo, logger))

	// Add Router/Controller
	controllers.NewDocumentRouter(ginHandler, logger, documentSrv, jobService, cfg.LogConfig.Directory)
	ginHandler.SetTrustedProxies(nil)
	ginHandler.Use(middleware.RequestSizeLimiter(15 * 1048576)) // X * MB
	ginHandler.Use(location.Default())

	if cfg.DebugRequests() {
		ginHandler.Use(middleware.RequestLoggerMiddleware(cfg.LogConfig.Directory, logger))
	}

	// K8s probe for kubernetes health checks -.
	ginHandler.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK", "message": "The server is up and running"})
	})

	// Prometheus metrics
	ginHandler.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Handling a page not found endpoint -.
	ginHandler.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND", "message": "The requested page is not found.Please try later!"})
	})

	var httpServer *httpserver.Server
	if cfg.HTTPConfig.UseTls {
		httpServer = httpserver.New(ginHandler, httpserver.Address(cfg.HTTPConfig.Listen), httpserver.TlsConfig(cfg.HTTPConfig.CertPath, cfg.HTTPConfig.KeyPath))
	} else {
		httpServer = httpserver.New(ginHandler, httpserver.Address(cfg.HTTPConfig.Listen))
	}

	// Waiting signal -.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		logger.Info().Msgf("Exiting due to signal: %s", s.String())
	case err = <-httpServer.Notify():
		logger.Error().Err(err).Msg("exiting due to httpServer error")
	}

	logger.Info().Msgf("Closing %s (api) v%s", cfg.AppConfig.Name, cfg.AppConfig.Version)
}
