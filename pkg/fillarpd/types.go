package fillarpd

import (
	"net/netip"
	"strings"
)

type RouteFiller interface {
	AddRoute(netip.Addr) error
	RemoveRoute(netip.Addr) error
	// Perodicially we should give it a list of known Addrs (likely from sweep)
	// []net.Addr The known good IPs.
	PurgeUnused([]netip.Addr) error
}

type ARPScanner interface {
	// Return something we can stream from to watch the ARP requsts
	Scan() (*strings.Reader, error)
}

type Sweeper interface {
	FindIPs() ([]netip.Addr, error)
}
