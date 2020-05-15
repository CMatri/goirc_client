package main

import (
	"fmt"
	"os"
	"strconv"
	"net"
	"time"
	"bufio"
	"strings"
	"io"
)

type IRCClient struct {
	user string
	nick string
	pass string
	addr string
	port int
	socket net.Conn
}

func (c *IRCClient) register_user() {
	fmt.Fprintf(c.socket, "PASS " + c.pass + "\r\nUSER " + c.user + " * * :" + c.user + "\r\nNICK " + c.nick + "\r\n")
}

func (c *IRCClient) receive_line() {
	for {
		reader := bufio.NewReader(c.socket) // Probably not the best, should probably move into struct. I feel like there's a more elegant solution though.
		ba, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			fmt.Println("Server closed connection.")
			c.socket.Close()
			os.Exit(0)	
		}

		line := string(ba)

		if strings.Contains(line, "PING") {
			fmt.Fprintf(c.socket, "PONG :" + strings.Split(line, ":")[1] + "\r\n")
		} else {
			println(line)
		}
	}
}

func (c *IRCClient) dump_buf() {
	for {
		data := make([]byte, 2048)
		length, err := c.socket.Read(data)
		if err != nil {
			c.socket.Close()
			fmt.Println("Server closed connection.")
			break
		}
		if length > 0 {
			fmt.Print(string(data[0:length]))
		}
	}
}

func initiate_connection(addr string, port int, user string, nick string, pass string) {
	full_addr := addr + ":" + strconv.Itoa(port)
	fmt.Print("Initiating connection with " + full_addr + "...")
	conn, err := net.Dial("tcp", full_addr)
	
	if(err != nil) {
		fmt.Println("Couldn't initiate connection with " + full_addr)
		return
	}
	
	fmt.Println("Success")
	client := &IRCClient { addr: addr, port: port, socket: conn, user: user, nick: nick, pass: pass, }
	client.register_user()
	go client.receive_line()
	
	time.Sleep(20 * time.Second)
}

func main() {
	args := os.Args[1:]
	
	if len(args) != 5 && len(args) != 4 {
		fmt.Println("Usage: irc_client [username] [nick] [pass] [address] [port (default = 6667)]")
	} else if len(args) == 4 {
		initiate_connection(args[3], 6667, args[0], args[1], args[2]);
	} else {
		port, err := strconv.Atoi(args[4])
		if err == nil { 
			initiate_connection(args[3], port, args[0], args[1], args[2])
		} else {
			fmt.Println("Invalid port number. Enter a port between 0 and 65535.")
		}
	}
}