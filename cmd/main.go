package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/actors"
	"github.com/luispfcanales/daemon-daa/internal/application/api"
	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/repositories"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"

	"github.com/anthdm/hollywood/actor"
)

func main() {
	fmt.Println("🚀 Iniciando Monitor de Dominios UNAMAD CON CONCURRENCIA")
	fmt.Println("=========================================================")

	// Configurar logger más detallado para ver concurrencia
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Cambiamos a Debug para ver más detalles
	})))

	// Crear repositorio
	repocsv := repositories.NewCSVDomainRepository("domain_configs.csv", "domain_checks.csv")

	eventBus := events.NewEventBus()
	iisService := services.NewIISService()

	// Configurar el engine de Hollywood
	config := actor.NewEngineConfig()
	engine, err := actor.NewEngine(config)
	if err != nil {
		panic(err)
	}

	// Crear actores
	monitorPID := engine.Spawn(actors.NewMonitorActor(repocsv), "monitor")
	loggerPID := engine.Spawn(actors.NewConsoleLogger(), "logger")

	// Suscribir el logger a los eventos
	engine.Subscribe(loggerPID)

	// Configurar y iniciar servidor HTTP
	router := api.NewRouter(engine, monitorPID, iisService, eventBus)
	mux := router.SetupRoutes()

	handler := api.CorsMiddleware(api.LoggingMiddleware(mux))
	server := &http.Server{
		Addr:        "0.0.0.0:8080",
		Handler:     handler,
		IdleTimeout: 120 * time.Second,
	}
	// Iniciar servidor HTTP en goroutine
	go func() {
		slog.Info("🌐 Servidor HTTP iniciado", "port", 8080)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error iniciando servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Manejar señales de sistema
	setupSignalHandler(engine)

	// Mantener el programa corriendo
	select {}
}

// Manejar señales del sistema para apagado limpio
func setupSignalHandler(_ *actor.Engine) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ch
		fmt.Println("\n🛑 Apagando monitor concurrente...")
		os.Exit(0)
	}()
}
