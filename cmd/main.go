package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"windsurf-gateway/internal/config"
	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/handler"
	"windsurf-gateway/internal/logger"
	"windsurf-gateway/internal/middleware"
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(&logger.Config{
		Enabled: cfg.Log.Enabled,
		Level:   cfg.Log.Level,
		Format:  cfg.Log.Format,
	})
	defer logger.Sync()

	gin.SetMode(cfg.Server.Mode)

	db, err := database.InitMySQL(&cfg.Database.MySQL)
	if err != nil {
		logger.Fatalf("Failed to init MySQL: %v", err)
	}

	rdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		logger.Fatalf("Failed to init Redis: %v", err)
	}

	services := service.NewServices(db, rdb, cfg)

	if err := services.Auth.CreateDefaultAdmin(); err != nil {
		logger.Warnf("Failed to create default admin: %v", err)
	}

	handlers := handler.NewHandlers(services, cfg)

	router := setupRouter(cfg, handlers)

	server := &http.Server{
		Addr:         cfg.Server.GetServerAddr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Infof("Windsurf Gateway starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("Server forced shutdown: %v", err)
	}

	logger.Info("Server stopped")
}

func setupRouter(cfg *config.Config, handlers *handler.Handlers) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(cfg))

	router.Any("/proxy/*path", handlers.Proxy.ForwardWithUserToken)

	router.Any("/api/proxy/*path", handlers.Proxy.ForwardWithSystemToken)

	api := router.Group(cfg.Frontend.APIPrefix)
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Auth.Login)
			auth.POST("/logout", handlers.Auth.Logout)
		}

		userAuth := api.Group("/user-auth")
		{
			userAuth.POST("/register", handlers.UserAuth.Register)
			userAuth.POST("/login", handlers.UserAuth.Login)
			userAuth.POST("/logout", handlers.UserAuth.Logout)
			userAuth.POST("/refresh", handlers.UserAuth.Refresh)
		}

		api.GET("/invitation-codes/validate", handlers.InvitationCode.Validate)
		api.GET("/system/version", handlers.System.GetVersion)

		user := api.Group("/user")
		user.Use(handlers.UserAuth.UserAuthMiddleware())
		{
			user.GET("/me", handlers.UserAuth.Me)
			user.PUT("/profile", handlers.UserAuth.UpdateProfile)
			user.POST("/change-password", handlers.UserAuth.ChangePassword)
			user.POST("/regenerate-token", handlers.UserAuth.RegenerateAPIToken)
			user.GET("/settings", handlers.UserAuth.GetUserSettings)
			user.PUT("/settings", handlers.UserAuth.UpdateUserSettings)
		}

		protected := api.Group("")
		protected.Use(handlers.Auth.AuthMiddleware())
		{
			protected.GET("/auth/me", handlers.Auth.Me)
			protected.PUT("/auth/profile", handlers.Auth.UpdateProfile)

			tokens := protected.Group("/tokens")
			{
				tokens.GET("", handlers.Token.List)
				tokens.POST("", handlers.Token.Create)
				tokens.GET("/:id", handlers.Token.Get)
				tokens.PUT("/:id", handlers.Token.Update)
				tokens.DELETE("/:id", handlers.Token.Delete)
				tokens.GET("/validate", handlers.Token.Validate)
				tokens.GET("/stats", handlers.Token.Stats)
				tokens.POST("/batch-import", handlers.Token.BatchImport)
				tokens.GET("/:id/users", handlers.Token.GetTokenUsers)
				tokens.GET("/:id/ban-reason", handlers.Token.GetBanReason)
				tokens.POST("/batch-refresh-auth-session", handlers.Token.BatchRefreshAuthSession)
			}

			users := protected.Group("/users")
			{
				users.GET("", handlers.UserAuth.ListUsers)
				users.PUT("/:id", handlers.UserAuth.AdminUpdateUser)
				users.POST("/:id/ban", handlers.UserAuth.BanUser)
				users.POST("/:id/unban", handlers.UserAuth.UnbanUser)
				users.POST("/:id/toggle-shared", handlers.UserAuth.ToggleSharedPermission)
			}

			stats := protected.Group("/stats")
			{
				stats.GET("/overview", handlers.Stats.Overview)
				stats.GET("/trend", handlers.Stats.Trend)
				stats.GET("/tokens/:id", handlers.Stats.TokenStats)
				stats.GET("/usage", handlers.Stats.Usage)
				stats.POST("/cleanup", handlers.Stats.Cleanup)
			}

			requestRecords := protected.Group("/request-records")
			{
				requestRecords.GET("", handlers.RequestRecord.List)
				requestRecords.GET("/search", handlers.RequestRecord.Search)
			}

			invitationCodes := protected.Group("/invitation-codes")
			{
				invitationCodes.GET("", handlers.InvitationCode.List)
				invitationCodes.POST("/generate", handlers.InvitationCode.Generate)
				invitationCodes.DELETE("/:id", handlers.InvitationCode.Delete)
			}

			systemConfig := protected.Group("/system-config")
			{
				systemConfig.GET("", handlers.SystemConfig.GetSystemConfig)
				systemConfig.PUT("", handlers.SystemConfig.UpdateSystemConfig)
				systemConfig.GET("/stats", handlers.SystemConfig.GetSystemStats)
			}
		}
	}

	router.Static("/assets", cfg.Frontend.StaticPath+"/assets")
	router.StaticFile("/favicon.ico", cfg.Frontend.StaticPath+"/favicon.ico")

	router.GET("/admin", func(c *gin.Context) {
		c.File(cfg.Frontend.StaticPath + "/index.html")
	})
	router.GET("/admin/*path", func(c *gin.Context) {
		c.File(cfg.Frontend.StaticPath + "/index.html")
	})

	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}
		c.File(cfg.Frontend.StaticPath + "/index.html")
	})

	return router
}
