package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gotd/td/telegram"
	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
	"github.com/rusq/osenv/v2"
	"github.com/rusq/tracer"
	"github.com/schollz/progressbar/v3"

	mtp "github.com/rusq/mtpwrap"
	"github.com/rusq/mtpwrap/authflow"

	"github.com/rusq/wipemychat/internal/session"
	"github.com/rusq/wipemychat/internal/tui"
	"github.com/rusq/wipemychat/internal/waipu"
)

const cacheDirName = "tgmsg_revoker"

const AppName = "Wipe My Chat for Telegram"

var (
	version = "dev"
	date    = "just now"
	commit  = ""

	versionSig = fmt.Sprintf("%s %s (built %s)", AppName, version, date)
)

var _ = godotenv.Load() // load environment variables from .env, if present

type Params struct {
	CacheDirName string

	ApiID   int
	ApiHash string
	Phone   string

	// Reset requests removal of the session and API credentials files.
	Reset bool
	// Logout requests removal of the session file.
	Logout bool

	List  bool
	Batch chatIDs

	Version bool
	Verbose bool
	Trace   string

	cacheDir string
}

func main() {
	p, err := parseCmdLine()
	if err != nil {
		dlog.Fatal(err)
	}
	if p.Version {
		ver(os.Stdout)
		return
	}

	dlog.SetDebug(p.Verbose)

	if err := p.initCacheDir(cacheDirName); err != nil {
		dlog.Fatalf("failed to create cache directory: %s", err)
	}

	dlog.SetDebug(p.Verbose)

	if err := run(context.Background(), p); err != nil {
		dlog.Fatal(err)
	}
}

type chatIDs []int64

func (c *chatIDs) Set(val string) error {
	ss := strings.Split(val, ",")
	ids := make([]int64, 0, len(ss))

	for _, sID := range ss {
		id, err := strconv.ParseInt(sID, 10, 64)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	*c = ids
	return nil
}

func (c *chatIDs) String() string {
	return fmt.Sprint([]int64(*c))
}

func parseCmdLine() (Params, error) {
	p := Params{CacheDirName: cacheDirName}
	{
		// auth options
		flag.IntVar(&p.ApiID, "api-id", osenv.Secret("APP_ID", 0), "Telegram API ID")
		flag.StringVar(&p.ApiHash, "api-token", osenv.Secret("APP_HASH", ""), "Telegram API token")
		flag.StringVar(&p.Phone, "phone", osenv.Value("PHONE", ""), "phone `number` in international format for authentication (optional)")

		// reset options
		flag.BoolVar(&p.Reset, "reset", false, "reset authentication (logout and remove credentials)")
		flag.BoolVar(&p.Logout, "logout", false, "logout current account, use this to login as another user with the same API ID")

		// batch mode
		flag.BoolVar(&p.List, "list", false, "list channels and their IDs")
		flag.Var(&p.Batch, "wipe", "batch mode, specify comma separated chat IDs on the command line")

		// sundry
		flag.BoolVar(&p.Version, "v", false, "print version and exit")
		flag.BoolVar(&p.Verbose, "verbose", osenv.Value("DEBUG", "") != "", "verbose output")
		flag.StringVar(&p.Trace, "trace", osenv.Value("TRACE_FILE", ""), "trace `filename`")

		flag.Parse()
	}
	return p, nil
}

func (p *Params) initCacheDir(appName string) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheDir, appName)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	p.cacheDir = cacheDir
	return nil
}

func unlink(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func run(ctx context.Context, p Params) error {
	if p.Trace != "" {
		tr := tracer.New(p.Trace)
		if err := tr.Start(); err != nil {
			return err
		}
		defer tr.End()
	}

	header(os.Stdout)

	sessfile := filepath.Join(p.cacheDir, "session.dat")
	if migrated, err := migratev120(sessfile); err != nil {
		return err
	} else if migrated {
		fmt.Fprintln(os.Stdout, "session file was migrated to new format")
	}

	sessStorage := session.FileStorage{Path: filepath.Join(p.cacheDir, "session.dat")}
	apiCredsFile := filepath.Join(p.cacheDir, "telegram.dat")
	if p.Logout {
		if err := unlink(sessStorage.Path); err != nil {
			return err
		} else {
			fmt.Fprintln(os.Stdout, "you were logged out")
		}
		os.Exit(0)
	}
	if p.Reset {
		for _, file := range []string{sessStorage.Path, apiCredsFile} {
			if err := unlink(file); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("error deleting %s: %w", file, err)
			}
		}
		fmt.Fprintln(os.Stdout, "logged out and credentials removed")
		os.Exit(0)
	}

	opts := telegram.Options{
		SessionStorage: &sessStorage,
	}

	cl, err := mtp.New(ctx, p.ApiID, p.ApiHash,
		mtp.WithAuth(authflow.NewTermAuth(p.Phone)),
		mtp.WithApiCredsFile(apiCredsFile),
		mtp.WithMTPOptions(opts),
		mtp.WithDebug(p.Verbose),
	)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	dlog.Println("Connecting to telegram . . .")
	if err := cl.Start(ctx); err != nil {
		return err
	}
	defer func() {
		if err := cl.Stop(); err != nil {
			dlog.Printf("stop error: %s", err)
		}
	}()

	if p.List {
		return waipu.List(ctx, os.Stdout, cl)
	} else if len(p.Batch) > 0 {
		return waipu.Batch(ctx, cl, []int64(p.Batch))
	} else {
		// run UI
		done, finished := fakeProgress("Getting chats . . .", 0)
		chats, err := cl.GetChats(ctx)
		close(done)
		<-finished
		if err != nil {
			return err
		}
		sort.Slice(chats, func(i, j int) bool {
			return chats[i].GetTitle() < chats[j].GetTitle()
		})
		dlog.Printf("got %d chats", len(chats))

		tva := tui.New(ctx, cl)
		if err := tva.Run(ctx, chats); err != nil {
			return err
		}
	}

	return nil
}

// fakeProgress starts a fake spinner and returns a channel that must be closed
// once the operation completes. interval is interval between iterations. If not
// set, will default to 50ms.
func fakeProgress(title string, interval time.Duration) (chan<- struct{}, <-chan struct{}) {
	if interval == 0 {
		interval = 50 * time.Millisecond
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		bar := progressbar.NewOptions(
			-1,
			progressbar.OptionSetDescription(title),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionSpinnerType(9),
		)
		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-done:
				bar.Finish()
				fmt.Println()
				close(finished)
				return
			case <-t.C:
				bar.Add(1)
			}
		}
	}()
	return done, finished
}

func header(w io.Writer) {
	fmt.Fprintf(w,
		"%s\n%s\n%s\n", versionSig, strings.Repeat("-", len(versionSig)),
		color.New(color.Italic).Sprint("In loving memory of V. Gorban, 1967-2022."),
	)
	fmt.Fprintln(w)
}

func ver(w io.Writer) {
	header(w)
	if commit != "" {
		fmt.Fprintf(w, "commit: %s\n", commit)
	}
}
