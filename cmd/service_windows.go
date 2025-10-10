//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type windowsService struct {
	stopFunc context.CancelFunc
}

func (m *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Crear contexto cancelable
	ctx, cancel := context.WithCancel(context.Background())
	m.stopFunc = cancel

	// Iniciar aplicación en goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- runApplication(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	slog.Info("Servicio Windows iniciado correctamente")

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				slog.Info("Recibida señal de detención del servicio")
				changes <- svc.Status{State: svc.StopPending}
				cancel() // Cancelar contexto
				break loop
			default:
				slog.Warn("Comando no esperado del servicio", "cmd", c.Cmd)
			}
		case err := <-errChan:
			if err != nil {
				slog.Error("Error en la aplicación", "error", err)
			}
			break loop
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return
}

func runAsService(name string) error {
	elog, err := eventlog.Open(name)
	if err != nil {
		return err
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Iniciando servicio %s", name))

	err = svc.Run(name, &windowsService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Error ejecutando servicio: %v", err))
		return err
	}

	elog.Info(1, fmt.Sprintf("Servicio %s detenido", name))
	return nil
}

func isWindowsService() (bool, error) {
	return svc.IsWindowsService()
}
