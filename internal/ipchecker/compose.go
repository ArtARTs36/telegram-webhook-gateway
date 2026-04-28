package ipchecker

import "net"

type Compose struct {
	checkers []Checker
}

func Wrap(checkers ...Checker) *Compose {
	return &Compose{checkers: checkers}
}

func (c *Compose) Contains(ip net.IP) bool {
	for _, checker := range c.checkers {
		if checker.Contains(ip) {
			return true
		}
	}
	return false
}
