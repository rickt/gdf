package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Quick & dirty CLI tool to see if a file exists in Google Drive
//
// Usage:
//      $ gdf <string or strings>
//
// Example Usage:
//      $ gdf Queen Elizabeth II
//      | 3.9 MB| Queen Elizabeth II (popart).png
//      $ gdf foobarbaz.png
//      No files found.
//
// Notes:
//      1) The service account will need to be granted 'drivescope' in the Admin Console. To grant this, go to
//         'Admin Console --> Security --> Advanced Settings --> Manage API Client Access'
//      2) Drive API will need to be enabled in the service account's GCP project
//      3) Service account will need to be granted Domain-wide Delegation authority.
//         See https://developers.google.com/admin-sdk/directory/v1/guides/delegation for more info.

const (
	credsfile  string = "credentials.json"
	drivescope string = "https://www.googleapis.com/auth/drive.readonly"
	pagesize   int64  = 1000
	who        string = "you@you.com"
)

// Get an authenticated http client
func httpclient(creds []byte) (*http.Client, error) {
	conf, err := google.JWTConfigFromJSON(creds, drivescope)
	if err != nil {
		return nil, err
	}
	conf.Subject = who
	return conf.Client(oauth2.NoContext), nil
}

func main() {
	// Were any filename search criteria specified as command line argument(s)?
	if len(os.Args) < 2 {
		fmt.Printf("error: no search criteria specified.\nUsage:\n\t$ gdf <string or strings>\n\n")
		return
	}

	// Load the service account JSON credentials file
	creds, err := ioutil.ReadFile(credsfile)
	if err != nil {
		log.Fatalf("error reading credentials file: %s", err)
	}

	// Get an authenticated http client
	client, err := httpclient(creds)
	if err != nil {
		log.Fatalf("error creating authenticated http client: %s", err)
	}

	// Get a Drive client
	dc, err := drive.New(client)
	if err != nil {
		log.Fatalf("error creating Google Drive client: %v", err)
	}

	// Search for files with filenames containing argv[1:]
	files, err := dc.Files.List().
		Fields("nextPageToken, files(id, mimeType, modifiedTime, name, size)").
		PageSize(pagesize).
		Q("name contains '" + strings.Join(os.Args[1:], " ") + "'").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		log.Fatalf("error reading files list from Google Drive: %v", err)
	}

	// Did any filenames match the specified search criteria?
	if len(files.Files) == 0 {
		fmt.Println("0 files found.")
		return
	}

	// Range through the list of files
	for _, i := range files.Files {
		// Get the last modified time
		lmt, err := time.Parse(time.RFC3339, i.ModifiedTime)
		if err != nil {
			log.Fatalf("error converting last modified time: %s", err)
		}
		if i.MimeType == "application/vnd.google-apps.folder" {
			fmt.Printf("|%8s |%14s | %s\n", "folder", humanize.Time(lmt), i.Name)
		} else {
			fmt.Printf("|%8s |%14s | %s\n", humanize.Bytes(uint64(i.Size)), humanize.Time(lmt), i.Name)
		}
	}

	fmt.Printf("%d files found.\n", len(files.Files))
	return
}

// EOF
