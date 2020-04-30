# Markdown Link Check
This application is used to find broken links on markdown files.

## Providers
Providers are the core of `markdown-link-check`. They enable the application to perform new kinds of checks like validating a resource exists in Jira or at a FTP server. There is just one provider avaiable now, the `file`.

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