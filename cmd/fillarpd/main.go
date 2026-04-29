package fillarpd

import (
	"context"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"

	"github.com/supercompucsd/fillarpd/pkg/fillarpd"
)

func main() {
	interfaceName := "ibs1"
	sourceIP := "10.2.0.1"
	//scanIP := "10.2.0.1/24"

	proxyInterface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		log.Fatalf("interface %s is not compatible/not found %s", interfaceName, err.Error())
	}
	// Init router
	router := &fillarpd.IBRouteFiller{Interface: proxyInterface,
		SourceIP: net.ParseIP(sourceIP), Routes: make(map[netip.Addr]bool)}

	// Init scanner
	scanner := &fillarpd.IBArpScanner{Interface: proxyInterface}

	// Init Sweeper

	/////
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	packetChan, err := scanner.Scan(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Daemon started. Waiting for ARP traffic...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully (waiting up to 1s)...")
			return
		case ip := <-packetChan:
			log.Printf("Detected IP: %s. Updating routes...", ip)
			router.AddRoute(ip)
		}
	}
}
