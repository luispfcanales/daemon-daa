package domain

import "time"

type Domain struct {
	Name      string
	IPs       []string
	Timestamp time.Time
}

type DomainCheck struct {
	Domain      string    `json:"domain,omitempty"`
	ExpectedIP  string    `json:"expected_ip,omitempty"`
	ActualIPs   []string  `json:"actual_ips,omitempty"`
	IsValid     bool      `json:"is_valid,omitempty"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	DurationMs  float64   `json:"duration_ms"`
	RequestTime float64   `json:"request_time"`
}

type DomainConfig struct {
	Domain     string
	ExpectedIP string
}

type StatsDomain struct {
	DNS              string    `json:"dns"`
	TotalChecks      int       `json:"total_checks"`
	SuccessRate      float64   `json:"success_rate"`
	AverageUptime    float64   `json:"average_uptime"`
	LastCheck        time.Time `json:"last_check"`
	AvgResponseTime  float64   `json:"avg_response_time"`
	MinResponseTime  float64   `json:"min_response_time"`
	MaxResponseTime  float64   `json:"max_response_time"`
	P95ResponseTime  float64   `json:"p95_response_time"`
	SuccessCount     int       `json:"success_count"`
	FailureCount     int       `json:"failure_count"`
	ChecksWithTiming int       `json:"checks_with_timing"`
}
