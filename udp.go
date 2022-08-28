package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	gv "github.com/asaskevich/govalidator"
)

var DOMAIN string = ""

const COUNT = 1000

func reply(pc net.PacketConn, addr *net.UDPAddr, hostname string, s *chan state) {
	for {
		buf := make([]byte, 1024)
		n, ad, err := pc.ReadFrom(buf)
		if err != nil {
			*s <- state{code: 1, err: err}
			return
		}

		if string(buf[:n]) != hostname && strings.Join(strings.Split(string(buf[:n]), ".")[1:], ".") == DOMAIN {
			fmt.Printf("%s: %s\n", ad, buf[:n])
			_, err = pc.WriteTo([]byte(hostname), addr)
			if err != nil {
				*s <- state{code: 1, err: err}
				return
			}
		}

		time.Sleep(1 * time.Millisecond)
	}
	*s <- state{code: 0, err: nil}
	return
}

func announce(pc net.PacketConn, addr *net.UDPAddr, hostname string, s *chan state) {
	_, err := pc.WriteTo([]byte(hostname), addr)
	if err != nil {
		*s <- state{code: 1, err: err}
		return
	}

	for i := 0; i < COUNT; i++ {
		buf := make([]byte, 1024)
		pc.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, ad, err := pc.ReadFrom(buf)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			break
		}

		if string(buf[:n]) != hostname && strings.Join(strings.Split(string(buf[:n]), ".")[1:], ".") == DOMAIN {
			fmt.Printf("%s: %s\n", ad, buf[:n])
			*s <- state{code: 0, err: nil}
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	*s <- state{code: 1, err: fmt.Errorf("Nothing listening here")}
}

func adddistinct(addresses []string, address string) []string {
	var found bool = false
	for _, ip := range addresses {
		if ip == address {
			found = true
			break
		}
	}
	if !found {
		addresses = append(addresses, address)
	}
	return addresses
}

func lookupBroadcast(hostname string) []string {
	var addresses []string = make([]string, 0)

	ips, err := net.LookupIP(hostname)
	if err == nil {
		for _, addr := range ips {
			ip := make(net.IP, len(addr.To4()))
			binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(addr.To4())|^binary.BigEndian.Uint32(net.IP(addr.DefaultMask()).To4()))
			addresses = adddistinct(addresses, ip.String())
		}
	}

	return addresses
}

type state struct {
	code int
	err  error
}

func main() {
	var (
		err error
		ret int        = 0
		s   chan state = make(chan state)
	)

	if len(os.Args) >= 2 {
		f := flag.NewFlagSet("cmd", flag.ContinueOnError)
		f.StringVar(&DOMAIN, "d", "", "The domain to listen on")
		f.Parse(os.Args[2:])
	}

	h, _ := os.Hostname()
	var hostname string = string(h)
	if len(strings.Split(hostname, ".")) < 2 || !gv.IsDNSName(hostname) {
		if DOMAIN == "" {
			fmt.Println("Need domain name")
			os.Exit(1)
		}
		hostname = strings.Join([]string{hostname, DOMAIN}, ".")
	}
	fmt.Printf("Found hostname %s\n", hostname)

	pc, err := net.ListenPacket("udp4", ":8829")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	defer pc.Close()

	for _, address := range lookupBroadcast(hostname) {
		fmt.Printf("Resolving on %s\n", address)
		addr, err := net.ResolveUDPAddr("udp4", address+":8829")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(2)
		}

		switch os.Args[1] {
		case "send":
			go announce(pc, addr, hostname, &s)
		case "reply":
			go reply(pc, addr, hostname, &s)
		}
	}
	select {
	case x := <-s:
		ret = x.code
		err = x.err
		break
	}

	if err != nil {
		fmt.Println(err.Error())
	}
	os.Exit(ret)
}
