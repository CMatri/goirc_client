package lib

import (
	"fmt"
	"os"
	"strconv"
	"net"
	"bufio"
	"strings"
	"io"
)

type IRCClient struct {
	User string
	Nick string
	Pass string
	Addr string
	Server string
	Port int
	CurChannel string
	Socket net.Conn
	IRCTui
}

func (c *IRCClient) Send() {
	for {
		data, ok := <-c.ClientChannel

		if !ok {
			break
		}

		var toSend string
		if data[0] == '/' {
			spl := strings.Split(data[1:], " ")
			cmd := strings.ToLower(spl[0])
			switch cmd {
			//case "list": TODO: fix this
			//	toSend = "LIST"
			case "join":
				c.CurChannel = spl[1]
				toSend = "JOIN " + spl[1]
			case "names":
				if c.CurChannel != "" {
					toSend = "NAMES " + c.CurChannel
				}
			case "nick":
				toSend = "NICK" + spl[1] 
			case "msg":
				msg := data[len(spl[1]) + len(cmd) + len(c.Nick) - 1:]
				c.UIChannel	<- "You whisper to " + spl[1] + ": " + msg
				toSend = "PRIVMSG " + spl[1] + " :" + msg
			case "quit":
				toSend = "QUIT"
			default:
				c.UIChannel	<- "Unknown command \"" + cmd + "\"."
			}
		} else {
			if c.CurChannel == "" {
				c.UIChannel	<- "You haven't joined any channels yet. Use /join #example."
			} else {
				c.UIChannel	<- c.Nick + ": " + data
				toSend = "PRIVMSG " + c.CurChannel + " :" + data
			}
		}

		if toSend != "" {
			fmt.Fprintf(c.Socket, toSend + "\r\n")
		}
	}
}

func (c *IRCClient) Receive() {
	for {
		reader := bufio.NewReader(c.Socket) // Probably not the best, should probably move into struct. I feel like there's a more elegant solution though.
		ba, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			fmt.Println("Server closed connection.")
			c.Socket.Close()
			break
		}

		line := string(ba)
		if c.Server == "" && strings.Contains(line, "001") {
			c.Server = strings.Split(line, "001")[0][1:]
			c.UIChannel <- "Put on server " + c.Server
		}

		if strings.Contains(line, "ERROR") {
			break
		} else {
			c.HandleResponse(line)
		}
	}

	close(c.UIChannel)
	close(c.ClientChannel)
}

func (c *IRCClient) dump_buf() {
	for {
		data := make([]byte, 2048)
		length, err := c.Socket.Read(data)
		if err != nil {
			c.Socket.Close()
			fmt.Println("Server closed connection.")
			break
		}
		if length > 0 {
			fmt.Print(string(data[0:length]))
		}
	}
}

func (c *IRCClient) RegisterUser() {
	fmt.Fprintf(c.Socket, "PASS " + c.Pass + "\r\nUSER " + c.User + " * * :" + c.User + "\r\nNICK " + c.Nick + "\r\n")
}

func (c *IRCClient) InitiateConnection() {
	full_addr := c.Addr + ":" + strconv.Itoa(c.Port)
	fmt.Print("Initiating connection with " + full_addr + "...")
	conn, err := net.Dial("tcp", full_addr)
	if(err != nil || conn == nil) {
		fmt.Println("Timed out")
		os.Exit(0)
	}
	c.Socket = conn
	fmt.Println("Success")
}

func (c *IRCClient) HandleResponse(data string) {
	if strings.HasPrefix(data, "PING") {
		fmt.Fprintf(c.Socket, "PONG " + data[5:] + "\r\n")
		return
	}

	spl := strings.Split(data, ":")
	if len(spl) > 2 {
		prefix := spl[1]
		postfix := spl[2]
		preSpl := strings.Split(prefix, " ")

		if len(preSpl) > 1 {
			key := preSpl[1]

			if code, err := strconv.Atoi(key); err == nil {
				switch code {
				case 001: fallthrough // welcome
				case 002: fallthrough
				case 003: fallthrough
				case 004: fallthrough
				case 005: 
					c.UIChannel <- postfix
				case 322: // list (not working)
					//for _, el := range strings.Split(postfix, "\n") { 
					//	c.UIChannel <- el + "\n\n"
					//}
				case 353: // RPL_NAMREPLY
					c.UIChannel <- c.CurChannel + " users: " + postfix
				case 479: // join
					c.UIChannel <- "Illegal channel name."
				}
			} else {
				fromNick := strings.Split(prefix, "!")[0]
				switch key {
				case "QUIT":
					c.UIChannel <- fromNick + " has left the chat."
				case "NOTICE":
					c.UIChannel <- postfix
				case "JOIN":
					c.UIChannel <- fromNick + " joined " + postfix
				case "NICK":
					c.UIChannel <- fromNick + " is now " + postfix
				case "PRIVMSG":
					to := strings.Split(strings.Split(prefix, "PRIVMSG")[1], " ")[1]
					if to == c.Nick {
						c.UIChannel <- fromNick + " whispers to you: " + postfix
					} else {
						c.UIChannel <- to + " (" + fromNick + "): " + postfix
					}
				default:
					c.UIChannel <- data 
				}
			}
		}
	}
}