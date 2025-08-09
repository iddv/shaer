package ui

import (
	"fmt"
	"time"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MainWindow represents the main application window
type MainWindow struct {
	app      fyne.App
	window   fyne.Window
	fileList *widget.List
	
	// UI components
	statusLabel   *widget.Label
	uploadBtn     *widget.Button
	settingsBtn   *widget.Button
	refreshBtn    *widget.Button
	
	// Data
	files []models.FileMetadata
	
	// Callbacks for business logic integration (will be set by main app)
	OnUploadFile func(filePath string, expiration time.Duration) error
	OnShareFile  func(fileID string, recipients []string, message string) error
	OnDeleteFile func(fileID string) error
	OnRefreshFiles func() ([]models.FileMetadata, error)
}

// NewMainWindow creates a new main window
func NewMainWindow(app fyne.App) *MainWindow {
	window := app.NewWindow("File Sharing App")
	window.Resize(fyne.NewSize(900, 700))
	window.SetIcon(theme.DocumentIcon())

	mw := &MainWindow{
		app:    app,
		window: window,
		files:  []models.FileMetadata{},
	}

	mw.setupUI()
	return mw
}

// Show displays the main window
func (mw *MainWindow) Show() {
	mw.window.ShowAndRun()
}

// UpdateFiles updates the file list display
func (mw *MainWindow) UpdateFiles(files []models.FileMetadata) {
	mw.files = files
	mw.fileList.Refresh()
}

// SetStatus updates the status label
func (mw *MainWindow) SetStatus(status string) {
	mw.statusLabel.SetText(status)
}

// EnableActions enables/disables action buttons
func (mw *MainWindow) EnableActions(enabled bool) {
	if enabled {
		mw.uploadBtn.Enable()
		mw.refreshBtn.Enable()
	} else {
		mw.uploadBtn.Disable()
		mw.refreshBtn.Disable()
	}
}

func (mw *MainWindow) setupUI() {
	// Create UI components
	mw.createComponents()
	
	// Set up the main layout
	content := mw.createLayout()
	mw.window.SetContent(content)
}

func (mw *MainWindow) createComponents() {
	// Status label
	mw.statusLabel = widget.NewLabel("Ready - Configure AWS credentials to begin")
	mw.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Action buttons
	mw.uploadBtn = widget.NewButton("Upload Files", mw.showUploadDialog)
	mw.uploadBtn.Importance = widget.HighImportance
	mw.uploadBtn.Disable()

	mw.settingsBtn = widget.NewButton("Settings", mw.showSettingsDialog)
	mw.settingsBtn.Icon = theme.SettingsIcon()

	mw.refreshBtn = widget.NewButton("Refresh", mw.refreshFiles)
	mw.refreshBtn.Icon = theme.ViewRefreshIcon()
	mw.refreshBtn.Disable()

	// File list
	mw.fileList = widget.NewList(
		func() int { return len(mw.files) },
		func() fyne.CanvasObject { return mw.createFileListItem() },
		func(id widget.ListItemID, obj fyne.CanvasObject) { mw.updateFileListItem(id, obj) },
	)
}

func (mw *MainWindow) createLayout() *fyne.Container {
	// Header with title
	title := widget.NewLabel("File Sharing Application")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Toolbar with action buttons
	toolbar := container.NewHBox(
		mw.uploadBtn,
		widget.NewSeparator(),
		mw.refreshBtn,
		widget.NewSeparator(),
		mw.settingsBtn,
	)

	// Files section header
	filesHeader := widget.NewLabel("Your Files")
	filesHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Empty state for file list
	emptyState := widget.NewLabel("No files uploaded yet. Click 'Upload Files' to get started.")
	emptyState.Alignment = fyne.TextAlignCenter
	emptyState.TextStyle = fyne.TextStyle{Italic: true}

	// File list container with empty state handling
	fileContainer := container.NewStack(
		mw.fileList,
		container.NewCenter(emptyState),
	)

	// Show/hide empty state based on file count
	if len(mw.files) > 0 {
		emptyState.Hide()
	} else {
		mw.fileList.Hide()
	}

	// Main content area
	content := container.NewBorder(
		// Top
		container.NewVBox(
			title,
			widget.NewSeparator(),
			toolbar,
			widget.NewSeparator(),
			filesHeader,
		),
		// Bottom
		mw.statusLabel,
		// Left, Right
		nil, nil,
		// Center
		fileContainer,
	)

	return content
}

func (mw *MainWindow) createFileListItem() fyne.CanvasObject {
	// File icon
	icon := widget.NewIcon(theme.DocumentIcon())
	icon.Resize(fyne.NewSize(32, 32))

	// File info labels
	nameLabel := widget.NewLabel("Filename")
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	sizeLabel := widget.NewLabel("Size")
	dateLabel := widget.NewLabel("Upload Date")
	expirationLabel := widget.NewLabel("Expires")
	statusLabel := widget.NewLabel("Status")

	// Action buttons
	copyLinkBtn := widget.NewButton("Copy Link", nil)
	copyLinkBtn.Icon = theme.ContentCopyIcon()
	copyLinkBtn.Importance = widget.MediumImportance

	shareBtn := widget.NewButton("Share", nil)
	shareBtn.Icon = theme.MailSendIcon()

	deleteBtn := widget.NewButton("Delete", nil)
	deleteBtn.Icon = theme.DeleteIcon()
	deleteBtn.Importance = widget.DangerImportance

	// Progress bar (hidden by default)
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// Layout
	infoContainer := container.NewVBox(
		nameLabel,
		container.NewHBox(sizeLabel, widget.NewLabel("•"), dateLabel),
		container.NewHBox(expirationLabel, widget.NewLabel("•"), statusLabel),
		progressBar,
	)

	actionContainer := container.NewHBox(
		copyLinkBtn,
		shareBtn,
		deleteBtn,
	)

	return container.NewBorder(
		nil, nil,
		container.NewHBox(icon, infoContainer),
		actionContainer,
		nil,
	)
}

func (mw *MainWindow) updateFileListItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(mw.files) {
		return
	}

	file := mw.files[id]
	border := obj.(*fyne.Container)
	
	// Get the left container with icon and info
	leftContainer := border.Objects[0].(*fyne.Container)
	infoContainer := leftContainer.Objects[1].(*fyne.Container)
	
	// Get the right container with actions
	actionContainer := border.Objects[1].(*fyne.Container)

	// Update file info
	nameLabel := infoContainer.Objects[0].(*widget.Label)
	nameLabel.SetText(file.FileName)

	sizeInfoContainer := infoContainer.Objects[1].(*fyne.Container)
	sizeLabel := sizeInfoContainer.Objects[0].(*widget.Label)
	dateLabel := sizeInfoContainer.Objects[2].(*widget.Label)
	sizeLabel.SetText(formatFileSize(file.FileSize))
	dateLabel.SetText(formatRelativeTime(file.UploadDate))

	expirationInfoContainer := infoContainer.Objects[2].(*fyne.Container)
	expirationLabel := expirationInfoContainer.Objects[0].(*widget.Label)
	statusLabel := expirationInfoContainer.Objects[2].(*widget.Label)
	expirationLabel.SetText(formatExpiration(file.ExpirationDate))
	statusLabel.SetText(formatStatus(file.Status))

	// Update progress bar visibility
	progressBar := infoContainer.Objects[3].(*widget.ProgressBar)
	if file.Status == models.StatusUploading {
		progressBar.Show()
		// Progress value would be set by upload callback
	} else {
		progressBar.Hide()
	}

	// Update action buttons
	copyLinkBtn := actionContainer.Objects[0].(*widget.Button)
	shareBtn := actionContainer.Objects[1].(*widget.Button)
	deleteBtn := actionContainer.Objects[2].(*widget.Button)

	// Set button callbacks
	copyLinkBtn.OnTapped = func() { mw.copyFileLink(file.ID) }
	shareBtn.OnTapped = func() { mw.showSharingDialog(file) }
	deleteBtn.OnTapped = func() { mw.confirmDeleteFile(file) }

	// Enable/disable buttons based on file status
	canShare := file.Status == models.StatusActive
	copyLinkBtn.Enable()
	if canShare {
		shareBtn.Enable()
	} else {
		shareBtn.Disable()
	}
	deleteBtn.Enable()

	// Apply status-based styling
	mw.applyStatusStyling(obj, file.Status)
}

