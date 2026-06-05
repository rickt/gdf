package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	defaultCredentialsFile = "credentials.json"
	driveScope             = "https://www.googleapis.com/auth/drive.readonly"
	defaultPageSize        = 1000
)

type config struct {
	credentialsFile string
	subject         string
	pageSize        int64
	allDrives       bool
}

type cliError struct {
	msg string
}

func (e cliError) Error() string {
	return e.msg
}

func main() {
	log.SetFlags(0)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var usageErr cliError
		if errors.As(err, &usageErr) {
			log.Println(usageErr.Error())
			os.Exit(2)
		}
		log.Fatalf("error: %v", err)
	}
}

func run(ctx context.Context, args []string, out, errOut io.Writer) error {
	cfg, searchTerms, err := parseArgs(args)
	if err != nil {
		return err
	}
	if cfg.subject == "" {
		fmt.Fprintln(errOut, "warning: no subject set; searching as the service account instead of a delegated Google Workspace user")
	}

	creds, err := os.ReadFile(cfg.credentialsFile)
	if err != nil {
		return fmt.Errorf("read credentials file %q: %w", cfg.credentialsFile, err)
	}

	client, err := httpClient(ctx, creds, cfg.subject)
	if err != nil {
		return fmt.Errorf("create authenticated HTTP client: %w", err)
	}

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("create Google Drive client: %w", err)
	}

	files, err := findFiles(ctx, driveService, strings.Join(searchTerms, " "), cfg.pageSize, cfg.allDrives)
	if err != nil {
		return fmt.Errorf("read files list from Google Drive: %w", err)
	}

	printFiles(out, files)
	return nil
}

func parseArgs(args []string) (config, []string, error) {
	fs := flag.NewFlagSet("gdf", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := config{
		credentialsFile: envOrDefault("GDF_CREDENTIALS", defaultCredentialsFile),
		subject:         os.Getenv("GDF_SUBJECT"),
		pageSize:        envInt64OrDefault("GDF_PAGE_SIZE", defaultPageSize),
		allDrives:       envBoolOrDefault("GDF_ALL_DRIVES", false),
	}

	fs.StringVar(&cfg.credentialsFile, "credentials", cfg.credentialsFile, "service account JSON credentials file")
	fs.StringVar(&cfg.subject, "subject", cfg.subject, "Google Workspace user to impersonate for domain-wide delegation")
	fs.Int64Var(&cfg.pageSize, "page-size", cfg.pageSize, "Google Drive API page size")
	fs.BoolVar(&cfg.allDrives, "all-drives", cfg.allDrives, "search all shared drives instead of the default user corpus")

	if err := fs.Parse(args); err != nil {
		return config{}, nil, cliError{msg: usage()}
	}
	if fs.NArg() == 0 {
		return config{}, nil, cliError{msg: usage()}
	}
	if cfg.pageSize < 1 || cfg.pageSize > 1000 {
		return config{}, nil, cliError{msg: "page size must be between 1 and 1000"}
	}

	return cfg, fs.Args(), nil
}

func usage() string {
	return `Usage:
  gdf [flags] <search terms>

Flags:
  -credentials string  service account JSON credentials file (default: credentials.json or GDF_CREDENTIALS)
  -subject string      Google Workspace user to impersonate (or GDF_SUBJECT)
  -page-size int       Google Drive API page size, 1-1000 (default: 1000 or GDF_PAGE_SIZE)
  -all-drives          search all shared drives instead of the default user corpus (or GDF_ALL_DRIVES)`
}

func httpClient(ctx context.Context, creds []byte, subject string) (*http.Client, error) {
	conf, err := google.JWTConfigFromJSON(creds, driveScope)
	if err != nil {
		return nil, err
	}
	conf.Subject = subject
	return conf.Client(ctx), nil
}

func findFiles(ctx context.Context, service *drive.Service, query string, pageSize int64, allDrives bool) ([]*drive.File, error) {
	var files []*drive.File

	call := service.Files.List().
		Fields("nextPageToken, files(id, mimeType, modifiedTime, name, size)").
		PageSize(pageSize).
		Q("name contains '" + escapeDriveQueryString(query) + "' and trashed = false").
		SupportsAllDrives(true)

	if allDrives {
		call = call.Corpora("allDrives").IncludeItemsFromAllDrives(true)
	}

	err := call.Pages(ctx, func(page *drive.FileList) error {
		files = append(files, page.Files...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func printFiles(out io.Writer, files []*drive.File) {
	if len(files) == 0 {
		fmt.Fprintln(out, "0 files found.")
		return
	}

	for _, file := range files {
		fmt.Fprintf(out, "|%8s |%14s | %s\n", fileSize(file), modifiedTime(file), file.Name)
	}

	fmt.Fprintf(out, "%d files found.\n", len(files))
}

func fileSize(file *drive.File) string {
	if file.MimeType == "application/vnd.google-apps.folder" {
		return "folder"
	}
	return humanize.Bytes(uint64(file.Size))
}

func modifiedTime(file *drive.File) string {
	if file.ModifiedTime == "" {
		return "unknown"
	}

	t, err := time.Parse(time.RFC3339, file.ModifiedTime)
	if err != nil {
		return "unknown"
	}

	return humanize.Time(t)
}

func escapeDriveQueryString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt64OrDefault(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
