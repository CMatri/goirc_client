package lib

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
	User string
	Nick string
	Pass string
	Addr string
	Port int
	Socket net.Conn
	IRCTui
}

type IRCTui struct {
	UIchannel chan string
	Sidebar *tui.Box
	History *tui.Box
}

func (c *IRCClient) Send(data string) {
	fmt.Fprintf(c.Socket, data)
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

		if strings.Contains(line, "PING") {
			fmt.Fprintf(c.Socket, "PONG :" + strings.Split(line, ":")[1] + "\r\n")
		} else if strings.Contains(line, "ERROR") {
			break
		} else {
			c.UIchannel <- line
			//fmt.Println(line)
		}
	}

	close(c.UIchannel)
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
	c.Send("PASS " + c.Pass + "\r\nUSER " + c.User + " * * :" + c.User + "\r\nNICK " + c.Nick + "\r\n")
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

func (t *IRCClient) BuildUI() {
	t.Sidebar = tui.NewVBox(tui.NewLabel("CHANNELS"))
	t.Sidebar.SetBorder(true)
	t.History = tui.NewVBox()
	historyScroll := tui.NewScrollArea(t.History)
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
	root := tui.NewHBox(t.Sidebar, chat)
	ui, _ := tui.New(root)
	ui.SetKeybinding("Esc", func() { 
		ui.Quit()
		close(t.UIchannel)
	})

	input.OnSubmit(func(e *tui.Entry) {
		t.Send(e.Text() + "\r\n")
		t.History.Append(tui.NewHBox(tui.NewLabel(e.Text())))
		input.SetText("")
	})
	
	go func() {
		if err := ui.Run(); err != nil {
			panic(err)
		}
	}()

	for {
		val, ok := <-t.UIchannel
		if !ok {
			break
		}
		ui.Update(func() {
			t.History.Append(tui.NewHBox(tui.NewLabel(val)))
		})
	}
}