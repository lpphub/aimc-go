package server

import (
	"aimc-go/app/infra"
	"aimc-go/app/modules/auth"
	"aimc-go/app/modules/core"
	"aimc-go/app/modules/post"
	"aimc-go/app/modules/user"
	"aimc-go/app/server/middleware"
	"aimc-go/assistant/server"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lpphub/goweb/ext/logx"
	"github.com/lpphub/goweb/monitor"
)

type App struct {
	engine *gin.Engine
	server *http.Server
}

func New() *App {
	return &App{engine: gin.New()}
}

func (a *App) Run() {
	a.init()
	a.start()
	a.waitForShutdown()
}

func (a *App) init() {
	if err := infra.Init(); err != nil {
		panic(fmt.Sprintf("Failed to init: %v", err))
	}
	a.setupRouter()
}

func (a *App) setupRouter() {
	r := a.engine

	r.Use(gin.Recovery())
	r.Use(logx.GinAccessLog(logx.WithSkipPaths("/metrics", "/health")))
	r.Use(middleware.Cors())

	monitor.RegisterMetrics(r)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	for _, m := range a.registerModules() {
		m.Routes(r.Group(""))
	}
}

func (a *App) registerModules() []core.Module {
	registry := core.NewRegistry()

	userMod := user.New(infra.DB)
	authMod := auth.New(userMod.Service)
	postMod := post.New(infra.DB)
	assistantMod, err := server.NewSSE()
	if err != nil {
		panic(fmt.Sprintf("Failed to init assistant module: %v", err))
	}

	registry.Register(userMod, authMod, postMod, assistantMod)
	return registry.Modules()
}

func (a *App) start() {
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", infra.Cfg.Server.Port),
		Handler: a.engine,
	}

	go func() {
		log.Printf("Server starting on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server start failed: %v", err)
		}
	}()
}

func (a *App) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server shutdown completed")
	}
}
