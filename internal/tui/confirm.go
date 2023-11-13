package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/gotd/td/telegram/query/messages"

	mtp "github.com/rusq/mtpwrap"
)

func (app *App) initConfirm(ctx context.Context) {
	app.pages.AddPage(stConfirming, app.view.mbConfirm, false, false)
	app.view.mbConfirm.
		AddButtons([]string{btnYes, btnNo}).
		SetDoneFunc(app.handleConfirm).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyESC {
				app.cancel(ctx)
				return nil
			}
			return event
		})
}

func (app *App) handleConfirm(_ int, buttonLabel string) {
	ctx := context.TODO()
	var err error
	switch buttonLabel {
	case btnYes:
		if !app.event(ctx, evConfirmed) {
			return
		}
		err = app.handleDelete(ctx)
	case btnNo:
		app.cancel(ctx)
	default:
		err = nil
	}
	if err != nil {
		app.error(err)
	}
}

// handleDelete handles the deletion of the messages.  It gets the chat
// and messages to delete from the FSM Metadata.
func (app *App) handleDelete(ctx context.Context) error {
	defer app.event(ctx, evDeleted)
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
