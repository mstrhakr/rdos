#!/usr/bin/env bash
# UFTC Enhanced Installer UI Library
# Provides styled, full-screen UI components with colors and ASCII art

# Color codes
readonly COLOR_RESET='\033[0m'
readonly COLOR_BOLD='\033[1m'
readonly COLOR_DIM='\033[2m'
readonly COLOR_UNDERLINE='\033[4m'

# Foreground colors
readonly COLOR_BLACK='\033[30m'
readonly COLOR_RED='\033[31m'
readonly COLOR_GREEN='\033[32m'
readonly COLOR_YELLOW='\033[33m'
readonly COLOR_BLUE='\033[34m'
readonly COLOR_MAGENTA='\033[35m'
readonly COLOR_CYAN='\033[36m'
readonly COLOR_WHITE='\033[37m'

# Bright colors
readonly COLOR_BRIGHT_RED='\033[91m'
readonly COLOR_BRIGHT_GREEN='\033[92m'
readonly COLOR_BRIGHT_YELLOW='\033[93m'
readonly COLOR_BRIGHT_BLUE='\033[94m'
readonly COLOR_BRIGHT_MAGENTA='\033[95m'
readonly COLOR_BRIGHT_CYAN='\033[96m'
readonly COLOR_BRIGHT_WHITE='\033[97m'

# Background colors
readonly BG_BLACK='\033[40m'
readonly BG_RED='\033[41m'
readonly BG_GREEN='\033[42m'
readonly BG_YELLOW='\033[43m'
readonly BG_BLUE='\033[44m'
readonly BG_MAGENTA='\033[45m'
readonly BG_CYAN='\033[46m'
readonly BG_WHITE='\033[47m'

# Clear screen and position cursor
ui_clear() {
  clear
  printf '\033[H'
}

# Print UFTC ASCII art header
ui_print_header() {
  printf '%b' "${COLOR_BRIGHT_CYAN}"
  cat << 'EOF'
╔═══════════════════════════════════════════════════════════════════════╗
║                                                                       ║
║                                                                       ║
║       █████  █████    ███████████    ███████████      █████████       ║
║      ░░███  ░░███    ░░███░░░░░░█   ░█░░░███░░░█     ███░░░░░███      ║
║       ░███   ░███     ░███   █ ░    ░   ░███  ░     ███     ░░░       ║
║       ░███   ░███     ░███████          ░███       ░███               ║
║       ░███   ░███     ░███░░░█          ░███       ░███               ║
║       ░███   ░███     ░███  ░           ░███       ░░███     ███      ║
║       ░░████████      █████             █████       ░░█████████       ║
║        ░░░░░░░░      ░░░░░             ░░░░░         ░░░░░░░░░        ║
║                                                                       ║
║                                                                       ║
║                       User Friendly Thin Client                       ║
║                          Guided Installation                          ║
║                                                                       ║
╚═══════════════════════════════════════════════════════════════════════╝
EOF
  printf '%b\n' "${COLOR_RESET}"
}

# Print a styled section header
ui_section_header() {
  local title="$1"
  printf "${COLOR_BRIGHT_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
  printf "${COLOR_BRIGHT_WHITE}${COLOR_BOLD} ▶ $title${COLOR_RESET}\n"
  printf "${COLOR_BRIGHT_CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${COLOR_RESET}\n"
}

# Print an info message with icon
ui_info_message() {
  local message="$1"
  printf "${COLOR_BRIGHT_BLUE}ℹ${COLOR_RESET}  $message\n"
}

# Print a warning message with icon
ui_warning_message() {
  local message="$1"
  printf "${COLOR_BRIGHT_YELLOW}⚠${COLOR_RESET}  ${COLOR_YELLOW}$message${COLOR_RESET}\n"
}

# Print an error message with icon
ui_error_message() {
  local message="$1"
  printf "${COLOR_BRIGHT_RED}✗${COLOR_RESET}  ${COLOR_RED}$message${COLOR_RESET}\n"
}

# Print a success message with icon
ui_success_message() {
  local message="$1"
  printf "${COLOR_BRIGHT_GREEN}✓${COLOR_RESET}  ${COLOR_GREEN}$message${COLOR_RESET}\n"
}

