# ptrap

A CLI tool to allow interaction with STDOUT from another app.


## Installation
`go install github.com/cyingfan/ptrap@latest`

## Features
- Build interactive pipelines of commands and run them over piped stdin. Each stage has a base command and optional live-editable arguments. 
- Add stages on the fly and reorder focus between stages to edit arguments; the pipeline automatically re-runs with a short debounce.
- View the combined output in a scrollable pane with a live scroll percentage indicator.
- Copy the current output to the clipboard with a single key.
- Copy the entire pipeline string (e.g., `jq . | rg foo`) to the clipboard.
- Modal prompt to quickly add new commands; close with Enter or Esc.
- Graceful cancellation of in-flight executions when the pipeline changes.

## Usage Examples
![Demo](ptrap.gif)

Keyboard shortcuts:
- Enter: copy current output to clipboard
- Ctrl+Y: copy the pipeline string to clipboard (e.g., `jq . | rg foo`)
- | (pipe): open the "Add command" modal
- Ctrl+[ : focus previous stage
- Ctrl+] : focus next stage
- Ctrl+D: delete current stage
- Ctrl+C: quit
- In modal: Enter to add/confirm, Esc to cancel
- Scrolling: use standard keys provided by the viewport (e.g., Up/Down, PageUp/PageDown)

```
# Run jq against json API
curl <API-endpoint> | ptrap jq

# Run ripgrep against huge file
cat <file> | ptrap rg --color=always
```


## TODO
- [X] App does not terminate immediately
- [X] Clipboard support
