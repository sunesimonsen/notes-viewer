# Notes Viewer

A small web viewer for my personal Markdown notes, authored with [`notes.nvim`](https://github.com/sunesimonsen/notes.nvim).

The notes use a deliberately flat file structure. Each filename contains the note's timestamp, title, and optional tags:

```text
20240229T123456--my-note-title__work_personal.md
```

- `20240229T123456` — creation timestamp (`YYYYMMDDTHHmmss`)
- `my-note-title` — title, with hyphens converted to spaces
- `work_personal` — optional underscore-separated tags

Notes are read directly from the configured directory, rendered as Markdown, and can be searched or browsed by tag.

## Development

```sh
cp .env.example .env
# Set NOTES_VIEWER_STORE_PATH to your notes directory
make dev
```

For local development, `make dev` skips email verification. To run with the normal login flow, use `make dev-with-login` and configure the SMTP settings in `.env`.

Other useful commands:

```sh
make test   # generate templates and run tests
make build  # build tmp/main
make run    # generate templates and start the viewer
```

The server listens on port `8081` by default; set `PORT` to override it.

## License

[MIT © Sune Simonsen](./LICENSE)
