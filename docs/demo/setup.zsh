# Sourced by demo.tape from the repository root. Prepares a clean shell for the
# recording: the build binary on PATH as `what-to-run` (the picker title), a
# fresh XDG config so the default classic theme is used, a short prompt and a
# bar cursor.
#
# The fixture is copied outside the repository: the Taskfile source searches
# parent directories, so a fixture inside the checkout would pick up the
# project's own Taskfile. The prompt says `~/demo` instead of the temp path.

typeset -g demo_root=${0:A:h:h:h}
if [[ ! -d $demo_root/docs/demo/fixture ]]; then
  print -u2 "demo: cannot find docs/demo/fixture from $demo_root"
  return 1
fi

typeset -g demo_build=$demo_root/docs/demo/.build
if [[ ! -x $demo_root/bin/wtr ]]; then
  print -u2 "demo: bin/wtr is missing; run task build first"
  return 1
fi

mkdir -p "$demo_build/bin" "$demo_build/xdg" "$demo_build/cache"
ln -sf "$demo_root/bin/wtr" "$demo_build/bin/what-to-run"

export PATH="$demo_build/bin:$PATH"
export XDG_CONFIG_HOME="$demo_build/xdg"
export XDG_CACHE_HOME="$demo_build/cache"

if ! demo_project=$(mktemp -d "${TMPDIR:-/tmp}/wtr-demo.XXXXXX"); then
  print -u2 "demo: cannot create the temporary project directory"
  return 1
fi
print -r -- "$demo_project" > "$demo_build/project-dir"
cp -R "$demo_root/docs/demo/fixture/." "$demo_project/" || return 1
cd "$demo_project" || return 1
eval "$(what-to-run init zsh)"

PROMPT='%F{blue}~/demo%f %F{magenta}❯%f '
unsetopt PROMPT_SP
PROMPT_EOL_MARK=''
precmd() { printf '\e[6 q' }
clear
