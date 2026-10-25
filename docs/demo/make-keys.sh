#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
out=$root/docs/demo/.build/keys
mkdir -p "$out"

font=${DEMO_FONT:-}
if [[ -z $font ]]; then
  font=$(fc-match -f '%{file}' 'JetBrainsMono Nerd Font' 2>/dev/null || true)
fi
if [[ -z $font || ! -f $font ]]; then
  echo "demo: install JetBrainsMono Nerd Font or set DEMO_FONT to its .ttf" >&2
  exit 1
fi

keycap() {
  local text=$1 file=$2
  local tw w h=48
  tw=$(magick -font "$font" -pointsize 26 label:"$text" -format '%w' info:)
  w=$((tw + 36))
  magick -size "${w}x${h}" xc:none \
    -fill '#181825' -draw "roundrectangle 0,3 $((w - 1)),$((h - 1)) 9,9" \
    -fill '#313244' -stroke '#585b70' -strokewidth 1 \
    -draw "roundrectangle 0.5,0.5 $((w - 2)),$((h - 5)) 9,9" \
    -stroke none -font "$font" -pointsize 26 -fill '#cdd6f4' \
    -gravity north -annotate +0+9 "$text" \
    "$file"
}

keycap Ctrl "$out/ctrl.png"
keycap O "$out/o.png"
keycap Enter "$out/enter.png"

magick -size 24x48 xc:none -font "$font" -pointsize 22 -fill '#7f849c' \
  -gravity center -annotate +0+0 '+' "$out/plus.png"
magick "$out/ctrl.png" "$out/plus.png" "$out/o.png" +append "$out/ctrl-group.png"

magick -background none -font "$font" -pointsize 18 -fill '#7f849c' \
  label:'press' +repage -gravity center -background none -extent x48 +repage "$out/press.png"
magick -size 12x48 xc:none "$out/spacer.png"

magick "$out/press.png" "$out/spacer.png" "$out/ctrl-group.png" +append "$out/ctrl-overlay.png"
magick "$out/press.png" "$out/spacer.png" "$out/enter.png" +append "$out/enter-overlay.png"
