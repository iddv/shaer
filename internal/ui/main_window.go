package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MainWindow represents the main application window
type MainWindow struct {
	window   fyne.Window
	fileList *widget.List
}

// NewMainWindow creates a new main window
func NewMainWindow(app fyne.App) *MainWindow {
	window := app.NewWindow("File Sharing App")
	window.Resize(fyne.NewSize(800, 600))

	return &MainWindow{
		window: window,
	}
}

// Show displays the main window
func (mw *MainWindow) Show() {
	content := mw.createContent()
	mw.window.SetContent(content)
	mw.window.ShowAndRun()
}

func (mw *MainWindow) createContent() *fyne.Container {
	// Placeholder content - will be implemented in later tasks
	title := widget.NewLabel("File Sharing Application")
	title.TextStyle.Bold = true

	status := widget.NewLabel("Ready - Configure AWS credentials to begin")

	uploadBtn := widget.NewButton("Upload File", func() {
		// Will be implemented in later tasks
	})
	uploadBtn.Disable()

	settingsBtn := widget.NewButton("Settings", func() {
		// Will be implemented in later tasks
	})
	settingsBtn.Disable()

	fileList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return widget.NewLabel("No files")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	buttonContainer := container.NewHBox(uploadBtn, settingsBtn)

	return container.NewVBox(
		title,
		status,
		buttonContainer,
		widget.NewSeparator(),
		widget.NewLabel("Files:"),
		fileList,
	)
}