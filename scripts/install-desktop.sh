#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
set -eu

prefix=${PREFIX:-"$HOME/.local"}
data_dir=${DATADIR:-"$prefix/share"}
dest_dir=${DESTDIR:-}
app_id=ca.cerberusgames.mailsalongui
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

case "$prefix" in /*) ;; *) echo "PREFIX must be an absolute path" >&2; exit 1 ;; esac
case "$prefix" in
    *=*|*%*) echo "PREFIX cannot contain = or % in a desktop executable path" >&2; exit 1 ;;
esac
case "$data_dir" in /*) ;; *) echo "DATADIR must be an absolute path" >&2; exit 1 ;; esac
# Desktop fields cannot contain newlines. Reject them before creating files.
case "$prefix$data_dir" in
    *'
'*|*"$(printf '\r')"*) echo "Installation paths cannot contain line breaks" >&2; exit 1 ;;
esac

binary="$dest_dir$prefix/bin/MailSalonGUI"
icon_path="$data_dir/icons/hicolor/512x512/apps/$app_id.png"
icon="$dest_dir$icon_path"
entry="$dest_dir$data_dir/applications/$app_id.desktop"

# Never update the host's desktop caches when staging a package.
refresh_desktop() {
    [ -z "$dest_dir" ] || return 0
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$data_dir/applications" ||
            echo "Could not refresh the desktop database" >&2
    fi
    for cache_tool in kbuildsycoca6 kbuildsycoca5; do
        if command -v "$cache_tool" >/dev/null 2>&1; then
            "$cache_tool" --noincremental ||
                echo "Could not refresh the KDE application cache" >&2
            break
        fi
    done
}

case "${1:-install}" in
    uninstall)
        rm -f -- "$entry" "$icon" "$binary"
        refresh_desktop
        echo "Removed MailSalonGUI desktop installation"
        exit 0
        ;;
    install) ;;
    *) echo "Usage: $0 [install|uninstall]" >&2; exit 1 ;;
esac

[ -f "$root/MailSalonGUI" ] || { echo "Build MailSalonGUI before installing" >&2; exit 1; }
install -d -- "$(dirname -- "$binary")" "$(dirname -- "$icon")" "$(dirname -- "$entry")"
install -m 755 -- "$root/MailSalonGUI" "$binary"
install -m 644 -- "$root/internal/assets/icon.png" "$icon"

# Desktop Exec uses both string escaping and shell-like argument quoting.
# Escape percent field codes too; this value is a literal executable path.
desktop_exec=$(printf '%s' "$prefix/bin/MailSalonGUI" |
    sed -e 's/\\/\\\\\\\\/g' -e 's/["`$]/\\\\&/g' -e 's/%/%%/g')
# Icon is a desktop string, not an Exec argument: no quotes or field codes.
# A direct path also avoids stale icon-theme caches in Plasma.
desktop_icon=$(printf '%s' "$icon_path" | sed -e 's/\\/\\\\/g' -e 's/\t/\\t/g')
temporary=$(mktemp)
trap 'rm -f -- "$temporary"' EXIT HUP INT TERM
while IFS= read -r line; do
    case "$line" in
        'Exec=@EXEC@') printf 'Exec="%s"\n' "$desktop_exec" ;;
        'Icon=@ICON@') printf 'Icon=%s\n' "$desktop_icon" ;;
        *) printf '%s\n' "$line" ;;
    esac
done < "$root/packaging/$app_id.desktop.in" > "$temporary"
install -m 644 -- "$temporary" "$entry"
refresh_desktop
echo "Installed MailSalonGUI in $dest_dir$prefix"
echo "Desktop launcher: $entry"
