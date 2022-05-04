package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/gotd/td/telegram/query/messages"

	"github.com/rusq/wipemychat/internal/mtp"
)

func (app *App) initConfirm() {
	app.pages.AddPage(stConfirming, app.view.mbConfirm, false, false)
	app.view.mbConfirm.
		AddButtons([]string{btnYes, btnNo}).
		SetDoneFunc(app.handleConfirm).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyESC {
				app.cancel()
				return nil
			}
			return event
		})
}

func (app *App) handleConfirm(_ int, buttonLabel string) {
	var err error
	switch buttonLabel {
	case btnYes:
		if !app.event(evConfirmed) {
			return
		}
		err = app.handleDelete()
	case btnNo:
		app.cancel()
	default:
		err = nil
	}
	if err != nil {
		app.error(err)
	}
}

// handleDelete handles the deletion of the messages.  It gets the chat
// and messages to delete from the FSM Metadata.
func (app *App) handleDelete() error {
	defer app.event(evDeleted)
	chat, err := metadata[mtp.Entity](app.fsm, metaChat)
	if err != nil {
		return fmt.Errorf("chat missing: %s", err)
	}

	msgs, err := metadata[[]messages.Elem](app.fsm, metaMessages)
	if err != nil {
		return fmt.Errorf("messages missing: %s", err)
	}
	app.logf("Deleting %d messages from %s, please wait . . .", len(msgs), chat.GetTitle())
	n, err := app.tg.DeleteMessages(context.Background(), chat, msgs)
	if err != nil {
		return err
	}
	app.logf("%d messages deleted in %q", n, chat.GetTitle())

	return nil
}
