package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/looplab/fsm"
	"github.com/rivo/tview"
	"github.com/rusq/dlog"
	"github.com/rusq/osenv/v2"

	mtp "github.com/rusq/mtpwrap"
	"github.com/rusq/wipemychat/internal/waipu"
)

const (
	btnYes = "Yes"
	btnNo  = "No"
	btnOK  = "OK"
)

type App struct {
	tva *tview.Application
	tg  waipu.Telegramer
	log *dlog.Logger
	fsm *fsm.FSM

	pages *tview.Pages
	view  views
}

type views struct {
	main      *tview.Flex
	mbConfirm *tview.Modal
	mbNothing *tview.Modal
	fmSearch  *tview.Form

	lvChats *tview.List
	tvLog   *tview.TextView
}

func New(ctx context.Context, tg waipu.Telegramer) *App {
	app := &App{
		tva: tview.NewApplication(),
		tg:  tg,

		pages: tview.NewPages(),
		view: views{
			main:      tview.NewFlex(),
			mbConfirm: tview.NewModal(),
			mbNothing: tview.NewModal(),
			fmSearch:  tview.NewForm(),

			lvChats: tview.NewList(),
			tvLog:   tview.NewTextView(),
		},
	}

	app.initMain(ctx)
	app.initFind(ctx)
	app.initConfirm(ctx)
	app.initNothing(ctx)

	app.tva.SetInputCapture(app.handleKeystrokes)

	app.log = dlog.New(app.view.tvLog, "", dlog.Flags(), osenv.Value("DEBUG", "") != "")

	// init finite state machine
	app.fsm = initFSM(app)

	return app
}

func (app *App) Run(ctx context.Context, chats []mtp.Entity) error {
	app.populateChatList(ctx, chats)

	if err := app.tva.SetRoot(app.pages, true).EnableMouse(false).Run(); err != nil {
		return err
	}
	return nil
}

func (app *App) logf(format string, a ...any) {
	app.log.Printf(format, a...)
}

func (app *App) error(err error) {
	app.log.Printf("ERROR: %s", err)
}

func (app *App) handleKeystrokes(event *tcell.EventKey) *tcell.EventKey {
	if app.fsm.Current() == stDeleting {
		// we do not process keystrokes until deletion is finished.
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlQ, tcell.KeyF10:
		app.tva.Stop()
	default:
		return event
	}
	return nil
}

// cancel sends a evCancelled event.
func (app *App) cancel(ctx context.Context) {
	app.event(ctx, evCancelled)
}

// event sends an event to FSM, will return true, if there were no errors.
func (app *App) event(ctx context.Context, event string) bool {
	if err := app.fsm.Event(ctx, event); err != nil {
		app.error(err)
		return false
	}
	return true
}

func (app *App) printf(format string, a ...any) {
	_, _ = fmt.Fprintf(app.view.tvLog, format, a...)
}

// modal wraps a primitive in a modal box.
func modal(p tview.Primitive, width int, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
