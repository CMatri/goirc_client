package main

import (
	"fmt"
	"os"
	"strconv"
	"net"
	"bufio"
	"strings"
	"io"
	"github.com/marcusolsson/tui-go"
)

type IRCClient struct {
	user string
	nick string
	pass string
	addr string
	port int
	socket net.Conn
	IRCTui
}

type IRCTui struct {
	uiChannel chan string
	sidebar *tui.Box
	history *tui.Box
}

func (c *IRCClient) send(data string) {
	fmt.Fprintf(c.socket, data)
}

func (c *IRCClient) receive() {
	for {
		reader := bufio.NewReader(c.socket) // Probably not the best, should probably move into struct. I feel like there's a more elegant solution though.
		ba, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			fmt.Println("Server closed connection.")
			c.socket.Close()
			break
		}

		line := string(ba)

		if strings.Contains(line, "PING") {
			fmt.Fprintf(c.socket, "PONG :" + strings.Split(line, ":")[1] + "\r\n")
		} else {
			c.uiChannel <- line
			//fmt.Println(line)
		}
	}

	close(c.uiChannel)
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

func (c *IRCClient) register_user() {
	c.send("PASS " + c.pass + "\r\nUSER " + c.user + " * * :" + c.user + "\r\nNICK " + c.nick + "\r\n")
}

func (c *IRCClient) initiate_connection() {
	full_addr := c.addr + ":" + strconv.Itoa(c.port)
	fmt.Print("Initiating connection with " + full_addr + "...")
	conn, err := net.Dial("tcp", full_addr)
	if(err != nil || conn == nil) {
		fmt.Println("Timed out")
		os.Exit(0)
	}
	c.socket = conn
	fmt.Println("Success")
}

func (t *IRCClient) build_ui() {
	t.sidebar = tui.NewVBox(tui.NewLabel("CHANNELS"))
	t.sidebar.SetBorder(true)
	t.history = tui.NewVBox()
	historyScroll := tui.NewScrollArea(t.history)
	historyScroll.SetAutoscrollToBottom(true)
	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)
	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)
	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)
	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)
	root := tui.NewHBox(t.sidebar, chat)
	ui, _ := tui.New(root)

	ui.SetKeybinding("Esc", func() { 
		ui.Quit()
		close(t.uiChannel)
	})

	input.OnSubmit(func(e *tui.Entry) {
		t.send(e.Text() + "\r\n")
		t.history.Append(tui.NewHBox(tui.NewLabel(e.Text())))
		input.SetText("")
	})

	go func() {
		if err := ui.Run(); err != nil {
			panic(err)
		}
	}()

	for {
		val, ok := <-t.uiChannel
		if !ok {
			break
		}
		ui.Update(func() {
			t.history.Append(tui.NewHBox(tui.NewLabel(val)))
		})
	}
}

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
		client := new(IRCClient)
		client.addr = args[3]
		client.port = port
		client.user = args[0]
		client.nick = args[1]
		client.pass = args[2]
		client.uiChannel = uiChannel
	
		client.initiate_connection()
		client.register_user()
		go client.receive()		
		client.build_ui()
	}
}