func (mw *MainWindow) showUploadDialog() {
	uploadDialog := NewFileUploadDialog(mw.window, mw.OnUploadFile)
	uploadDialog.Show()
}

func (mw *MainWindow) showSharingDialog(file models.FileMetadata) {
	sharingDialog := NewSharingDialog(mw.window, file, mw.OnShareFile)
	sharingDialog.Show()
}

func (mw *MainWindow) showSettingsDialog() {
	// Placeholder for settings dialog
	dialog.ShowInformation("Settings", "Settings dialog will be implemented in a future task.", mw.window)
}

func (mw *MainWindow) refreshFiles() {
	if mw.OnRefreshFiles != nil {
		mw.SetStatus("Refreshing files...")
		if files, err := mw.OnRefreshFiles(); err == nil {
			mw.UpdateFiles(files)
			mw.SetStatus("Files refreshed")
		} else {
			mw.SetStatus("Error refreshing files: " + err.Error())
		}
	}
}

func (mw *MainWindow) copyFileLink(fileID string) {
	// This would generate and copy the presigned URL
	// For now, show a placeholder
	dialog.ShowInformation("Copy Link", "Link copied to clipboard (functionality will be implemented in future task)", mw.window)
}

func (mw *MainWindow) confirmDeleteFile(file models.FileMetadata) {
	dialog.ShowConfirm(
		"Delete File",
		fmt.Sprintf("Are you sure you want to delete '%s'? This action cannot be undone.", file.FileName),
		func(confirmed bool) {
			if confirmed && mw.OnDeleteFile != nil {
				if err := mw.OnDeleteFile(file.ID); err != nil {
					dialog.ShowError(err, mw.window)
				} else {
					mw.refreshFiles()
				}
			}
		},
		mw.window,
	)
}

