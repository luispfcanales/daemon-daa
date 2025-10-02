package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/actors"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/repositories"

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
	repo := repositories.NewInMemoryDomainRepository()

	// Configurar el engine de Hollywood
	config := actor.NewEngineConfig()
	engine, err := actor.NewEngine(config)
	if err != nil {
		panic(err)
	}

	// Crear actores
	monitorPID := engine.Spawn(actors.NewMonitorActor(repo), "monitor")
	loggerPID := engine.Spawn(actors.NewConsoleLogger(), "logger")

	// Suscribir el logger a los eventos
	engine.Subscribe(loggerPID)

	// Manejar señales de sistema
	setupSignalHandler(engine)

	// Verificación inicial CONCURRENTE
	fmt.Println("\n🔍 Verificación inicial CONCURRENTE de dominios:")
	engine.Send(monitorPID, actors.CheckAllDomains{})

	// Iniciar monitoreo automático cada 30 segundos
	time.Sleep(3 * time.Second) // Esperar que termine la verificación inicial
	fmt.Println("\n🔄 Iniciando monitoreo automático concurrente cada 30 segundos...")
	engine.Send(monitorPID, actors.StartMonitoring{Interval: 30})

	// Mantener el programa corriendo
	select {}
}

func setupSignalHandler(engine *actor.Engine) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ch
		fmt.Println("\n🛑 Apagando monitor concurrente...")
		os.Exit(0)
	}()
}
