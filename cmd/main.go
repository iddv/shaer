package main

import (
	"file-sharing-app/internal/config"
	"file-sharing-app/pkg/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Initialize logging
	log := logger.New()
	log.Info("File Sharing App starting...")

	// Load default configuration
	cfg := config.DefaultConfig()
	log.Info("Configuration loaded")

	// Create Fyne application
	myApp := app.New()
	// Basic app setup

	myWindow := myApp.NewWindow("File Sharing App")
	myWindow.Resize(fyne.NewSize(800, 600))

	// Create basic UI layout
	content := createMainUI(log, cfg)
	myWindow.SetContent(content)

	log.Info("Application UI initialized")
	myWindow.ShowAndRun()
}

func createMainUI(log *logger.Logger, cfg *config.AppConfig) *fyne.Container {
	// Header
	title := widget.NewLabel("File Sharing Application")
	title.TextStyle.Bold = true

	// Status label
	status := widget.NewLabel("Ready - Configure AWS credentials to begin")

	// Placeholder buttons for future functionality
	uploadBtn := widget.NewButton("Upload File", func() {
		log.Info("Upload button clicked (not implemented yet)")
	})
	uploadBtn.Disable()

	settingsBtn := widget.NewButton("Settings", func() {
		log.Info("Settings button clicked (not implemented yet)")
	})
	settingsBtn.Disable()

	// File list placeholder
	fileList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return widget.NewLabel("No files")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	// Layout
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