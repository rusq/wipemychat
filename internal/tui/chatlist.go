package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	mtp "github.com/rusq/mtpwrap"
)

const infoText = "Press [Ctrl+Q] or [F10] to quit, [Ctrl+F] or [/] to search chats"

func (app *App) initMain() {
	app.view.lvChats.
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.Color190).
		SetSelectedTextColor(tcell.ColorBlack).
		SetMainTextColor(tcell.Color190).
		ShowSecondaryText(true).
		SetBorder(true).
		SetInputCapture(app.chatInputCapture).
		SetTitle("[ Chats ]")

	app.view.tvLog.
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() { app.tva.Draw() }).
		SetBorder(true).
		SetTitle("[ Information ]")

	// main is the main screen, split in two parts.
	workspace := tview.NewFlex().
		AddItem(app.view.lvChats, 0, 25, true).
		AddItem(app.view.tvLog, 0, 75, false)

	// The bottom row is the help message
	info := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorRed).
		SetText(infoText)

	mainScreen := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(workspace, 0, 1, true).
		AddItem(info, 1, 1, false)

	app.pages.AddPage(stSelecting, mainScreen, true, true)

}

func (app *App) populateChatList(ctx context.Context, chats []mtp.Entity) {
	for _, chat := range chats {
		app.view.lvChats.AddItem(
			chat.GetTitle(),
			fmt.Sprintf("  %s (%d)", chat.TypeInfo().Name, chat.GetID()),
			0,
			func() { app.handleChats(ctx, chats) },
		)
	}
}

func (app *App) handleChats(ctx context.Context, chats []mtp.Entity) {
	if !app.event(evSelected) {
		return
	}

	selected := chats[app.view.lvChats.GetCurrentItem()]
	// async fetch is needed so that the tvLog will keep updating.
	go app.runDelete(selected)
}

func (app *App) runDelete(selected mtp.Entity) {
	// disable input on lvChats
	app.view.lvChats.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return nil })
	defer func() {
		// once finished collecting information, reenable the input.
		app.view.lvChats.SetInputCapture(app.chatInputCapture)
	}()
	app.view.tvLog.Clear()

	app.logf("Scanning chat: %s, please wait...", selected.GetTitle())
	total := 0
	msgs, err := app.tg.SearchAllMyMessages(context.Background(), selected, func(n int) {
		total += n
		if total > 0 && total%100 == 0 {
			app.printf("...%d", total)
		}
	})
	if total > 0 {
		app.printf("...%d\n", total)
	}
	if err != nil {
		app.error(err)
		app.cancel()
		return
	}
	app.logf("Scan complete, found %d messages", len(msgs))

	if len(msgs) == 0 {
		// show nothing to do message.
		if !app.event(evNothingToDo) {
			app.cancel()
		}
		return
	}

	app.fsm.SetMetadata(metaChat, selected)
	app.fsm.SetMetadata(metaMessages, msgs)
	app.view.mbConfirm.SetText(fmt.Sprintf("Found %d messages in %q.  Delete?", len(msgs), selected.GetTitle()))

	if !app.event(evFetched) {
		app.cancel()
		return
	}
}

func (app *App) chatInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlF:
		if !app.event(evSearch) {
			return event
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case '/':
			if !app.event(evSearch) {
				return event
			}
		}
	}
	return event
}
