package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/luispfcanales/daemon-daa/internal/core/domain"
)

// Estructuras para el parseo JSON
type PowerShellStateResult struct {
	WebsiteState string `json:"website_state"`
	AppPoolState string `json:"apppool_state"`
	SiteName     string `json:"site_name"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

type IISService struct {
}

func NewIISService() *IISService { // ✅ Sin parámetros
	return &IISService{}
}

// ✅ Obtener todos los sitios disponibles
func (s *IISService) GetAllSites() ([]map[string]interface{}, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("IIS solo disponible en Windows")
	}

	script := `
		Get-IISSite | Select-Object Name, State, Id, @{
			Name="URL";
			Expression={
				$bindings = $_.Bindings | ForEach-Object {
					$protocol = $_.Protocol
					$bindingInfo = $_.BindingInformation

					$parts = $bindingInfo -split ':'
					$ip = $parts[0]
					$port = $parts[1]
					$hostname = $parts[2]

					if ($hostname) {
						"${protocol}://${hostname}:${port}"
					} elseif ($ip -and $ip -ne "*") {
						"${protocol}://${ip}:${port}"
					} else {
						"${protocol}://localhost:${port}"
					}
				}
				$bindings -join ", "
			}
		} | ConvertTo-Json -Depth 3
	`

	cmd := exec.Command("powershell", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error ejecutando PowerShell: %v - %s", err, stderr.String())
	}

	var sites []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout.String()), &sites); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %v", err)
	}

	return sites, nil
}

func (s *IISService) GetAllSitesWithContext(ctx context.Context) ([]map[string]interface{}, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("IIS solo disponible en Windows")
	}

	script := `
		Get-IISSite | Select-Object Name, State, Id, @{
			Name="URL";
			Expression={
				$bindings = $_.Bindings | ForEach-Object {
					$protocol = $_.Protocol
					$bindingInfo = $_.BindingInformation

					$parts = $bindingInfo -split ':'
					$ip = $parts[0]
					$port = $parts[1]
					$hostname = $parts[2]

					if ($hostname) {
						"${protocol}://${hostname}:${port}"
					} elseif ($ip -and $ip -ne "*") {
						"${protocol}://${ip}:${port}"
					} else {
						"${protocol}://localhost:${port}"
					}
				}
				$bindings -join ", "
			}
		} | ConvertTo-Json -Depth 3
	`

	// Crear comando con contexto
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Ejecutar en goroutine para mejor control
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Esperar por finalización o cancelación
	select {
	case err := <-done:
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, context.Canceled
			}
			return nil, fmt.Errorf("error ejecutando PowerShell: %v - %s", err, stderr.String())
		}
	case <-ctx.Done():
		// Intentar terminar el proceso más agresivamente
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, context.Canceled
	}

	// Verificar una última vez si el contexto fue cancelado
	if ctx.Err() != nil {
		return nil, context.Canceled
	}

	var sites []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout.String()), &sites); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %v", err)
	}

	return sites, nil
}

// ✅ Controlar un sitio específico por nombre
func (s *IISService) ControlSite(siteName string, action string) (*domain.ControlSiteResult, error) {
	if runtime.GOOS != "windows" {
		return &domain.ControlSiteResult{}, fmt.Errorf("IIS solo disponible en Windows")
	}

	var commands []string

	state, err := s.getSiteState(siteName)
	if err != nil {
		return &domain.ControlSiteResult{}, fmt.Errorf("error obteniendo estado: %v", err)
	}

	switch strings.ToLower(action) {
	case "stop":
		if state.IsStopped() {
			return &domain.ControlSiteResult{
				IISSite:    siteName,
				IISAction:  domain.GetIISStateName(domain.IISStateStopped),
				IISSuccess: true,
				IISOutput:  "El sitio ya está detenido",
			}, nil
		}
		commands = []string{
			fmt.Sprintf("Stop-Website -Name \"%s\"", siteName),
			fmt.Sprintf("Stop-WebAppPool -Name \"%s\"", siteName),
		}
	case "start":
		if state.IsRunning() {
			return &domain.ControlSiteResult{
				IISSite:    siteName,
				IISAction:  domain.GetIISStateName(domain.IISStateStarted),
				IISSuccess: true,
				IISOutput:  "El sitio ya está en ejecución",
			}, nil
		}
		commands = []string{
			fmt.Sprintf("Start-WebAppPool -Name \"%s\"", siteName),
			fmt.Sprintf("Start-Website -Name \"%s\"", siteName),
		}
	case "restart":
		commands = []string{
			fmt.Sprintf("Stop-Website -Name \"%s\"", siteName),
			fmt.Sprintf("Stop-WebAppPool -Name \"%s\"", siteName),
			"Start-Sleep -Seconds 2",
			fmt.Sprintf("Start-WebAppPool -Name \"%s\"", siteName),
			"Start-Sleep -Seconds 1",
			fmt.Sprintf("Start-Website -Name \"%s\"", siteName),
		}
	default:
		return &domain.ControlSiteResult{}, fmt.Errorf("acción no válida: %s. Use: start, stop, restart", action)
	}

	// Ejecutar comandos
	script := strings.Join(commands, "; ")
	cmd := exec.Command("powershell", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err = cmd.Run()
	duration := time.Since(startTime)

	result := &domain.ControlSiteResult{
		IISSite:      siteName,
		IISAction:    action,
		IISSuccess:   err == nil,
		IISOutput:    strings.TrimSpace(stdout.String()),
		IISTimestamp: startTime,
		IISDuration:  duration.String(),
	}

	if err != nil {
		errorMsg := strings.TrimSpace(stderr.String())
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		result.IISError = errorMsg
	}

	switch action {
	case "start":
		result.IISAction = domain.GetIISStateName(domain.IISStateStarted)
	case "stop":
		result.IISAction = domain.GetIISStateName(domain.IISStateStopped)
	case "restart":
	}

	return result, err
}

// Función mejorada con JSON
func (s *IISService) getSiteState(siteName string) (*domain.SiteState, error) {
	// Script PowerShell corregido
	script := fmt.Sprintf(`
		$siteName = "%s"
		try {
			# Importar el módulo necesario
			Import-Module WebAdministration -ErrorAction Stop
			
			$website = Get-Website -Name $siteName -ErrorAction Stop
			$appPool = Get-IISAppPool -Name $siteName -ErrorAction Stop
			
			$result = @{
				website_state = $website.State.ToString()
				apppool_state = $appPool.State.ToString()
				site_name = $siteName
				success = $true
			}
			$result | ConvertTo-Json -Compress
		} catch {
			$result = @{
				website_state = "NotFound"
				apppool_state = "NotFound"
				site_name = $siteName
				success = $false
				error = $_.Exception.Message
			}
			$result | ConvertTo-Json -Compress
		}
	`, siteName)

	cmd := exec.Command("powershell", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error ejecutando comando: %v, stderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())

	// Parsear la respuesta JSON
	var psResult struct {
		WebsiteState string `json:"website_state"`
		AppPoolState string `json:"apppool_state"`
		SiteName     string `json:"site_name"`
		Success      bool   `json:"success"`
		Error        string `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(output), &psResult); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %v, output: %s", err, output)
	}

	state := &domain.SiteState{
		SiteName:     psResult.SiteName,
		WebsiteState: psResult.WebsiteState,
		AppPoolState: psResult.AppPoolState,
	}

	return state, nil
}
