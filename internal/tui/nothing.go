package tui

import "context"

func (app *App) initNothing(ctx context.Context) {
	app.pages.AddPage(stNothing, app.view.mbNothing, false, false)
	app.view.mbNothing.
		SetDoneFunc(func(_ int, _ string) {
			app.cancel(ctx)
		}).
		SetText("There are no messages to delete").
		AddButtons([]string{btnOK})
}
