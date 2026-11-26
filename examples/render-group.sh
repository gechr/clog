#!/bin/sh
set -eu

tape=$1
output=$2
blank_rows=$3
shift 3

# Clog parks the cursor below a live group. Keep that space in the PTY so every
# task can render, then remove the unused rows inside the GIF's window frame.
rows=$(awk '$1 == "Set" && $2 == "Rows" {print $3}' "$tape")
inset=$(awk '$1 == "Set" && ($2 == "Padding" || $2 == "Margin") {n += $3} END {print n}' "$(dirname "$0")/base.tape")
# Leave the entire rounded corner below the cut, even when padding is smaller than the radius.
frame=$(awk '$1 == "Set" && ($2 == "Margin" || $2 == "BorderRadius") {n += $3} END {print n}' "$(dirname "$0")/base.tape")
height="(h-2*$inset)/$rows*$blank_rows"
keep="max($inset,$frame)"

tmp=$(mktemp -d "${output%/*}/.reel-XXXXXX")
trap 'rm -rf "$tmp"' EXIT
reel run "$tape" "$@" -o "$tmp/full.gif"
magick "$tmp/full.gif" -coalesce -chop "0x%[fx:$height+$inset-$keep]+0+%[fx:h-$inset-$height]" +repage -layers Optimize "$tmp/cropped.gif"
mv "$tmp/cropped.gif" "$output"
