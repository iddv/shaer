package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FileUploadDialog handles file upload with expiration selection
type FileUploadDialog struct {
	window       fyne.Window
	dialog       *dialog.CustomDialog
	
	// UI components
	fileLabel      *widget.Label
	fileSizeLabel  *widget.Label
	expirationSelect *widget.Select
	progressBar    *widget.ProgressBar
	uploadBtn      *widget.Button
	cancelBtn      *widget.Button
	
	// Data
	selectedFile string
	onUpload     func(filePath string, expiration time.Duration) error
}

// NewFileUploadDialog creates a new file upload dialog
func NewFileUploadDialog(parent fyne.Window, onUpload func(string, time.Duration) error) *FileUploadDialog {
	d := &FileUploadDialog{
		window:   parent,
		onUpload: onUpload,
	}
	
	d.setupDialog()
	return d
}

// Show displays the upload dialog
func (d *FileUploadDialog) Show() {
	d.dialog.Show()
}

// Hide closes the upload dialog
func (d *FileUploadDialog) Hide() {
	d.dialog.Hide()
}

// SetProgress updates the upload progress
func (d *FileUploadDialog) SetProgress(value float64) {
	d.progressBar.SetValue(value)
	if value >= 1.0 {
		d.uploadBtn.SetText("Upload Complete")
		d.uploadBtn.Disable()
		// Auto-close after a brief delay
		time.AfterFunc(2*time.Second, func() {
			d.Hide()
		})
	}
}

func (d *FileUploadDialog) setupDialog() {
	// File selection section
	d.fileLabel = widget.NewLabel("No file selected")
	d.fileLabel.TextStyle = fyne.TextStyle{Italic: true}
	
	d.fileSizeLabel = widget.NewLabel("")
	d.fileSizeLabel.Hide()
	
	selectFileBtn := widget.NewButton("Select File", d.selectFile)
	selectFileBtn.Icon = theme.FolderOpenIcon()
	selectFileBtn.Importance = widget.MediumImportance
	
	// Expiration selection
	expirationLabel := widget.NewLabel("File Expiration:")
	expirationLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.expirationSelect = widget.NewSelect(
		[]string{
			"1 hour",
			"1 day", 
			"1 week",
			"1 month",
		},
		nil,
	)
	d.expirationSelect.SetSelected("1 day") // Default selection
	
	// Progress section
	d.progressBar = widget.NewProgressBar()
	d.progressBar.Hide()
	
	// Action buttons
	d.uploadBtn = widget.NewButton("Upload File", d.uploadFile)
	d.uploadBtn.Icon = theme.UploadIcon()
	d.uploadBtn.Importance = widget.HighImportance
	d.uploadBtn.Disable()
	
	d.cancelBtn = widget.NewButton("Cancel", func() {
		d.Hide()
	})
	
	// Layout
	fileSection := container.NewVBox(
		widget.NewLabel("Select File to Upload"),
		container.NewBorder(nil, nil, nil, selectFileBtn, d.fileLabel),
		d.fileSizeLabel,
	)
	
	expirationSection := container.NewVBox(
		expirationLabel,
		d.expirationSelect,
		widget.NewLabel("Files will be automatically deleted after the expiration time."),
	)
	
	progressSection := container.NewVBox(
		widget.NewLabel("Upload Progress:"),
		d.progressBar,
	)
	
	buttonSection := container.NewHBox(
		d.cancelBtn,
		widget.NewSeparator(),
		d.uploadBtn,
	)
	
	content := container.NewVBox(
		fileSection,
		widget.NewSeparator(),
		expirationSection,
		widget.NewSeparator(),
		progressSection,
		widget.NewSeparator(),
		buttonSection,
	)
	
	d.dialog = dialog.NewCustom("Upload File", "", content, d.window)
	d.dialog.Resize(fyne.NewSize(500, 400))
}

func (d *FileUploadDialog) selectFile() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return // User cancelled or error
		}
		defer reader.Close()
		
		uri := reader.URI()
		d.selectedFile = uri.Path()
		
		// Update UI
		filename := filepath.Base(d.selectedFile)
		d.fileLabel.SetText(filename)
		d.fileLabel.TextStyle = fyne.TextStyle{} // Remove italic
		d.fileLabel.Refresh()
		
		// Show file size
		if info, err := storage.LoadResourceFromURI(uri); err == nil {
			size := len(info.Content())
			d.fileSizeLabel.SetText(fmt.Sprintf("Size: %s", formatFileSize(int64(size))))
			d.fileSizeLabel.Show()
			
			// Check file size limit (100MB as per requirements)
			const maxSize = 100 * 1024 * 1024 // 100MB
			if int64(size) > maxSize {
				d.fileSizeLabel.SetText(fmt.Sprintf("Size: %s (Too large! Maximum 100MB)", formatFileSize(int64(size))))
				d.uploadBtn.Disable()
				return
			}
		}
		
		// Enable upload button
		d.uploadBtn.Enable()
		
	}, d.window)
	
	// Set file filters for common file types
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{
		".txt", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		".zip", ".rar", ".7z", ".tar", ".gz",
	}))
	
	fileDialog.Show()
}

func (d *FileUploadDialog) uploadFile() {
	if d.selectedFile == "" || d.onUpload == nil {
		return
	}
	
	// Parse expiration duration
	expiration := d.parseExpiration(d.expirationSelect.Selected)
	
	// Show progress and disable upload button
	d.progressBar.Show()
	d.progressBar.SetValue(0)
	d.uploadBtn.SetText("Uploading...")
	d.uploadBtn.Disable()
	d.cancelBtn.Disable()
	
	// Start upload in goroutine
	go func() {
		// Simulate progress updates (in real implementation, this would come from the upload callback)
		for i := 0; i <= 10; i++ {
			time.Sleep(100 * time.Millisecond)
			progress := float64(i) / 10.0
			d.progressBar.SetValue(progress)
		}
		
		// Call the upload callback
		if err := d.onUpload(d.selectedFile, expiration); err != nil {
			// Show error dialog
			dialog.ShowError(err, d.window)
			
			// Reset UI
			d.progressBar.Hide()
			d.uploadBtn.SetText("Upload File")
			d.uploadBtn.Enable()
			d.cancelBtn.Enable()
		} else {
			// Success - progress bar will show completion
			d.SetProgress(1.0)
		}
	}()
}

func (d *FileUploadDialog) parseExpiration(selected string) time.Duration {
	switch selected {
	case "1 hour":
		return time.Hour
	case "1 day":
		return 24 * time.Hour
	case "1 week":
		return 7 * 24 * time.Hour
	case "1 month":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour // Default to 1 day
	}
}