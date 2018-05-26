// This program implements a client program that computes a Dirtree on some machine
// and sends the operations over the network to the main Spacehoarder UI which renders them.
package main

import (
	"flag"
	"fmt"
	"github.com/jeffwilliams/spacehoarder/dirtree"
	"net"
)

// Run as a server for debugging.
var optServer = flag.Bool("server", false, "For debugging. Run as a server and print out data sent by client.")
var optHelp = flag.Bool("h", false, "Show help")

func doclient(basedir string, addr string) {
	ops, prog := dirtree.Build(basedir, dirtree.DefaultBuildOpts)

	opConn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Connecting failed:", err)
		return
	}

	defer opConn.Close()

	progConn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Connecting failed:", err)
		return
	}

	defer progConn.Close()

	dirtree.Encode(opConn, progConn, ops, prog)
}

func doserver() {
	listener, err := net.Listen("tcp", ":7570")
	if err != nil {
		fmt.Println("Listening failed:", err)
		return
	}

	opConn, err := listener.Accept()
	if err != nil {
		fmt.Println("Accept failed: ", err)
		return
	}
	defer opConn.Close()

	progConn, err := listener.Accept()
	if err != nil {
		fmt.Println("Accept failed: ", err)
		return
	}
	defer progConn.Close()

	listener.Close()

	ops := make(chan dirtree.OpData)
	prog := make(chan string)

	dirtree.Decode(opConn, progConn, ops, prog)

	for {
		select {
		case op, ok := <-ops:
			if !ok {
				ops = nil
				break
			}
			fmt.Println("Op: ", op)
		case f, ok := <-prog:
			if !ok {
				prog = nil
				break
			}
			fmt.Println(f)
		}

		if ops == nil && prog == nil {
			break
		}
	}
}

func help() {
	fmt.Println("Usage: sphclient <directory> <server addr>")
	fmt.Println("")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if flag.NArg() < 2 || *optHelp {
		help()
		return
	}

	if !*optServer {
		doclient(flag.Arg(0), flag.Arg(1))
	} else {
		doserver()
	}
}
