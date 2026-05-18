# UFTC Guided Installer - Enhanced UI Guide

## Overview

The UFTC guided installer has been upgraded with a full-screen, colorful, and visually appealing UI that provides a much better user experience than the previous plain whiptail dialogs.

## New Features

### Visual Enhancements

- **ASCII Art Header**: Large, styled UFTC logo with colors and borders
- **Styled Section Headers**: Color-coded section dividers and titles
- **Status Messages**: Icons (✓, ✗, ⚠, ℹ) with colored text for various message types
- **Full-Screen Layout**: Uses the entire terminal for better readability
- **Color Scheme**:
  - Cyan for headers and borders
  - Green for success messages
  - Red for errors and warnings
  - Yellow for warnings
  - Blue for info messages
  - White for user-facing text

### UI Components

#### 1. Welcome Screen

- Shows UFTC logo and welcome message
- Displays the live media disk information
- Prompts to continue to disk selection

#### 2. Disk Selection Menu

- Displays all available disks with:
  - Disk name (`/dev/sdX`)
  - Size and model information
- Simple numbered menu for selection (no arrow keys required)
- Clear warning about data erasure

#### 3. Confirmation Screen

- Large warning banner with clear text
- Final confirmation before proceeding with installation

#### 4. Progress Gauge

- Full-screen progress bar with percentage
- Unicode block characters (█░) for visual representation
- Real-time feedback during installation
- Success message upon completion
- Filesystem sync notification

#### 5. Post-Install Menu

- Three options after successful installation:
  1. Power off (default, recommended)
  2. Reboot immediately
  3. Open shell for troubleshooting
- Clear selection prompt

## How It Works

### File Structure

- **`tcfiles/tc-installer-ui.sh`** - Enhanced UI library with all styling functions
- **`tcfiles/installer-install.sh`** - Updated installer script that uses the UI library
- **`build-installer-iso.sh`** - Updated to include both files in the ISO

### Fallback Behavior

The installer gracefully degrades to previous behavior:

1. If `tc-installer-ui.sh` is available, uses enhanced UI
2. Falls back to `whiptail` if available
3. Falls back to plain text prompts if no UI is available

This ensures compatibility with all environments.

## Testing the Enhanced UI

### Local Testing on Linux/WSL2

```bash
# Source the UI library directly to test components
source tcfiles/tc-installer-ui.sh

# Test the header
ui_clear
ui_print_header

# Test section header
ui_section_header "TEST SECTION"

# Test various messages
ui_info_message "This is an info message"
ui_warning_message "This is a warning"
ui_error_message "This is an error"
ui_success_message "This is a success message"

# Test confirmation
ui_confirm "Do you want to proceed?"
```

### Full Installer Testing

1. Build the ISO with the new UI:

   ```bash
   ./build-installer-iso.sh
   ```

2. Boot the ISO in a VM with the guided installer mode

3. You should see:
   - Large ASCII art UFTC header
   - Styled welcome screen
   - Colorful disk selection menu
   - Progress gauge during installation
   - Post-install action menu

## Customization

You can easily customize the UI by editing `tcfiles/tc-installer-ui.sh`:

- **Colors**: Modify color code variables at the top of the file
- **ASCII Art**: Update the header in `ui_print_header()`
- **Icons**: Change ✓, ✗, ⚠, ℹ symbols in message functions
- **Borders**: Modify the `━`, `║`, `╔`, etc. characters in section headers

## Performance Considerations

The enhanced UI adds minimal overhead:

- Library is ~350 lines of bash
- No external dependencies beyond standard utilities
- Colors use ANSI escape sequences (universal terminal support)
- Progress display uses lightweight polling

## Compatibility

- **ANSI Colors**: Works on any modern terminal (tested on Linux console, SSH, USB serial)
- **Keyboard Input**: Uses simple `read` command compatible with all input methods
- **Unicode**: Uses unicode box-drawing characters; ensure UTF-8 locale for proper display
- **Fallback**: If terminal doesn't support colors, text still displays but without colors

## Known Limitations

- Arrow key navigation is not implemented (uses numbered menu instead) - simpler and more compatible
- Unicode characters may not display correctly on very old terminals
- ANSI colors won't render in non-color terminals (but text will still be readable)

## Files Modified

1. **`tcfiles/tc-installer-ui.sh`** - NEW: Complete UI library
2. **`tcfiles/installer-install.sh`** - MODIFIED: Updated to use enhanced UI
3. **`build-installer-iso.sh`** - MODIFIED: Added tc-installer-ui.sh to ISO

No other files require changes; the enhanced UI is fully backward compatible.
