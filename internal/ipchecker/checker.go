package ipchecker

import "net"

type Checker interface {
	Contains(ip net.IP) bool
}
