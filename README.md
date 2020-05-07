# Markdown Link Check
This application is used to find broken links on markdown files.

## Providers
Providers are the core of `markdown-link-check`. They enable the application to perform new kinds of checks like validating a resource exists in Jira or at a FTP server. There is just one provider avaiable now, the `file`.

### File
The file provider checks if the links points to valid files or directories. If it's a file and the link has an anchor it will be validated.

### GitHub
There is a initial support for verification on private GitHub repositories. More information can be found at #7.

### Web
The web provider verify public HTTP endpoints. The link is assumed as valid if the status code is `>=200 and <300`. The redirect status code `301` and `308` will be followed, other redirect codes are treated as an error.

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