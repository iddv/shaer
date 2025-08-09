package ui

import (
	"fmt"
	"regexp"
	"strings"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SharingDialog handles file sharing with recipient management
type SharingDialog struct {
	window fyne.Window
	dialog *dialog.CustomDialog
	file   models.FileMetadata
	
	// UI components
	fileInfoLabel    *widget.Label
	emailEntry       *widget.Entry
	addEmailBtn      *widget.Button
	recipientList    *widget.List
	messageEntry     *widget.Entry
	shareBtn         *widget.Button
	cancelBtn        *widget.Button
	
	// Data
	recipients []string
	onShare    func(fileID string, recipients []string, message string) error
}

// NewSharingDialog creates a new sharing dialog
func NewSharingDialog(parent fyne.Window, file models.FileMetadata, onShare func(string, []string, string) error) *SharingDialog {
	d := &SharingDialog{
		window:     parent,
		file:       file,
		recipients: []string{},
		onShare:    onShare,
	}
	
	d.setupDialog()
	return d
}

// Show displays the sharing dialog
func (d *SharingDialog) Show() {
	d.dialog.Show()
}

// Hide closes the sharing dialog
func (d *SharingDialog) Hide() {
	d.dialog.Hide()
}

func (d *SharingDialog) setupDialog() {
	// File information section
	d.fileInfoLabel = widget.NewLabel(fmt.Sprintf("Sharing: %s", d.file.FileName))
	d.fileInfoLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	fileDetails := widget.NewLabel(fmt.Sprintf("Size: %s • Expires: %s", 
		formatFileSize(d.file.FileSize), 
		formatExpiration(d.file.ExpirationDate)))
	fileDetails.TextStyle = fyne.TextStyle{Italic: true}
	
	// Email input section
	emailLabel := widget.NewLabel("Add Recipients:")
	emailLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.emailEntry = widget.NewEntry()
	d.emailEntry.SetPlaceHolder("Enter email address...")
	d.emailEntry.OnSubmitted = func(email string) {
		d.addRecipient()
	}
	
	d.addEmailBtn = widget.NewButton("Add", d.addRecipient)
	d.addEmailBtn.Icon = theme.ContentAddIcon()
	
	// Recipients list
	recipientsLabel := widget.NewLabel("Recipients:")
	recipientsLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.recipientList = widget.NewList(
		func() int { return len(d.recipients) },
		func() fyne.CanvasObject { return d.createRecipientItem() },
		func(id widget.ListItemID, obj fyne.CanvasObject) { d.updateRecipientItem(id, obj) },
	)
	d.recipientList.Resize(fyne.NewSize(400, 120))
	
	// Message section
	messageLabel := widget.NewLabel("Custom Message (optional):")
	messageLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	d.messageEntry = widget.NewMultiLineEntry()
	d.messageEntry.SetPlaceHolder("Add a personal message for recipients...")
	d.messageEntry.Resize(fyne.NewSize(400, 80))
	
	// Action buttons
	d.shareBtn = widget.NewButton("Share File", d.shareFile)
	d.shareBtn.Icon = theme.MailSendIcon()
	d.shareBtn.Importance = widget.HighImportance
	d.shareBtn.Disable() // Disabled until recipients are added
	
	d.cancelBtn = widget.NewButton("Cancel", func() {
		d.Hide()
	})
	
	// Layout
	fileSection := container.NewVBox(
		d.fileInfoLabel,
		fileDetails,
	)
	
	emailSection := container.NewVBox(
		emailLabel,
		container.NewBorder(nil, nil, nil, d.addEmailBtn, d.emailEntry),
	)
	
	recipientsSection := container.NewVBox(
		recipientsLabel,
		container.NewScroll(d.recipientList),
	)
	
	messageSection := container.NewVBox(
		messageLabel,
		container.NewScroll(d.messageEntry),
	)
	
	buttonSection := container.NewHBox(
		d.cancelBtn,
		widget.NewSeparator(),
		d.shareBtn,
	)
	
	content := container.NewVBox(
		fileSection,
		widget.NewSeparator(),
		emailSection,
		widget.NewSeparator(),
		recipientsSection,
		widget.NewSeparator(),
		messageSection,
		widget.NewSeparator(),
		buttonSection,
	)
	
	d.dialog = dialog.NewCustom("Share File", "", content, d.window)
	d.dialog.Resize(fyne.NewSize(550, 600))
}

func (d *SharingDialog) createRecipientItem() fyne.CanvasObject {
	emailLabel := widget.NewLabel("email@example.com")
	
	removeBtn := widget.NewButton("", nil)
	removeBtn.Icon = theme.DeleteIcon()
	removeBtn.Importance = widget.DangerImportance
	
	return container.NewBorder(nil, nil, nil, removeBtn, emailLabel)
}

func (d *SharingDialog) updateRecipientItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(d.recipients) {
		return
	}
	
	email := d.recipients[id]
	border := obj.(*fyne.Container)
	
	// Update email label
	emailLabel := border.Objects[0].(*widget.Label)
	emailLabel.SetText(email)
	
	// Update remove button
	removeBtn := border.Objects[1].(*widget.Button)
	removeBtn.OnTapped = func() {
		d.removeRecipient(id)
	}
}

func (d *SharingDialog) addRecipient() {
	email := strings.TrimSpace(d.emailEntry.Text)
	if email == "" {
		return
	}
	
	// Validate email format
	if !d.isValidEmail(email) {
		dialog.ShowError(fmt.Errorf("invalid email format: %s", email), d.window)
		return
	}
	
	// Check for duplicates
	for _, existing := range d.recipients {
		if existing == email {
			dialog.ShowError(fmt.Errorf("email already added: %s", email), d.window)
			return
		}
	}
	
	// Add recipient
	d.recipients = append(d.recipients, email)
	d.recipientList.Refresh()
	d.emailEntry.SetText("")
	
	// Enable share button if we have recipients
	if len(d.recipients) > 0 {
		d.shareBtn.Enable()
	}
}

func (d *SharingDialog) removeRecipient(index int) {
	if index < 0 || index >= len(d.recipients) {
		return
	}
	
	// Remove recipient
	d.recipients = append(d.recipients[:index], d.recipients[index+1:]...)
	d.recipientList.Refresh()
	
	// Disable share button if no recipients
	if len(d.recipients) == 0 {
		d.shareBtn.Disable()
	}
}

func (d *SharingDialog) shareFile() {
	if len(d.recipients) == 0 || d.onShare == nil {
		return
	}
	
	message := strings.TrimSpace(d.messageEntry.Text)
	
	// Disable buttons during sharing
	d.shareBtn.SetText("Sharing...")
	d.shareBtn.Disable()
	d.cancelBtn.Disable()
	
	// Perform sharing in goroutine
	go func() {
		if err := d.onShare(d.file.ID, d.recipients, message); err != nil {
			// Show error dialog
			dialog.ShowError(err, d.window)
			
			// Re-enable buttons
			d.shareBtn.SetText("Share File")
			d.shareBtn.Enable()
			d.cancelBtn.Enable()
		} else {
			// Success - show confirmation and close
			dialog.ShowInformation("Success", 
				fmt.Sprintf("File shared with %d recipient(s)", len(d.recipients)), 
				d.window)
			d.Hide()
		}
	}()
}

func (d *SharingDialog) isValidEmail(email string) bool {
	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}