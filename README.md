# trap

A CLI tool to allow interaction with STDIN.


## Usage Examples

```
# Run jq against json API
curl <API-endpoint> | ./trap jq

# Run ripgrep against huge file
cat <file> | ./trap rg --color=always
```


## TODO
- [ ] App does not terminate immediately
- [ ] Clipboard support
