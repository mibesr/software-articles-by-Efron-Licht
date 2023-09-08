// dns is a simple command line tool to lookup the ip address of a host;
// it prints the first ipv4 and ipv6 addresses it finds, or "none" if none are found.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: dns <host>")
	}
	host := os.Args[1]
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatalf("error looking up %s: %v", host, err)
	}
	if len(ips) == 0 {
		log.Fatalf("no ips found for %s", host)
	}
	// print the first ipv4 we find
	for _, ip := range ips {
		if ip.To4() != nil {
			fmt.Printf("%s\n", ip)
		}
		goto IPV6
	}
	fmt.Printf("none\n")

IPV6: // print the first ipv6 we find
	for _, ip := range ips {
		if ip.To4() == nil {
			fmt.Printf("%s\n", ip)
			return
		}
	}
	fmt.Printf("none\n")
}
