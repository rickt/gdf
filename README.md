# gdf

`gdf` is a small command-line tool for searching Google Drive filenames with a Google service account and the Google Drive API v3.

## Build

This project now uses Go modules. Use Go 1.25.8 or newer.

```sh
go mod tidy
go build -o gdf .
```

To install it into your Go binary directory:

```sh
go install .
```

## Usage

```sh
./gdf [flags] <search terms>
```

Examples:

```sh
./gdf Queen Elizabeth II
|  3.9 MB |   4 hours ago | Queen Elizabeth II (popart).png
1 files found.

./gdf foobarbaz.png
0 files found.
```

Flags:

```text
-credentials string  service account JSON credentials file
-subject string      Google Workspace user to impersonate
-page-size int       Google Drive API page size, 1-1000
-all-drives          search all shared drives instead of the default user corpus
```

Environment variables can also provide defaults:

```sh
export GDF_CREDENTIALS=/path/to/credentials.json
export GDF_SUBJECT=you@example.com
export GDF_PAGE_SIZE=1000
export GDF_ALL_DRIVES=false
```

If `-credentials` and `GDF_CREDENTIALS` are not set, `gdf` reads `credentials.json` from the current directory.
If `-subject` and `GDF_SUBJECT` are not set, `gdf` searches as the service account itself. That usually returns far fewer files than a delegated Google Workspace user search.

## Google Setup

The service account needs:

- Google Drive API enabled in its Google Cloud project.
- Domain-wide delegation enabled if you use `-subject` or `GDF_SUBJECT`.
- The Drive readonly scope granted by your Google Workspace administrator:

```text
https://www.googleapis.com/auth/drive.readonly
```
