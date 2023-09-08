// writetcp connects to a TCP server at at localhost with the specified port (8080 by default) and forwards stdin to the server,
// line-by-line, until EOF is reached.
// received lines from the server are printed to stdout.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	const name = "writetcp"
	log.SetPrefix(name + "\t")

	// register the command-line flags: -p specifies the port to connect to
	port := flag.Int("p", 8080, "port to connect to")
	host := flag.String("h", "", "host to connect to; leave empty for localhost")
	flag.Parse()

	var ip net.IP // find the ip address of the host we want to connect to
	if *host != "" {
		var err error
		ip, err = findIP(*host)
		if err != nil {
			log.Fatalf("findIP(%s): %v", *host, err)
		}
		log.Printf("found ip address for %s: %s", *host, ip)
	}

	// if IP is nil, we'll connect to localhost.
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ip, Port: *port})
	if err != nil {
		log.Fatalf("error connecting to localhost:%d: %v", *port, err)
	}
	if ip != nil {
		log.Printf("connected to %s:%d (%s:%d): forwarding stdin", *host, *port, ip, *port)
	} else {
		log.Printf("connected to localhost:%d: forwarding stdin", *port)
	}
	defer conn.Close()
	go func() { // spawn a goroutine to read incoming lines from the server and print them to stdout.
		// TCP is full-duplex, so we can read and write at the same time; we just need to spawn a goroutine to do the reading.

		for connScanner := bufio.NewScanner(conn); connScanner.Scan(); {

			fmt.Printf("%s\n", connScanner.Text()) // note: printf doesn't add a newline, so we need to add it ourselves

			if err := connScanner.Err(); err != nil {
				log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
			}
			if connScanner.Err() != nil {
				log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
			}
		}
	}()

	// read incoming lines from stdin and forward them to the server.
	for stdinScanner := bufio.NewScanner(os.Stdin); stdinScanner.Scan(); { // find the next newline in stdin
		log.Printf("sent: %s\n", stdinScanner.Text())
		if _, err := conn.Write(stdinScanner.Bytes()); err != nil { // scanner.Bytes() returns a slice of bytes up to but not including the next newline
			log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
		}
		if _, err := conn.Write([]byte("\n")); err != nil { // we need to add the newline back in
			log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
		}
		if stdinScanner.Err() != nil {
			log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
		}
	}
}

func findIP(host string) (ip net.IP, err error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("no ips found for known host")
	}
	// look for the first ipv4 address
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip, nil
		}
	}
	// none of them were ipv4, so return the first ipv6 address
	return ips[0], nil
}