# Print a checkbox-style message
ui_check_message() {
  local message="$1"
  local status="${2:-pending}"  # pending, done, error
  
  case "$status" in
    done)
      printf "${COLOR_BRIGHT_GREEN}☑${COLOR_RESET}  ${COLOR_GREEN}$message${COLOR_RESET}\n"
      ;;
    error)
      printf "${COLOR_BRIGHT_RED}☒${COLOR_RESET}  ${COLOR_RED}$message${COLOR_RESET}\n"
      ;;
    *)
      printf "${COLOR_DIM}☐${COLOR_RESET}  ${COLOR_DIM}$message${COLOR_RESET}\n"
      ;;
  esac
}

# Styled yes/no confirmation prompt
ui_confirm() {
  local message="$1"
  local default="${2:-n}"  # 'y' or 'n'
  
  printf "\n"
  ui_warning_message "$message"
  printf "\n"
  
  if [[ "$default" == "y" ]]; then
    printf "${COLOR_BRIGHT_WHITE}Continue? [${COLOR_BRIGHT_GREEN}Y${COLOR_RESET}${COLOR_BRIGHT_WHITE}/n]: ${COLOR_RESET}"
  else
    printf "${COLOR_BRIGHT_WHITE}Continue? [y/${COLOR_BRIGHT_RED}N${COLOR_RESET}${COLOR_BRIGHT_WHITE}]: ${COLOR_RESET}"
  fi
  
  read -r -p "" ans
  if [[ -z "$ans" ]]; then
    [[ "$default" == "y" ]]
  else
    [[ "$ans" =~ ^[Yy]$ ]]
  fi
}

