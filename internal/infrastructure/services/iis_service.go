package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type IISService struct {
	// ❌ ELIMINAR: websiteName y appPoolName fijos
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
        Get-IISSite | Select-Object Name, State, Id | ConvertTo-Json -Depth 2
    `

	cmd := exec.Command("powershell", "-Command", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error ejecutando PowerShell: %v - %s", err, stderr.String())
	}

	// Parsear el array JSON de sitios
	var sites []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout.String()), &sites); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %v", err)
	}

	return sites, nil
}

// ✅ Controlar un sitio específico por nombre
func (s *IISService) ControlSite(siteName string, action string) (map[string]interface{}, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("IIS solo disponible en Windows")
	}

	var commands []string
	switch strings.ToLower(action) {
	case "stop":
		commands = []string{
			fmt.Sprintf("Stop-Website -Name \"%s\"", siteName),
			fmt.Sprintf("Stop-WebAppPool -Name \"%s\"", siteName),
		}
	case "start":
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
		return nil, fmt.Errorf("acción no válida: %s. Use: start, stop, restart", action)
	}

	// Ejecutar comandos
	script := strings.Join(commands, "; ")
	cmd := exec.Command("powershell", "-Command", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	result := map[string]interface{}{
		"site":      siteName,
		"action":    action,
		"success":   err == nil,
		"output":    strings.TrimSpace(stdout.String()),
		"timestamp": time.Now(),
		"duration":  duration.String(),
	}

	if err != nil {
		errorMsg := strings.TrimSpace(stderr.String())
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		result["error"] = errorMsg
	}

	return result, err
}
