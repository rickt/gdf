# gdf #
gdf: Google Drive Find -- Search for a file in Google Drive using a service account and the (Golang) Google Drive API (v3)

Sometimes you just need to be able to verify via command line if a file exists in Google Drive. `gdf` is a very quick and dirty way to let you do that. 

## Usage ##
Basic Usage:
``` 
$ gdf <string or strings>
```
Example Usage:
```
$ gdf Queen Elizabeth II
|  3.9 MB |   4 hours ago | Queen Elizabeth II (popart).png
1 files found.
$ gdf foobarbaz.png
0 files found.
```

## Compiling ##
```
go get github.com/dustin/go-humanize
go get golang.org/x/oauth2/google
go get google.golang.org/api/drive/v3
```