# Styled menu with disk information
ui_disk_menu() {
  local -n disk_array=$1
  local selected_index=0
  
  while true; do
    {
      ui_clear
      ui_print_header
      printf "\n"
      ui_section_header "SELECT INSTALL TARGET DISK"
      printf "\n"
      ui_warning_message "All data on the selected disk will be permanently erased"
      printf "\n"
    } >&2
    
    local idx=0
    for ((i = 0; i < ${#disk_array[@]}; i += 2)); do
      local disk="${disk_array[$i]}"
      local desc="${disk_array[$((i+1))]}"
      
      if [[ $i -eq $((selected_index * 2)) ]]; then
        printf "${COLOR_BRIGHT_WHITE}${BG_BLUE} ▶ /dev/${disk}  │  $desc ${COLOR_RESET}\n" >&2
      else
        printf "${COLOR_DIM}   /dev/${disk}  │  $desc${COLOR_RESET}\n" >&2
      fi
    done
    
    printf "\n${COLOR_DIM}Use ↑/↓ to select, Enter to confirm${COLOR_RESET}\n" >&2
    printf "\nSelection: " >&2
    
    # Simple keyboard navigation (can be enhanced if needed)
    read -r -s -n1 key
    case "$key" in
      $'\x0a')  # Enter
        echo "${disk_array[$((selected_index * 2))]}"
        return 0
        ;;
      $'\x1b')  # Escape sequence
        read -r -s -n2 key
        case "$key" in
          '[A')  # Up arrow
            selected_index=$((selected_index - 1))
            if (( selected_index < 0 )); then
              selected_index=$((${#disk_array[@]} / 2 - 1))
            fi
            ;;
          '[B')  # Down arrow
            selected_index=$((selected_index + 1))
            if (( selected_index >= ${#disk_array[@]} / 2 )); then
              selected_index=0
            fi
            ;;
        esac
        ;;
    esac
  done
}

# Simplified disk selection menu (if arrow keys don't work)
ui_simple_disk_menu() {
  local -n disk_array=$1
  
  {
    ui_clear
    ui_print_header
    printf "\n"
    ui_section_header "SELECT INSTALL TARGET DISK"
    printf "\n"
    ui_warning_message "All data on the selected disk will be permanently erased"
    printf "\n"
  } >&2
  
  local idx=1
  for ((i = 0; i < ${#disk_array[@]}; i += 2)); do
    local disk="${disk_array[$i]}"
    local desc="${disk_array[$((i+1))]}"
    printf "  ${COLOR_BRIGHT_WHITE}[$idx]${COLOR_RESET}  /dev/${disk}  │  $desc\n" >&2
    ((idx++))
  done
  
  printf "\n${COLOR_BRIGHT_WHITE}Enter disk number [1-$((${#disk_array[@]} / 2))]: ${COLOR_RESET}" >&2
  read -r choice
  
  if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#disk_array[@]} / 2 )); then
    local idx=$((choice - 1))
    echo "${disk_array[$((idx * 2))]}"
    return 0
  fi
  
  return 1
}

# Full-screen progress gauge with styled output
ui_progress_gauge() {
  local target_disk="$1"
  local image_path="$2"
  local log_file="${LOG_FILE:-/var/log/uftc-installer.log}"
  local dev="/dev/$target_disk"
  local expected_bytes=""
  local start_sectors="0"

  # Convert size+unit (for example: 14.0 GiB) to bytes.
  _size_to_bytes() {
    local value="$1"
    local unit="$2"
    awk -v v="$value" -v u="$unit" '
      BEGIN {
        m = 1
        if (u == "B" || u == "bytes") m = 1
        else if (u == "KB") m = 1000
        else if (u == "MB") m = 1000 * 1000
        else if (u == "GB") m = 1000 * 1000 * 1000
        else if (u == "TB") m = 1000 * 1000 * 1000 * 1000
        else if (u == "KiB") m = 1024
        else if (u == "MiB") m = 1024 * 1024
        else if (u == "GiB") m = 1024 * 1024 * 1024
        else if (u == "TiB") m = 1024 * 1024 * 1024 * 1024

        printf "%.0f\n", (v * m)
      }
    '
  }

  # Try to read uncompressed image size from zstd metadata.
  _detect_expected_bytes() {
    local line value unit
    line="$(zstd --list -v "$image_path" 2>>"$log_file" | awk '/^[[:space:]]*[0-9]+[[:space:]]+[0-9]+[[:space:]]/ {print; exit}')"
    if [[ -n "$line" ]]; then
      set -- $line
      if (( $# >= 6 )); then
        value="$5"
        unit="$6"
        if [[ "$value" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
          _size_to_bytes "$value" "$unit"
          return 0
        fi
      fi
    fi

    # Fallback: UFTC images are built as fixed 14 GiB VHDs.
    _size_to_bytes "14" "GiB"
  }

  _read_written_sectors() {
    local stat_path="/sys/class/block/$target_disk/stat"
    if [[ -r "$stat_path" ]]; then
      awk '{print $7}' "$stat_path" 2>/dev/null || echo 0
    else
      echo 0
    fi
  }

  expected_bytes="$(_detect_expected_bytes)"
  start_sectors="$(_read_written_sectors)"
  
  ui_clear
  ui_print_header
  printf "\n"
  ui_section_header "INSTALLING UFTC"
  printf "\n"
  ui_info_message "Writing image to /dev/$target_disk"
  printf "\n"
  
  # Start the write operation in the background and track one process.
  zstd -d -c "$image_path" 2>>"$log_file" | dd of="$dev" bs=16M conv=fsync status=none 2>>"$log_file" &
  local bg_pid=$!
  
  # Update progress from real block-device write counters.
  local pct=0
  local current_sectors=0
  local written_sectors=0
  local written_bytes=0
  local bar_width=30
  local fill=0
  local written_mib=0
  local total_mib=0
  while kill -0 "$bg_pid" 2>/dev/null; do
    current_sectors="$(_read_written_sectors)"
    if [[ "$current_sectors" =~ ^[0-9]+$ ]] && [[ "$start_sectors" =~ ^[0-9]+$ ]] && (( current_sectors >= start_sectors )); then
      written_sectors=$((current_sectors - start_sectors))
    else
      written_sectors=0
    fi
    written_bytes=$((written_sectors * 512))

    if [[ "$expected_bytes" =~ ^[0-9]+$ ]] && (( expected_bytes > 0 )); then
      pct=$((written_bytes * 100 / expected_bytes))
      if (( pct > 99 )); then
        pct=99
      fi
      total_mib=$((expected_bytes / 1024 / 1024))
    else
      pct=0
      total_mib=0
    fi

    written_mib=$((written_bytes / 1024 / 1024))
    fill=$((pct * bar_width / 100))

    printf "\r  "
    printf "${COLOR_BRIGHT_GREEN}"
    for ((i = 0; i < fill; i++)); do
      printf "█"
    done
    printf "${COLOR_DIM}"
    for ((i = fill; i < bar_width; i++)); do
      printf "░"
    done
    if (( total_mib > 0 )); then
      printf "${COLOR_RESET}  %3d%%  (%d/%d MiB)" "$pct" "$written_mib" "$total_mib"
    else
      printf "${COLOR_RESET}  writing... (%d MiB)" "$written_mib"
    fi
    
    sleep 1
  done
  
  if ! wait "$bg_pid"; then
    printf "\n\n"
    ui_error_message "Install failed while writing image. See $log_file"
    return 1
  fi
  
  printf "\r  "
  printf "${COLOR_BRIGHT_GREEN}"
  for ((i = 0; i < bar_width; i++)); do
    printf "█"
  done
  printf "${COLOR_RESET}  100%%\n"
  
  printf "\n"
  ui_success_message "Image written successfully"
  printf "\n${COLOR_DIM}Syncing filesystem...${COLOR_RESET}\n"
  sync
  printf "${COLOR_BRIGHT_GREEN}✓${COLOR_RESET} ${COLOR_GREEN}Sync complete${COLOR_RESET}\n"
}

# Menu for post-install action
ui_post_action_menu() {
  local target_disk="$1"

  {
    ui_clear
    ui_print_header
    printf "\n"
    ui_section_header "INSTALLATION COMPLETE"
    printf "\n"
    ui_success_message "UFTC has been successfully installed to /dev/$target_disk"
    printf "\n"
    printf "${COLOR_BRIGHT_WHITE}What would you like to do next?${COLOR_RESET}\n\n"
  } >&2
  
  printf "  ${COLOR_BRIGHT_WHITE}[1]${COLOR_RESET}  Power off (recommended for USB removal)\n" >&2
  printf "  ${COLOR_BRIGHT_WHITE}[2]${COLOR_RESET}  Reboot immediately\n" >&2
  printf "  ${COLOR_BRIGHT_WHITE}[3]${COLOR_RESET}  Open shell for troubleshooting\n" >&2
  
  printf "\n${COLOR_BRIGHT_WHITE}Select [1-3]: ${COLOR_RESET}" >&2
  read -r choice
  
  case "$choice" in
    2)
      echo "reboot"
      ;;
    3)
      echo "shell"
      ;;
    *)
      echo "poweroff"
      ;;
  esac
}

# Display a status with a spinner
ui_status_spinner() {
  local message="$1"
  local status="${2:-running}"
  
  case "$status" in
    done)
      printf "${COLOR_BRIGHT_GREEN}✓${COLOR_RESET}  ${COLOR_GREEN}$message${COLOR_RESET}\n"
      ;;
    error)
      printf "${COLOR_BRIGHT_RED}✗${COLOR_RESET}  ${COLOR_RED}$message${COLOR_RESET}\n"
      ;;
    *)
      printf "${COLOR_BRIGHT_BLUE}⟳${COLOR_RESET}  ${COLOR_BLUE}$message${COLOR_RESET}\n"
      ;;
  esac
}

# Display a full-screen info page
ui_info_page() {
  local title="$1"
  local message="$2"
  
  ui_clear
  ui_print_header
  printf "\n"
  ui_section_header "$title"
  printf "\n"
  echo "$message"
  printf "\n"
  printf "${COLOR_DIM}Press Enter to continue...${COLOR_RESET}"
  read -r
}

# Export all functions
export -f ui_clear ui_print_header ui_section_header ui_info_message ui_warning_message
export -f ui_error_message ui_success_message ui_check_message ui_confirm
export -f ui_disk_menu ui_simple_disk_menu ui_progress_gauge ui_post_action_menu
export -f ui_status_spinner ui_info_page
