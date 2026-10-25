#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
cd "$root"

build=$root/docs/demo/.build
mkdir -p "$build"

cleanup() {
  if [[ -f $build/project-dir ]]; then
    local dir
    dir=$(<"$build/project-dir")
    if [[ $dir == "${TMPDIR:-/tmp}"/wtr-demo.* ]]; then
      rm -rf "$dir"
    fi
    rm -f "$build/project-dir"
  fi
}
trap cleanup EXIT
cleanup

for cmd in vhs ffmpeg ffprobe gifski magick; do
  if ! command -v "$cmd" >/dev/null; then
    echo "demo: $cmd is required (see docs/development.md)" >&2
    exit 1
  fi
done

docs/demo/make-keys.sh
vhs docs/demo/demo.tape

IFS=, read -r width height <<< "$(ffprobe -v error -select_streams v:0 \
  -show_entries stream=width,height -of csv=p=0 "$build/demo.mp4")"

# These times mirror the Sleeps in demo.tape: Ctrl-O at 0.8s and Enter at 3.1s.
# Each keycap appears 0.25s before the key and stays for under a second after it.
hud_y=$((height + 24))
canvas_h=$((height + 96))
filter="[0:v]pad=${width}:${canvas_h}:0:0:color=0x11111b[bg];"
filter+="[bg][1:v]overlay=x=(W-w)/2:y=${hud_y}:enable='between(t,0.55,1.75)'[v1];"
filter+="[v1][2:v]overlay=x=(W-w)/2:y=${hud_y}:enable='between(t,2.85,4.0)'[out]"
ffmpeg -y -v error -i "$build/demo.mp4" \
  -i "$build/keys/ctrl-overlay.png" -i "$build/keys/enter-overlay.png" \
  -filter_complex "$filter" -map "[out]" \
  -c:v libx264 -pix_fmt yuv420p -crf 18 -preset medium -an \
  "$build/demo-hud.mp4"

rm -rf "$build/frames"
mkdir "$build/frames"
ffmpeg -v error -i "$build/demo-hud.mp4" -vf fps=15 "$build/frames/%04d.png"

mkdir -p docs/images
gifski --fps 15 --width "$width" --quality 90 \
  -o docs/images/what-to-run.gif "$build/frames"/*.png >/dev/null

ls -lh docs/images/what-to-run.gif
