package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/application/actors"
	"github.com/luispfcanales/daemon-daa/internal/application/api"
	"github.com/luispfcanales/daemon-daa/internal/application/events"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/repositories"
	"github.com/luispfcanales/daemon-daa/internal/infrastructure/services"

	"github.com/anthdm/hollywood/actor"
)

const serviceName = "IISMonitor"

func main() {
	// Verificar si se ejecuta como servicio de Windows
	isService, err := isWindowsService()
	if err != nil {
		log.Fatal("Error verificando modo de ejecución:", err)
	}

	if isService {
		// Ejecutar como servicio de Windows
		if err := runAsService(serviceName); err != nil {
			log.Fatal("Error ejecutando servicio:", err)
		}
	} else {
		// Ejecutar como aplicación de consola
		fmt.Println("🚀 Ejecutando en modo CONSOLA")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Manejar Ctrl+C
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			<-ch
			fmt.Println("\n🛑 Deteniendo aplicación...")
			cancel()
		}()

		if err := runApplication(ctx); err != nil {
			log.Fatal("Error ejecutando aplicación:", err)
		}
	}
}

// runApplication contiene la lógica principal de tu aplicación
func runApplication(ctx context.Context) error {
	// OBTENER RUTA ABSOLUTA del ejecutable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error obteniendo ruta del ejecutable: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Cambiar al directorio del ejecutable
	if err := os.Chdir(exeDir); err != nil {
		return fmt.Errorf("error cambiando directorio: %w", err)
	}

	fmt.Println("🚀 Iniciando Monitor de Dominios UNAMAD CON CONCURRENCIA")
	fmt.Println("=========================================================")
	fmt.Println("Directorio de trabajo:", exeDir)

	// Configurar logs con ruta absoluta
	logPath := filepath.Join(exeDir, "service.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error creando archivo de log: %w", err)
	}
	defer logFile.Close()

	// Configurar logger para archivo Y consola
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	slog.SetDefault(slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Iniciando aplicación", "directorio", exeDir)

	// Crear repositorio con rutas absolutas
	configPath := filepath.Join(exeDir, "domain_configs.csv")
	checksPath := filepath.Join(exeDir, "domain_checks.csv")

	slog.Info("Usando archivos", "config", configPath, "checks", checksPath)

	repocsv := repositories.NewCSVDomainRepository(configPath, checksPath)

	eventBus := events.NewEventBus()
	iisService := services.NewIISService()

	// Configurar el engine de Hollywood
	config := actor.NewEngineConfig()
	engine, err := actor.NewEngine(config)
	if err != nil {
		return fmt.Errorf("error creando engine de actores: %w", err)
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

	port := getPort()
	server := &http.Server{
		Addr:        fmt.Sprintf("0.0.0.0:%s", port),
		Handler:     handler,
		IdleTimeout: 120 * time.Second,
	}

	// Iniciar servidor HTTP en goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		slog.Info("🌐 Servidor HTTP iniciado", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	slog.Info("✅ Aplicación iniciada correctamente")
	slog.Info("📊 Endpoints disponibles:")
	slog.Info("   - http://localhost:" + port + "/monitoring/events")
	slog.Info("   - http://localhost:" + port + "/monitoring/control")
	slog.Info("   - http://localhost:" + port + "/iis/sites")
	slog.Info("   - http://localhost:" + port + "/iis/control")

	// Esperar señal de cancelación o error
	select {
	case <-ctx.Done():
		slog.Info("🛑 Recibida señal de apagado...")

		// Shutdown graceful del servidor HTTP
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error en shutdown del servidor", "error", err)
		}

		slog.Info("✅ Servidor HTTP detenido")
		return nil

	case err := <-serverErrChan:
		return fmt.Errorf("error en servidor HTTP: %w", err)
	}
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}
