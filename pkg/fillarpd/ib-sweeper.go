package fillarpd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync"
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

	// Force bind a dialer on this interface to ignore routes
	d := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, sweeper.Interface.Name)
			})
		},
	}

	jobs := make(chan netip.Addr)
	var wg sync.WaitGroup
	// Setup the jobs
	for w := 0; w < sweeper.Threads; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// just fire something off to update the arptables,
			for targetIP := range jobs {
				conn, err := d.Dial("udp", net.JoinHostPort(targetIP.String(), "9"))
				if err == nil {
					conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
					conn.Write([]byte{0})
					conn.Close()
				}
			}
		}()
	}
	// Send the ips
	go func() {
		for ; sweeper.IPRange.Contains(addr.AsSlice()); addr = addr.Next() {
			jobs <- addr
		}
		close(jobs)
	}()

	wg.Wait()
	// we completely ignore the result of the last one because it did what we wanted which is to update the route table

	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

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
