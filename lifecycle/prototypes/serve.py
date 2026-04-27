#!/usr/bin/env python3
"""Simple static file server for browsing prototype directories."""

import argparse
import http.server
import os
import sys


def main():
    parser = argparse.ArgumentParser(
        description="Serve a directory of static files for local viewing."
    )
    parser.add_argument(
        "root",
        nargs="?",
        default=os.path.dirname(os.path.abspath(__file__)),
        help="Root directory to serve (default: directory containing this script)",
    )
    parser.add_argument(
        "-p", "--port",
        type=int,
        default=8888,
        help="Port to listen on (default: 8888)",
    )
    args = parser.parse_args()

    root = os.path.abspath(args.root)
    if not os.path.isdir(root):
        print(f"error: {root!r} is not a directory", file=sys.stderr)
        sys.exit(1)

    os.chdir(root)

    handler = http.server.SimpleHTTPRequestHandler

    # Silence the default per-request log lines; comment out to re-enable.
    handler.log_message = lambda *_: None

    print(f"Serving {root}")
    print(f"http://localhost:{args.port}/")
    print("Press Ctrl-C to stop.")

    with http.server.HTTPServer(("", args.port), handler) as httpd:
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\nStopped.")


if __name__ == "__main__":
    main()
