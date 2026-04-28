package ipchecker

import "net"

type Static struct {
	IPs []string
}

func (s *Static) Contains(ip net.IP) bool {
	ipString := ip.String()

	for _, sip := range s.IPs {
		if sip == ipString {
			return true
		}
	}
	return false
}
