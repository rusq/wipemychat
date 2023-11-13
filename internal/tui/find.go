package tui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) initFind(ctx context.Context) {
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
				app.findChat(ctx)
				input.SetText("")
				return nil
			case tcell.KeyESC:
				app.cancel(ctx)
				return nil
			}
			return event
		})
}

func (app *App) findChat(ctx context.Context) {
	val := app.view.fmSearch.GetFormItem(0).(*tview.InputField)

	text := val.GetText()
	if text == "" {
		app.logf("search input is empty")
		app.cancel(ctx)
		return
	}

	loc := app.view.lvChats.FindItems(text, text, false, true)
	if len(loc) == 0 {
		app.logf("search term not found: %q", text)
		app.cancel(ctx)
		return
	}

	app.view.lvChats.SetCurrentItem(loc[0])
	if !app.event(ctx, evLocate) {
		return
	}
}
