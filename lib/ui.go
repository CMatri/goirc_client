package lib

import (
	"github.com/marcusolsson/tui-go"
)

type IRCTui struct {
	UIChannel chan string
	ClientChannel chan string
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

func (t *IRCTui) BuildUI() {
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
		close(t.UIChannel)
		close(t.ClientChannel)
	})

	ui.SetKeybinding("Up", func() { 
		input.SetText(t.GetHistory(true))
	})

	ui.SetKeybinding("Down", func() {
		input.SetText(t.GetHistory(false))
	})

	input.OnSubmit(func(e *tui.Entry) {
		if len(e.Text()) > 0 {
			//t.Send(e.Text())
			t.ClientChannel <- e.Text()
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
		val, ok := <-t.UIChannel

		if !ok {
			break
		}
		ui.Update(func() {
			t.History.Append(tui.NewHBox(tui.NewLabel(val)))
		})
	}
}