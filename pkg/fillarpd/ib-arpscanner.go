package fillarpd

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type IBArpScanner struct {
	Interface *net.Interface
}

func (scanner *IBArpScanner) Scan(ctx context.Context) (chan netip.Addr, error) {
	// snoop (promiscious mode)
	// block forever should be fine here
	handle, err := pcap.OpenLive(scanner.Interface.Name, 65536, true, time.Second)
	if err != nil {
		return nil, err
	}
	// Don't spam this daemon with every broadcast packet that crosses the
	// network.
	if err := handle.SetBPFFilter("arp"); err != nil {
		return nil, err
	}

	out := make(chan netip.Addr)
	source := gopacket.NewPacketSource(handle, handle.LinkType())

	go func() {
		defer handle.Close()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case packet := <-source.Packets():
				if packet == nil {
					// this packet is useless
					return
				}

				arpLayer := packet.Layer(layers.LayerTypeARP)
				if arpLayer == nil {
					continue
				}

				arp := arpLayer.(*layers.ARP)

				if ip, ok := netip.AddrFromSlice(arp.SourceProtAddress); ok {
					out <- ip.Unmap()
				}
			}
		}
	}()
	return out, nil
}
