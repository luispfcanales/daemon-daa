package domain

import "time"

// IIS Site States
const (
	IISStateStarting = 0 // Starting
	IISStateStarted  = 1 // Started
	IISStateStopping = 2 // Stopping
	IISStateStopped  = 3 // Stopped
)

// Estado nombres legibles
var IISStateNames = map[int]string{
	IISStateStarting: "Starting",
	IISStateStarted:  "Started",
	IISStateStopping: "Stopping",
	IISStateStopped:  "Stopped",
}

// GetStateName retorna el nombre legible del estado
func GetIISStateName(state int) string {
	if name, ok := IISStateNames[state]; ok {
		return name
	}
	return "Unknown"
}

type ControlSiteResult struct {
	IISSite       string    `json:"iis_site"`
	IISAction     string    `json:"iis_action"`
	IISSuccess    bool      `json:"iis_success"`
	IISOutput     string    `json:"iis_output,omitempty"`
	IISError      string    `json:"iis_error,omitempty"`
	IISTimestamp  time.Time `json:"iis_timestamp"`
	IISDuration   string    `json:"iis_duration"`
	IISInProgress bool      `json:"iis_in_progress,omitempty"`
}

type SiteState struct {
	WebsiteState string `json:"website_state"`
	AppPoolState string `json:"apppool_state"`
	SiteName     string `json:"site_name"`
}

// Método para verificar si el sitio está ejecutándose
func (s *SiteState) IsRunning() bool {
	return s.WebsiteState == "Started" && s.AppPoolState == "Started"
}

// Método para verificar si el sitio está detenido
func (s *SiteState) IsStopped() bool {
	return s.WebsiteState == "Stopped" && s.AppPoolState == "Stopped"
}
