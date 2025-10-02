package domain

import "time"

type Domain struct {
	Name      string
	IPs       []string
	Timestamp time.Time
}

type DomainCheck struct {
	Domain     string
	ExpectedIP string
	ActualIPs  []string
	IsValid    bool
	Error      string
	Timestamp  time.Time
}

type DomainConfig struct {
	Domain     string
	ExpectedIP string
}
