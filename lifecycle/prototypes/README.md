# Prototypes

Static prototype files for Kaos Control UI concepts.

## Serving locally

Use `serve.py` to browse prototypes in a browser without needing a build step.
It requires only the Python 3 standard library — no `pip install` needed.

```
python3 serve.py [ROOT] [-p PORT]
```

| Argument | Default | Description |
|---|---|---|
| `ROOT` | directory containing `serve.py` | Directory to serve |
| `-p`, `--port` | `8888` | Port to listen on |

### Examples

Serve this directory on the default port:
```sh
python3 serve.py
# → http://localhost:8888/
```

Serve a specific subdirectory on a custom port:
```sh
python3 serve.py "Kaos Control" -p 9000
# → http://localhost:9000/
```

Serve from anywhere by passing the full path:
```sh
python3 serve.py /path/to/kaos-control/lifecycle/prototypes -p 8080
```

### Navigating prototypes

- `Kaos Control/` — HTML/JSX prototype for the graph and layout views.
  Open `Kaos Control/Kaos Control.html` directly in a browser, or serve
  the parent directory and navigate to it.

## Adding a new prototype

Drop a folder here with a self-contained `index.html` (or named `.html` file).
No framework or bundler is required — the server serves files as-is.
