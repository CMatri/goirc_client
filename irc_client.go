package main

import (
	"com/cmatri/irc_client/lib"
	"os"
	"strconv"
	"fmt"
)

func b() {}

func main() {

	args := os.Args[1:]

	if len(args) != 5 && len(args) != 4 {
		fmt.Println("Usage: irc_client [username] [nick] [pass] [address] [port (default = 6667)]")
	} else {
		var port int
		if len(args) == 4 { 
			port = 6667 
		} else { 
			p, err := strconv.Atoi(args[4])
			if err == nil { 
				port = p
			} else {
				fmt.Println("Invalid port number. Enter a port between 0 and 65535.")
				os.Exit(0)
			}
		}

		uiChannel := make(chan string)
		client := new(lib.IRCClient)
		client.Addr = args[3]
		client.Port = port
		client.User = args[0]
		client.Nick = args[1]
		client.Pass = args[2]
		client.UIchannel = uiChannel
	
		client.InitiateConnection()
		client.RegisterUser()
		go client.Receive()		
		client.BuildUI()
	}
}