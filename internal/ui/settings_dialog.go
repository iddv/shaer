package ui

import (
	"fmt"

	"file-sharing-app/internal/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SettingsDialog represents the settings configuration dialog
type SettingsDialog struct {
	parent   fyne.Window
	dialog   *dialog.CustomDialog
	settings *models.ApplicationSettings
	
	// Form widgets
	awsRegionEntry      *widget.Entry
	s3BucketEntry       *widget.Entry
	defaultExpirationSelect *widget.Select
	maxFileSizeEntry    *widget.Entry
	uiThemeSelect       *widget.Select
	autoRefreshCheck    *widget.Check
	showNotificationsCheck *widget.Check
	
	// Callbacks
	OnSaveSettings func(settings *models.ApplicationSettings) error
	OnLoadSettings func() (*models.ApplicationSettings, error)
}

// NewSettingsDialog creates a new settings dialog
func NewSettingsDialog(parent fyne.Window) *SettingsDialog {
	sd := &SettingsDialog{
		parent: parent,
	}
	
	sd.createDialog()
	return sd
}

// SetCallbacks sets the callback functions for settings operations
func (sd *SettingsDialog) SetCallbacks(
	onSave func(settings *models.ApplicationSettings) error,
	onLoad func() (*models.ApplicationSettings, error),
) {
	sd.OnSaveSettings = onSave
	sd.OnLoadSettings = onLoad
}

// Show displays the settings dialog
func (sd *SettingsDialog) Show() {
	// Load current settings
	if sd.OnLoadSettings != nil {
		if settings, err := sd.OnLoadSettings(); err == nil {
			sd.settings = settings
			sd.populateForm()
		} else {
			// Use default settings if loading fails
			sd.settings = models.DefaultApplicationSettings()
			sd.populateForm()
			dialog.ShowError(fmt.Errorf("Failed to load settings, using defaults: %v", err), sd.parent)
		}
	} else {
		sd.settings = models.DefaultApplicationSettings()
		sd.populateForm()
	}
	
	sd.dialog.Show()
}

// Hide closes the settings dialog
func (sd *SettingsDialog) Hide() {
	sd.dialog.Hide()
}

func (sd *SettingsDialog) createDialog() {
	// Create form widgets
	sd.createFormWidgets()
	
	// Create form layout
	form := sd.createFormLayout()
	
	// Create action buttons
	buttons := sd.createActionButtons()
	
	// Main content
	content := container.NewVBox(
		form,
		widget.NewSeparator(),
		buttons,
	)
	
	// Create dialog
	sd.dialog = dialog.NewCustom("Application Settings", "Close", content, sd.parent)
	sd.dialog.Resize(fyne.NewSize(500, 600))
}

func (sd *SettingsDialog) createFormWidgets() {
	// AWS Configuration
	sd.awsRegionEntry = widget.NewEntry()
	sd.awsRegionEntry.SetPlaceHolder("e.g., us-west-2")
	
	sd.s3BucketEntry = widget.NewEntry()
	sd.s3BucketEntry.SetPlaceHolder("e.g., my-file-sharing-bucket")
	
	// Default expiration options
	sd.defaultExpirationSelect = widget.NewSelect(
		[]string{"1h", "1d", "1w", "1m"},
		nil,
	)
	
	// Max file size (in MB for user convenience)
	sd.maxFileSizeEntry = widget.NewEntry()
	sd.maxFileSizeEntry.SetPlaceHolder("100")
	
	// UI Theme
	sd.uiThemeSelect = widget.NewSelect(
		[]string{"light", "dark", "auto"},
		nil,
	)
	
	// Boolean settings
	sd.autoRefreshCheck = widget.NewCheck("Automatically refresh file list", nil)
	sd.showNotificationsCheck = widget.NewCheck("Show system notifications", nil)
}

func (sd *SettingsDialog) createFormLayout() *fyne.Container {
	// AWS Configuration section
	awsSection := widget.NewCard("AWS Configuration", "",
		container.NewVBox(
			widget.NewFormItem("AWS Region", sd.awsRegionEntry).Widget,
			widget.NewFormItem("S3 Bucket", sd.s3BucketEntry).Widget,
		),
	)
	
	// File Settings section
	fileSection := widget.NewCard("File Settings", "",
		container.NewVBox(
			widget.NewFormItem("Default Expiration", sd.defaultExpirationSelect).Widget,
			container.NewHBox(
				widget.NewFormItem("Max File Size (MB)", sd.maxFileSizeEntry).Widget,
				widget.NewLabel("Maximum file size for uploads"),
			),
		),
	)
	
	// UI Settings section
	uiSection := widget.NewCard("User Interface", "",
		container.NewVBox(
			widget.NewFormItem("Theme", sd.uiThemeSelect).Widget,
			sd.autoRefreshCheck,
			sd.showNotificationsCheck,
		),
	)
	
	// Help text
	helpText := widget.NewRichTextFromMarkdown(`
**AWS Configuration Help:**
- AWS Region: The AWS region where your S3 bucket is located
- S3 Bucket: The name of your S3 bucket for file storage

**File Settings Help:**
- Default Expiration: How long files remain accessible by default
- Max File Size: Maximum size limit for file uploads (in MB)

**Note:** Changes require application restart to take full effect.
	`)
	helpText.Wrapping = fyne.TextWrapWord
	
	helpSection := widget.NewCard("Help", "", helpText)
	
	return container.NewVBox(
		awsSection,
		fileSection,
		uiSection,
		helpSection,
	)
}

func (sd *SettingsDialog) createActionButtons() *fyne.Container {
	// Save button
	saveBtn := widget.NewButton("Save Settings", sd.saveSettings)
	saveBtn.Importance = widget.HighImportance
	saveBtn.Icon = theme.DocumentSaveIcon()
	
	// Reset to defaults button
	resetBtn := widget.NewButton("Reset to Defaults", sd.resetToDefaults)
	resetBtn.Icon = theme.ViewRefreshIcon()
	
	// Cancel button
	cancelBtn := widget.NewButton("Cancel", func() {
		sd.Hide()
	})
	
	return container.NewHBox(
		resetBtn,
		widget.NewSeparator(),
		cancelBtn,
		saveBtn,
	)
}

func (sd *SettingsDialog) populateForm() {
	if sd.settings == nil {
		return
	}
	
	// Populate AWS settings
	sd.awsRegionEntry.SetText(sd.settings.AWSRegion)
	sd.s3BucketEntry.SetText(sd.settings.S3Bucket)
	
	// Populate file settings
	sd.defaultExpirationSelect.SetSelected(sd.settings.DefaultExpiration)
	sd.maxFileSizeEntry.SetText(fmt.Sprintf("%.0f", float64(sd.settings.MaxFileSize)/(1024*1024)))
	
	// Populate UI settings
	sd.uiThemeSelect.SetSelected(sd.settings.UITheme)
	sd.autoRefreshCheck.SetChecked(sd.settings.AutoRefresh)
	sd.showNotificationsCheck.SetChecked(sd.settings.ShowNotifications)
}

func (sd *SettingsDialog) saveSettings() {
	// Validate form inputs
	if err := sd.validateForm(); err != nil {
		dialog.ShowError(err, sd.parent)
		return
	}
	
	// Update settings from form
	sd.updateSettingsFromForm()
	
	// Save settings
	if sd.OnSaveSettings != nil {
		if err := sd.OnSaveSettings(sd.settings); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save settings: %v", err), sd.parent)
			return
		}
	}
	
	// Show success message and close dialog
	dialog.ShowInformation("Settings Saved", 
		"Settings have been saved successfully. Some changes may require application restart.", 
		sd.parent)
	sd.Hide()
}

