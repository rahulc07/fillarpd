package fillarpd

import (
	"bytes"
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type IBArpSnooper struct {
	Interface *net.Interface
}

func (scanner *IBArpSnooper) Scan(ctx context.Context) (chan netip.Addr, error) {
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

	out := make(chan netip.Addr, 256)
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
					// we've stopped getting packets
					return
				}

				arpLayer := packet.Layer(layers.LayerTypeARP)
				if arpLayer == nil {
					continue
				}

				arp := arpLayer.(*layers.ARP)
				if arp.Operation == layers.ARPReply &&
					bytes.Equal(arp.SourceHwAddress, scanner.Interface.HardwareAddr) {
					// despite what the documentation comments say at some point Go Net was updated
					// to support arbitrarty length hw addresses (like IB)
					continue
				}
				if ip, ok := netip.AddrFromSlice(arp.SourceProtAddress); ok {
					out <- ip.Unmap()
				}
			}
		}
	}()
	return out, nil
}
