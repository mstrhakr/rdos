#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="/var/log/uftc-installer.log"
touch "$LOG_FILE"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "=== UFTC installer ==="
date

# Silence console bell and unload pcspkr if it is present.
setterm -blength 0 -bfreq 0 >/dev/null 2>&1 || true
rmmod pcspkr >/dev/null 2>&1 || true

MODE="${1:-guided}"
UI_AVAILABLE=0

# Try to load enhanced UI library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/tc-installer-ui.sh" ]]; then
  source "$SCRIPT_DIR/tc-installer-ui.sh"
  UI_AVAILABLE=1
elif command -v whiptail >/dev/null 2>&1 && [[ -t 0 && -t 1 ]]; then
  UI_AVAILABLE=1
fi

show_msg() {
  local title="$1"
  local text="$2"
  if [[ "$UI_AVAILABLE" == "1" ]] && declare -F ui_info_page >/dev/null 2>&1; then
    ui_info_page "$title" "$text"
  elif command -v whiptail >/dev/null 2>&1; then
    whiptail --title "$title" --msgbox "$text" 12 78 2>/dev/null || true
  else
    echo
    echo "[$title]"
    echo "$text"
  fi
}

confirm_msg() {
  local title="$1"
  local text="$2"
  if [[ "$UI_AVAILABLE" == "1" ]] && declare -F ui_confirm >/dev/null 2>&1; then
    ui_confirm "$text"
  elif command -v whiptail >/dev/null 2>&1; then
    whiptail --title "$title" --yesno "$text" 14 78 2>/dev/null || true
  else
    echo
    echo "[$title] $text"
    read -r -p "Continue? [y/N]: " ans
    [[ "$ans" =~ ^[Yy]$ ]]
  fi
}

run_with_gauge() {
  local target_disk="$1"

  if [[ "$UI_AVAILABLE" != "1" ]] || ! declare -F ui_progress_gauge >/dev/null 2>&1; then
    zstd -d -c "$IMAGE_ZST" | dd of="/dev/$target_disk" bs=16M status=progress conv=fsync
    sync
    return
  fi

  ui_progress_gauge "$target_disk" "$IMAGE_ZST"
}

choose_post_action() {
  local target_disk="${1:-}"

  if [[ "$UI_AVAILABLE" != "1" ]] || ! declare -F ui_post_action_menu >/dev/null 2>&1; then
    echo "poweroff"
    return
  fi

  ui_post_action_menu "$target_disk"
}

IMAGE_ZST="/run/live/medium/uftc/uftc.img.zst"
if [[ ! -f "$IMAGE_ZST" ]]; then
  IMAGE_ZST="/lib/live/mount/medium/uftc/uftc.img.zst"
fi

if [[ ! -f "$IMAGE_ZST" ]]; then
  echo "Installer image payload missing from live media."
  poweroff -f
fi

LIVE_SOURCE="$(findmnt -n -o SOURCE /run/live/medium 2>/dev/null || true)"
if [[ -z "$LIVE_SOURCE" ]]; then
  LIVE_SOURCE="$(findmnt -n -o SOURCE /lib/live/mount/medium 2>/dev/null || true)"
fi
LIVE_DISK=""

if [[ -n "$LIVE_SOURCE" ]]; then
  LIVE_RESOLVED="$(readlink -f "$LIVE_SOURCE" || echo "$LIVE_SOURCE")"
  LIVE_DISK="$(lsblk -ndo PKNAME "$LIVE_RESOLVED" 2>/dev/null || true)"
fi

list_candidate_disks() {
  while read -r disk _type; do
    if [[ -z "$disk" ]]; then
      continue
    fi
    if [[ "$disk" == "$LIVE_DISK" ]]; then
      continue
    fi
    if [[ -f "/sys/block/$disk/removable" ]] && [[ "$(cat "/sys/block/$disk/removable")" != "0" ]]; then
      continue
    fi
    echo "$disk"
  done < <(lsblk -ndo NAME,TYPE | awk '$2=="disk" { print $1, $2 }')
}

