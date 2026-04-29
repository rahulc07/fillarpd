package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/supercompucsd/fillarpd/pkg/fillarpd"
)

func main() {
	inter, _ := net.InterfaceByName("ibs1")
	_, network, err := net.ParseCIDR("10.2.0.0/24")
	sweeper := fillarpd.IBSweeper{Interface: inter, IPRange: network, Threads: 24}
	ctx, _ := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	fmt.Printf("Finding Ips\n")
	_, err = sweeper.FindIPs(ctx)
	if err != nil {
		log.Fatalf("error %s", err.Error())
	}
}
