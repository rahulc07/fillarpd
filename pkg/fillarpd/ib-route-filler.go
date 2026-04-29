package fillarpd

import (
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/vishvananda/netlink"
)

type IBRouteFiller struct {
	// Source Interface, must be setdiscrod
	Interface *net.Interface
	// Initial Routes Map, this is assumed to reflect the actual kernel state
	Routes map[netip.Addr]bool
	// The source ip to add to routes
	SourceIP net.IP
	// race conditions
	mu sync.Mutex
}

func (router *IBRouteFiller) AddRoute(addr netip.Addr) error {
	// thread safety
	router.mu.Lock()
	defer router.mu.Unlock()

	if router.Routes[addr] {
		return nil // already exists
	}
	route := &netlink.Route{
		LinkIndex: router.Interface.Index,
		Dst: &net.IPNet{
			IP:   net.IP(addr.AsSlice()),
			Mask: net.CIDRMask(32, 32),
		},
		Scope: netlink.SCOPE_LINK, // Link, maybe host might work?
		Src:   router.SourceIP,
	}

	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add netlink route: %w", err)
	}

	router.Routes[addr] = true
	return nil
}

func (router *IBRouteFiller) RemoveRoute(addr netip.Addr) error {
	// thread safety
	router.mu.Lock()
	defer router.mu.Unlock()

	route := &netlink.Route{
		LinkIndex: router.Interface.Index,
		Dst: &net.IPNet{
			IP:   net.IP(addr.AsSlice()),
			Mask: net.CIDRMask(32, 32),
		},
	}

	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("could not remove route, does it exist? %w", err)
	}
	delete(router.Routes, addr)
	return nil
}

func (router *IBRouteFiller) PurgeUnused(knownIPs []netip.Addr) error {
	// this doesn't have to add as the sweep itself triggers a ton of arp
	// requests
	knownMap := make(map[netip.Addr]bool)
	for _, ip := range knownIPs {
		knownMap[ip] = true
	}

	for activeIP := range router.Routes {
		if !knownMap[activeIP] {
			if err := router.RemoveRoute(activeIP); err != nil {
				return err
			}
		}
	}
	return nil
}
