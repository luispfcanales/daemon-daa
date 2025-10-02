package adapters

import (
	"fmt"
	"net"
)

type DNSResolver struct{}

func NewDNSResolver() *DNSResolver {
	return &DNSResolver{}
}

func (d *DNSResolver) ResolveIP(domain string) ([]string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", domain, err)
	}

	var ipStrings []string
	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.String())
	}

	return ipStrings, nil
}
