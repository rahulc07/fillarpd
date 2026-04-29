package fillarpd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"
)

type IBSweeper struct {
	Interface *net.Interface
	IPRange   *net.IPNet
	Threads   int
}

func (sweeper *IBSweeper) FindIPs(ctx context.Context) ([]netip.Addr, error) {
	addr, ok := netip.AddrFromSlice(sweeper.IPRange.IP)
	if !ok {
		return nil, fmt.Errorf("invalid base IP")
	}
	// maybe make this multi-threaded

	// Force bind a dialer on this interface to ignore routes
	d := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, sweeper.Interface.Name)
			})
		},
	}
	// just fire something off to update the arptables,
	for ; sweeper.IPRange.Contains(addr.AsSlice()); addr = addr.Next() {
		conn, err := d.Dial("udp", fmt.Sprintf("%s:9", addr.String()))
		if err == nil {
			conn.Write([]byte{0})
			conn.Close()
			conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
		}
	}
	// we completely ignore the result of the last one because it did what we wanted which is to update the route table
	time.Sleep(10 * time.Second)
	knownHosts, err := netlink.NeighList(sweeper.Interface.Index, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve route table %w", err)
	}
	// filter the knownhosts down to only the one's reachable and in range.
	var reachableHosts []netip.Addr
	for _, neighbor := range knownHosts {
		if (neighbor.State & (netlink.NUD_REACHABLE)) != 0 {
			log.Printf("%s is REACHABLE", neighbor.IP.String())
			ip, ok := netip.AddrFromSlice(neighbor.IP)
			if !ok {
				return nil, fmt.Errorf("route table ip invalid %w", err)
			}
			reachableHosts = append(reachableHosts, ip)
		}
	}
	return reachableHosts, nil
}