auto_target_disk() {
  local d
  while read -r d; do
    if [[ -n "$d" ]]; then
      echo "$d"
      return 0
    fi
  done < <(list_candidate_disks)
  return 1
}

pick_target_disk() {
  if [[ "$UI_AVAILABLE" != "1" ]] || ! declare -F ui_simple_disk_menu >/dev/null 2>&1; then
    auto_target_disk
    return
  fi

  local options=()
  local d size model desc
  while read -r d; do
    size="$(lsblk -dn -o SIZE "/dev/$d" 2>/dev/null || echo "?")"
    model="$(lsblk -dn -o MODEL "/dev/$d" 2>/dev/null | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' || true)"
    if [[ -z "$model" ]]; then
      model="unknown model"
    fi
    desc="$size - $model"
    options+=("$d" "$desc")
  done < <(list_candidate_disks)

  if (( ${#options[@]} == 0 )); then
    return 1
  fi

  if (( ${#options[@]} == 2 )); then
    local only_disk="${options[0]}"
    local only_desc="${options[1]}"

    if declare -F ui_clear >/dev/null 2>&1 && declare -F ui_print_header >/dev/null 2>&1; then
      ui_clear
      ui_print_header
      printf "\n"
      ui_section_header "INSTALL TARGET"
      printf "\n"
      ui_info_message "Only one eligible install target was detected."
      printf "\n"
      printf "  /dev/%s - %s\n" "$only_disk" "$only_desc"
      printf "\n${COLOR_DIM}Press Enter to continue...${COLOR_RESET}"
      read -r
    fi

    echo "$only_disk"
    return 0
  fi

  ui_simple_disk_menu options
}

TARGET_DISK=""

if [[ "$MODE" == "auto" ]]; then
  TARGET_DISK="$(auto_target_disk || true)"
  if [[ -z "$TARGET_DISK" ]]; then
    echo "No install target disk was found."
    echo "Live media disk was: ${LIVE_DISK:-unknown}"
    poweroff -f
  fi
else
  if [[ "$UI_AVAILABLE" == "1" ]] && declare -F ui_print_header >/dev/null 2>&1; then
    ui_clear
    ui_print_header
    printf "\n"
    printf "${COLOR_BRIGHT_CYAN}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n"
    printf "${COLOR_BRIGHT_WHITE}Welcome to UFTC Guided Installation${COLOR_RESET}\n"
    printf "${COLOR_BRIGHT_CYAN}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n\n"
    ui_info_message "Live media disk: ${LIVE_DISK:-unknown}"
    printf "\n${COLOR_DIM}Press Enter to continue...${COLOR_RESET}"
    read -r
  else
    show_msg "UFTC Installer" "Guided mode will let you choose target disk and post-install behavior.\n\nLive media disk: ${LIVE_DISK:-unknown}"
  fi
  
  TARGET_DISK="$(pick_target_disk || true)"
  if [[ -z "$TARGET_DISK" ]]; then
    show_msg "UFTC Installer" "No install target disk was selected or available."
    poweroff -f
  fi
  if ! confirm_msg "Final Warning" "All data on /dev/$TARGET_DISK will be permanently erased. Continue?"; then
    poweroff -f
  fi
fi

echo "Writing image to /dev/$TARGET_DISK"
echo "This will erase all data on /dev/$TARGET_DISK"

run_with_gauge "$TARGET_DISK"

if [[ "$MODE" == "auto" ]]; then
  echo "Install complete. Powering off."
  poweroff -f
fi

post_action="$(choose_post_action "$TARGET_DISK" || echo poweroff)"
case "$post_action" in
  reboot)
    echo "Install complete. Rebooting."
    reboot -f
    ;;
  shell)
    echo "Install complete. Opening shell."
    /bin/bash
    poweroff -f
    ;;
  *)
    echo "Install complete. Powering off."
    poweroff -f
    ;;
esac