func (mw *MainWindow) applyStatusStyling(obj fyne.CanvasObject, status models.FileStatus) {
	// Apply visual styling based on file status
	// This is a simplified approach - in a real app you might use custom themes
	border := obj.(*fyne.Container)
	
	switch status {
	case models.StatusUploading:
		// Blue tint for uploading files
		border.Objects[0].(*fyne.Container).Objects[0].(*widget.Icon).SetResource(theme.UploadIcon())
	case models.StatusActive:
		// Green tint for active files
		border.Objects[0].(*fyne.Container).Objects[0].(*widget.Icon).SetResource(theme.ConfirmIcon())
	case models.StatusExpired:
		// Gray out expired files
		border.Objects[0].(*fyne.Container).Objects[0].(*widget.Icon).SetResource(theme.WarningIcon())
	case models.StatusError:
		// Red tint for error files
		border.Objects[0].(*fyne.Container).Objects[0].(*widget.Icon).SetResource(theme.ErrorIcon())
	default:
		border.Objects[0].(*fyne.Container).Objects[0].(*widget.Icon).SetResource(theme.DocumentIcon())
	}
}

// Utility functions for formatting
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
}

func formatExpiration(t time.Time) string {
	now := time.Now()
	if t.Before(now) {
		return "expired"
	}
	
	diff := t.Sub(now)
	if diff < time.Hour {
		return fmt.Sprintf("expires in %d min", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("expires in %d hours", int(diff.Hours()))
	} else {
		return fmt.Sprintf("expires in %d days", int(diff.Hours()/24))
	}
}

func formatStatus(status models.FileStatus) string {
	switch status {
	case models.StatusUploading:
		return "Uploading..."
	case models.StatusActive:
		return "Ready"
	case models.StatusExpired:
		return "Expired"
	case models.StatusDeleted:
		return "Deleted"
	case models.StatusError:
		return "Error"
	default:
		return string(status)
	}
}