func (sd *SettingsDialog) resetToDefaults() {
	dialog.ShowConfirm("Reset Settings", 
		"Are you sure you want to reset all settings to their default values?",
		func(confirmed bool) {
			if confirmed {
				sd.settings = models.DefaultApplicationSettings()
				sd.populateForm()
			}
		}, sd.parent)
}

func (sd *SettingsDialog) validateForm() error {
	// Validate AWS region
	if sd.awsRegionEntry.Text == "" {
		return fmt.Errorf("AWS region cannot be empty")
	}
	
	// Validate S3 bucket
	if sd.s3BucketEntry.Text == "" {
		return fmt.Errorf("S3 bucket name cannot be empty")
	}
	
	// Validate expiration selection
	if sd.defaultExpirationSelect.Selected == "" {
		return fmt.Errorf("Please select a default expiration period")
	}
	
	// Validate max file size
	if sd.maxFileSizeEntry.Text == "" {
		return fmt.Errorf("Max file size cannot be empty")
	}
	
	// Validate theme selection
	if sd.uiThemeSelect.Selected == "" {
		return fmt.Errorf("Please select a UI theme")
	}
	
	return nil
}

func (sd *SettingsDialog) updateSettingsFromForm() {
	if sd.settings == nil {
		sd.settings = models.DefaultApplicationSettings()
	}
	
	// Update AWS settings
	sd.settings.AWSRegion = sd.awsRegionEntry.Text
	sd.settings.S3Bucket = sd.s3BucketEntry.Text
	
	// Update file settings
	sd.settings.DefaultExpiration = sd.defaultExpirationSelect.Selected
	
	// Parse max file size (convert from MB to bytes)
	var maxFileSizeMB float64
	if _, err := fmt.Sscanf(sd.maxFileSizeEntry.Text, "%f", &maxFileSizeMB); err == nil {
		sd.settings.MaxFileSize = int64(maxFileSizeMB * 1024 * 1024)
	}
	
	// Update UI settings
	sd.settings.UITheme = sd.uiThemeSelect.Selected
	sd.settings.AutoRefresh = sd.autoRefreshCheck.Checked
	sd.settings.ShowNotifications = sd.showNotificationsCheck.Checked
}