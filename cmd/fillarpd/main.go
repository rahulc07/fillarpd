package main

import (
	"context"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/supercompucsd/fillarpd/pkg/fillarpd"
)

type config struct {
	// Using 'required' ensures the app won't run without these
	Interface     string `arg:"-i,env:INTERFACE,required" help:"Network interface name (e.g., ibs1)"`
	SourceIP      string `arg:"-s,env:SOURCE_IP,required" help:"Source IP for route injection (e.g., 10.2.0.1)"`
	Network       string `arg:"-n,env:NETWORK,required"   help:"CIDR range to sweep (e.g., 10.2.0.0/24)"`
	SweepInterval int    `arg:"-t,env:SWEEP_INTERVAL,required" help:"Seconds between sweeps"`
	Threads       int    `arg:"-j,env:THREADS" default:"24" help:"Number of parallel sweep threads"`
}

func main() {
	var cfg config
	arg.MustParse(&cfg)

	_, network, err := net.ParseCIDR(cfg.Network)
	if err != nil {
		log.Fatalf("Invalid Network Range %s, %v", cfg.Network, err)
	}

	proxyInterface, err := net.InterfaceByName(cfg.Interface)
	if err != nil {
		log.Fatalf("interface %s is not compatible/not found %s", cfg.Interface, err.Error())
	}
	// Init router
	router := &fillarpd.IBRouteFiller{Interface: proxyInterface,
		SourceIP: net.ParseIP(cfg.SourceIP), Routes: make(map[netip.Addr]bool)}

	// Init scanner
	scanner := &fillarpd.IBArpSnooper{Interface: proxyInterface}

	// Init Sweeper
	sweeper := &fillarpd.IBSweeper{
		Interface: proxyInterface,
		IPRange:   network,
		Threads:   24,
	}

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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in sweep goroutine: %v", r)
			}
		}()
		ticker := time.NewTicker(time.Duration(cfg.SweepInterval) * time.Second)
		defer ticker.Stop()
		// run a sweep at start to populate the route table
		log.Println("Initial sweep starting...")
		if ips, err := sweeper.FindIPs(ctx); err == nil {
			for _, ip := range ips {
				packetChan <- ip
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("Sweeping...")
				ips, err := sweeper.FindIPs(ctx)
				if err != nil {
					log.Printf("Sweep error: %v", err)
					continue
				}
				// Technically this isn't needed but just incase something was missed

				for _, ip := range ips {
					if err := router.AddRoute(ip); err != nil {
						log.Printf("Could not add routes? %s\n", err.Error())
					}
				}
				if err := router.PurgeUnused(ips); err != nil {
					log.Fatalf("Could not purge routes? %s\n", err.Error())
				}
			}
		}
	}()

	log.Println("Daemon started. Waiting for ARP traffic...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully (waiting up to 1s)...")
			log.Println("Removing all ARP discovered routes")
			router.PurgeAll()
			return
		case ip := <-packetChan:
			if !router.Routes[ip] {
				log.Printf("Detected IP: %s. Updating routes...", ip)
			}
			if err := router.AddRoute(ip); err != nil {
				log.Printf("Could not add routes? %s\n", err.Error())
			}
		}
	}
}
