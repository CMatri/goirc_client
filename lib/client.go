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
	Server string
	Port int
	CurChannel string
	Socket net.Conn
	IRCTui
}

type IRCTui struct {
	UIchannel chan string
	Sidebar *tui.Box
	History *tui.Box
	Entries []string
	EntryIdx int
}

func (t *IRCTui) GetHistory(up bool) string {
	if up && t.EntryIdx > 0 {
		t.EntryIdx -= 1
		return t.Entries[t.EntryIdx]
	} else if !up && t.EntryIdx < len(t.Entries) - 1 {
		t.EntryIdx += 1
		return t.Entries[t.EntryIdx]
	}

	if up {
		return t.Entries[t.EntryIdx]
	} else {
		return ""
	}
}

func (t *IRCTui) PushHistory(line string) {
	t.Entries = append(t.Entries, line)
	t.EntryIdx = len(t.Entries)
}

func (c *IRCClient) Send(data string) {
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
		case "msg":
			msg := data[3 + len(cmd) + len(c.Nick):]
			c.UIchannel <- "You whisper to " + spl[1] + ": " + msg
			toSend = "PRIVMSG " + spl[1] + " :" + msg
		case "quit":
			toSend = "QUIT"
		default:
			c.UIchannel <- "Unknown command \"" + cmd + "\"."
		}
	} else {
		if c.CurChannel == "" {
			c.UIchannel <- "You haven't joined any channels yet. Use /join #example."
		} else {
			c.UIchannel <- c.Nick + ": " + data
			toSend = "PRIVMSG " + c.CurChannel + " :" + data
		}
	}

	if toSend != "" {
		fmt.Fprintf(c.Socket, toSend + "\r\n")
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
			c.UIchannel <- "Put on server " + c.Server
		}

		if strings.Contains(line, "ERROR") {
			break
		} else {
			c.HandleResponse(line)
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

	ui.SetKeybinding("Up", func() { 
		input.SetText(t.GetHistory(true))
	})

	ui.SetKeybinding("Down", func() {
		input.SetText(t.GetHistory(false))
	})

	input.OnSubmit(func(e *tui.Entry) {
		if len(e.Text()) > 0 {
			t.Send(e.Text())
			t.PushHistory(e.Text())
			input.SetText("") 
		}
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
					c.UIchannel <- postfix
				case 322: // list (not working)
					//for _, el := range strings.Split(postfix, "\n") { 
					//	c.UIchannel <- el + "\n\n"
					//}
				case 479: // join
					c.UIchannel <- "Illegal channel name."
				}
			} else {
				fromNick := strings.Split(prefix, "!")[0]
				switch key {
				case "NOTICE":
					c.UIchannel <- postfix
				case "JOIN":
					c.UIchannel <- fromNick + " joined " + postfix
				case "NICK":
					c.UIchannel <- fromNick + " is now " + postfix
				case "PRIVMSG":
					to := strings.Split(strings.Split(prefix, "PRIVMSG")[1], " ")[1]
					if to == c.Nick {
						c.UIchannel <- fromNick + " whispers to you: " + postfix
					} else {
						c.UIchannel <- to + " (" + fromNick + "): " + postfix
					}
				default:
					c.UIchannel <- data 
				}
			}
		}
	}
}