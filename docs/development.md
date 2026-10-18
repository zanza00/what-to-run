# Development

Run these commands from the repository root:

```sh
make build # build bin/wtr and bin/what-to-run
make test  # run tests with the race detector and coverage
make all   # format, vet, test, and build
```

To try the widget from a checkout in the current shell:

```sh
eval "$(make bind)"             # Ctrl-S
eval "$(make bind KEY=ctrl-g)"  # choose another key
```
