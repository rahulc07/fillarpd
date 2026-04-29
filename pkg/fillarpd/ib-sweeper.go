package fillarpd

import (
	"bufio"
	"context"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

type IBSweeper struct {
	Threads int
}

func (sweeper *IBSweeper) FindIPs(ctx context.Context, prefix netip.Prefix) ([]netip.Addr, error) {
	var found []netip.Addr
	var mu sync.Mutex

	ips := make(chan netip.Addr)
	var wg sync.WaitGroup

	for i := 0; i < sweeper.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for addr := range ips {
				if sweeper.probe(ctx, addr) {
					mu.Lock()
					found = append(found, addr)
					mu.Unlock()
				}
			}
		}()
	}

	go func() {
	loop:
		for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
			if addr.IsUnspecified() || addr.As4()[3] == 0 || addr.As4()[3] == 255 {
				continue
			}

			select {
			case <-ctx.Done():
				break loop
			case ips <- addr:
			}
		}
		close(ips)
	}()

	wg.Wait()
	return found, nil
}
func (sweeper *IBSweeper) probe(ctx context.Context, addr netip.Addr) bool {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: addr.AsSlice(), Port: 7})
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	if _, err = conn.Write([]byte{0}); err != nil {
		return false
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(50 * time.Millisecond):
		}

		f, err := os.Open("/proc/net/arp")
		if err != nil {
			return false
		}

		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip header
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 3 {
				continue
			}
			ip, err := netip.ParseAddr(fields[0])
			if err != nil || ip != addr {
				continue
			}
			const COMPLETE_ARP = "0x2"
			const COMPLETE_PUBLISHED_ARP = "0x6"
			if fields[2] == COMPLETE_ARP || fields[2] == COMPLETE_PUBLISHED_ARP {
				f.Close()
				return true
			}
		}
		f.Close()
	}
	return false
}
