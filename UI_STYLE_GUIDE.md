# UI Style Guide - File Sharing Desktop App

## Design Philosophy
- **Simplicity First**: Clean, uncluttered interface focusing on core actions
- **Native Feel**: Use Fyne's default theme and standard widgets
- **Progressive Disclosure**: Show essential info first, details on demand
- **Clear Status**: Always communicate what's happening and file states

## Visual Style

### Colors & Theme
- Use Fyne's built-in theme system (`theme.PrimaryColor()`, `theme.SuccessColor()`, etc.)
- **File Status Colors:**
  - Active files: subtle green tint
  - Expiring soon: yellow/orange tint  
  - Expired: grayed out
  - Uploading: blue tint with progress

### Typography
- **Primary text**: Default theme sizes for filenames and labels
- **Secondary text**: Smaller, muted color for metadata (dates, sizes, counts)
- **Avoid** custom fonts or unusual text styling

### Layout & Spacing
- Use standard Fyne spacing: `theme.Padding()` and `theme.InnerPadding()`
- **Hierarchy**: Toolbar → Main Content → Status Bar
- **Generous whitespace** around primary actions

## Key UI Patterns

### Primary Actions
- **Upload button**: Most prominent element, easily accessible
- **Copy Link**: Quick one-click sharing for each file
- **Share**: Opens detailed sharing options

### File Display
- **List format** with file icon, name, and key metadata
- **Status indicators**: Visual cues for file state (active, expiring, uploading)
- **Actions per file**: Copy link and detailed share options

### Progress & Feedback
- **Upload progress**: Show per-file and overall progress
- **Status bar**: Current operation and connection state
- **Loading states**: Use spinners for ongoing operations

### Dialogs
- **Modal dialogs** for upload and sharing workflows
- **Clear cancel/confirm** button patterns
- **Form validation** with inline error messages

## Content Guidelines

### Labels & Text
- **Concise button text**: "Upload Files", "Copy Link", "Share"
- **Clear status messages**: "Uploading...", "Ready", "Connected to AWS"
- **Helpful empty states**: Guide users to first actions

### File Information
- Show: filename, upload date, expiration, file size
- **Relative dates**: "2 hours ago", "expires in 3 days"
- **Smart truncation**: Long filenames with tooltips

## Responsive Behavior
- **Minimum window size**: Ensure all elements remain usable
- **Graceful degradation**: Handle offline states and errors clearly
- **Keyboard navigation**: Support standard desktop patterns (Tab, Enter, Escape)

## Implementation Notes
- Prefer standard Fyne widgets over custom components
- Use `container.NewBorder()` for main layout structure
- Handle state changes smoothly (file uploads, status updates)
- Test with various file types and sizes