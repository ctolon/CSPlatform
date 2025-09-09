package utils

import (
	"net"
	"strings"
)

func IPEqual(a, b string) bool {
	na := NormalizeIP(a)
	nb := NormalizeIP(b)
	if na == "" || nb == "" {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return na == nb
}

func NormalizeIP(ipAddr string) string {
	ipAddr = strings.TrimSpace(ipAddr)
	if host, _, err := net.SplitHostPort(ipAddr); err == nil {
		ipAddr = host
	}
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}
