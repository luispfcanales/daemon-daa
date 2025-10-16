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
	DNS              string    `json:"dns,omitempty"`
	TotalChecks      int       `json:"total_checks,omitempty"`
	SuccessRate      float64   `json:"success_rate,omitempty"`
	AverageUptime    float64   `json:"average_uptime,omitempty"`
	LastCheck        time.Time `json:"last_check,omitempty"`
	AvgResponseTime  float64   `json:"avg_response_time,omitempty"`
	MinResponseTime  float64   `json:"min_response_time,omitempty"`
	MaxResponseTime  float64   `json:"max_response_time,omitempty"`
	P95ResponseTime  float64   `json:"p95_response_time,omitempty"`
	SuccessCount     int       `json:"success_count,omitempty"`
	FailureCount     int       `json:"failure_count,omitempty"`
	ChecksWithTiming int       `json:"checks_with_timing,omitempty"`
}
