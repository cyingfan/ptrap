# ptrap

A CLI tool to allow interaction with STDOUT from another app.


## Installation
`go install github.com/cyingfan/ptrap@latest`

## Usage Examples
![Demo](ptrap.gif)

```
# Run jq against json API
curl <API-endpoint> | ptrap jq

# Run ripgrep against huge file
cat <file> | ptrap rg --color=always
```


## TODO
- [ ] App does not terminate immediately
- [X] Clipboard support
