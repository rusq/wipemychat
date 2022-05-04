package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) initFind() {
	app.pages.AddPage(stSearching, modal(app.view.fmSearch, 60, 5), true, false)
	input := tview.NewInputField().SetLabel("Search")
	app.view.fmSearch.
		AddFormItem(input).
		SetBorder(true).
		SetTitle("[ Find Chat ]").
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCR:
				app.findChat()
				input.SetText("")
				return nil
			case tcell.KeyESC:
				app.cancel()
				return nil
			}
			return event
		})
}

func (app *App) findChat() {
	val := app.view.fmSearch.GetFormItem(0).(*tview.InputField)

	text := val.GetText()
	if text == "" {
		app.logf("search input is empty")
		app.cancel()
		return
	}

	loc := app.view.lvChats.FindItems(text, text, false, true)
	if len(loc) == 0 {
		app.logf("search term not found: %q", text)
		app.cancel()
		return
	}

	app.view.lvChats.SetCurrentItem(loc[0])
	if !app.event(evLocate) {
		return
	}
}
