package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var verbose bool

type Packet struct {
	size int
	data [1500]byte
}

func ConnToChan(src io.Reader, ch chan Packet, done chan int) {
	defer func() {
		done <- 0
	}()
	defer close(ch)
	var buffer Packet
	bytes, err := src.Read(buffer.data[:])
	if err != nil {
		if IsEOF(err) {
			buffer.size = bytes
		} else {
			log.Printf("Read error: %s\n", err.Error())
			buffer.size = -1
		}
	} else {
		buffer.size = bytes
	}
	ch <- buffer
	for buffer.size > 0 {
		bytes, err := src.Read(buffer.data[:])
		if err != nil {
			if IsEOF(err) {
				buffer.size = bytes
			} else {
				log.Printf("Read error: %s\n", err.Error())
				buffer.size = -1
			}
		} else {
			buffer.size = bytes
		}
		ch <- buffer
	}
}

func ChanToConn(ch chan Packet, tgt io.Writer, done chan int) {
	defer func() {
		done <- 0
	}()
	defer close(ch)
	var buffer Packet
	buffer, ok := <-ch
	for ok && buffer.size > 0 {
		_, err := tgt.Write(buffer.data[0:buffer.size])
		if err != nil {
			log.Printf("Write error: %s\n", err.Error())
			return
		}
		buffer, ok = <-ch
	}
}
func doConn(con net.Conn) {
	ch_stdout := make(chan Packet)
	ch_remote := make(chan Packet)
	complete := make(chan int)

	go ConnToChan(con, ch_stdout, complete)
	go ChanToConn(ch_stdout, os.Stdout, complete)

	go ConnToChan(os.Stdin, ch_remote, complete)
	go ChanToConn(ch_remote, con, complete)

	<-complete
}

func IsEOF(err error) bool {
	if err == nil {
		return false
	} else if err == io.EOF {
		return true
	} else if oerr, ok := err.(*net.OpError); ok {
		if oerr.Err.Error() == "use of closed network connection" {
			return true
		}
	} else {
		if err.Error() == "use of closed network connection" {
			return true
		}
	}
	return false
}

func proxyConnect(proxy string, protocol string, host string) net.Conn {
	var con net.Conn
	if protocol == "connect" {
		con = connect(proxy)
		conString := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: go-nc\r\nProxy-Connection: Keep-Alive\r\n\r\n", host, host)
		if verbose {
			log.Print("Sending request to proxy:\n", conString)
		}
		var buffer Packet
		buffer.size = len(conString)
		copy(buffer.data[:], []byte(conString))
		_, err := io.Writer(con).Write(buffer.data[0:buffer.size])
		if err != nil {
			log.Fatalln("Error writing CONNECT message to proxy", err)
		}
		bytes, err := io.Reader(con).Read(buffer.data[:])
		if err != nil {
			log.Fatalln("Error reading CONNECT result from proxy", err)
		}
		if verbose {
			log.Print("Received response from proxy:\n", string(buffer.data[0:bytes]))
		}
	} else {
		log.Fatalln("Bad proxy protocol.  We should not be here.")
	}
	return con
}

func connect(host string) net.Conn {
	con, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(err)
	}
	if verbose {
		log.Println("Connected to", host)
	}
	return con
}

func main() {
	var listen string
	var destinationPort string
	var isListen bool
	var host string
	var proxyProtocol string
	var proxyString string
	flag.StringVar(&listen, "l", "", "listen to port number n, :n or b:n, where n is port number, b is binding inteface, defaults to 0")
	flag.StringVar(&proxyProtocol, "X", "", "connect to HTTP proxy")
	flag.StringVar(&proxyString, "x", "", "host[:port] of proxy to connect to")
	flag.BoolVar(&verbose, "v", false, "Noisy")
	flag.Parse()
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		fmt.Println("\nSimplified nc [-v] [-l port] or [hostname port]")
		fmt.Println("")
		flag.Usage()
		fmt.Println(`
Examples:
	nc -l 9899

	means listen on port 9899

	nc localhost 6767

	means connect to port 6767`)

		os.Exit(1)
	}
	if listen != "" {
		isListen = true
		ret := strings.SplitN(listen, ":", 2)
		port := ret[len(ret)-1]
		if v, err := strconv.Atoi(port); err != nil || (v > 65535 || v < 1) {
			log.Println("Listen port must be an integer between from 1-65536")
			os.Exit(1)
		}
		if len(ret) == 1 {
			listen = ":" + port
		} else if ret[0] == "" {
			listen = ":" + port
		}

		if flag.NArg() != 0 {
			log.Printf("Listen mode on: Arguments %v are extra\n", flag.Args())
			os.Exit(1)
		}
		if verbose {
			log.Println("Listen: ", listen)
		}
	} else {
		if flag.NArg() != 2 {
			log.Println("Listen mode off: [hostname] [port] are mandatory arguments")
			os.Exit(1)
		}
		if v, err := strconv.Atoi(flag.Arg(1)); err != nil || (v > 65535 || v < 1) {
			log.Println("Destination port must be an integer between from 1-65536")
			os.Exit(1)
		}
		host = flag.Arg(0)
		destinationPort = ":" + flag.Arg(1)

		if proxyString != "" {
			ret := strings.SplitN(proxyString, ":", 2)
			if ret[0] == "" {
				log.Println("Proxy host must not be empty")
				os.Exit(1)
			}
			port := ret[len(ret)-1]
			if v, err := strconv.Atoi(port); err != nil || (v > 65535 || v < 1) {
				log.Println("Proxy port must be an integer between from 1-65536")
				os.Exit(1)
			}
		}

		if verbose {
			log.Println("Hostname:", host)
			log.Println("Port:", destinationPort)
			log.Println("Proxy protocol:", proxyProtocol)
			log.Println("Proxy:", proxyString)
		}
	}

	if isListen {
		listener, err := net.Listen("tcp", listen)
		if err != nil {
			log.Fatalln(err)
		}
		if verbose {
			log.Println("Listening on", listen)
		}
		con, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		if verbose {
			log.Println("Connect from", con.RemoteAddr())
		}
		doConn(con)

	} else if host != "" {
		var con net.Conn
		if proxyString != "" {
			con = proxyConnect(proxyString, proxyProtocol, host+destinationPort)
		} else {
			con = connect(host + destinationPort)
		}
		doConn(con)
	} else {
		flag.Usage()
	}
}
