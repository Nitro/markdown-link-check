# <img src="./misc/logo/logo.png" width="25" height="25"/> Markdown Link Check
This application is used to find broken links at markdown files.

## Providers
Providers are the core of `markdown-link-check`. They enable the application to perform new kinds of checks like validating a resource that exists in Jira or even at an FTP server.

### File
The file provider checks if the links point to valid files or directories. If the link points to a file and it has an anchor it will be validated as well.

### GitHub
There is initial support for verification on private GitHub repositories. More information can be found at #7.

### Web
The web provider verifies public HTTP endpoints. The link is assumed as valid if the status code is `>=200 and <300`. The redirect status code `301` and `308` will be followed, other redirect codes are treated as an invalid link.

## Compiling
```bash
git clone git@github.com:Nitro/markdown-link-check.git
cd markdown-link-check
make build # This generate a binary at './cmd/markdown-link-check'
```

## How to use it?
```bash
➜ ./markdown-link-check --help

Usage: markdown-link-check --config=STRING <path>

Arguments:
  <path>    Path to be processed

Flags:
      --help             Show context-sensitive help.
  -c, --config=STRING    Path to the configuration file.
```

## CI
### GitHub Actions
There is a [action](https://github.com/Nitro/markdown-link-check-action) available.

### Others
It's possible to build from the source or use a [pre-built release](https://github.com/Nitro/markdown-link-check-action/releases).