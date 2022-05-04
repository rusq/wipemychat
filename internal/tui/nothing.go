package tui

func (app *App) initNothing() {
	app.pages.AddPage(stNothing, app.view.mbNothing, false, false)
	app.view.mbNothing.
		SetDoneFunc(func(_ int, _ string) {
			app.cancel()
		}).
		SetText("There are no messages to delete").
		AddButtons([]string{btnOK})
}
