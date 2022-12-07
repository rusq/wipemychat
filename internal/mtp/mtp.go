// Package mtp provides some functions for the gotd/td functions
package mtp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluele/gcache"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/session"
	"github.com/gotd/td/tdp"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/mattn/go-colorable"
	"github.com/rusq/dlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rusq/wipemychat/internal/mtp/authflow"
	"github.com/rusq/wipemychat/internal/mtp/bg"
)

const (
	defBatchSize  = 100
	defCacheEvict = 10 * time.Minute
	defCacheSz    = 20
)

var (
	// ErrAlreadyRunning is returned if the attempt is made to start the client,
	// while there's another instance running asynchronously.
	ErrAlreadyRunning = errors.New("already running asynchronously, stop the running instance first")
)

type Client struct {
	cl *telegram.Client

	cache   gcache.Cache
	storage storage.PeerStorage
	creds   credsStorage

	waiter     *floodwait.SimpleWaiter
	waiterStop func()

	stop bg.StopFunc

	auth         authflow.FullAuthFlow
	sendcodeOpts auth.SendCodeOptions
	telegramOpts telegram.Options
}

// Entity interface is the subset of functions that are commonly defined on most
// entities in telegram lib. It can be a user, a chat or channel, or any other
// telegram Entity.
type Entity interface {
	GetID() int64
	GetTitle() string
	TypeInfo() tdp.Type
	Zero() bool
}

type cacheKey int64

const (
	cacheDlgStorage cacheKey = iota
)

type Option func(c *Client)

func WithMTPOptions(opts telegram.Options) Option {
	return func(c *Client) {
		c.telegramOpts = opts
	}
}

// WithStorage allows to specify custom session storage.
func WithStorage(path string) Option {
	return func(c *Client) {
		c.telegramOpts.SessionStorage = &session.FileStorage{Path: path}
	}
}

// WithPeerStorage allows to specify a custom storage for peer data.
func WithPeerStorage(s storage.PeerStorage) Option {
	return func(c *Client) {
		if s == nil {
			return
		}
		c.storage = s
	}
}

// WithAuth allows to override the authorization flow
func WithAuth(flow authflow.FullAuthFlow) Option {
	return func(c *Client) {
		c.auth = flow
	}
}

func WithApiCredsFile(path string) Option {
	return func(c *Client) {
		c.creds = credsStorage{filename: path}
	}
}

func WithDebug(enable bool) Option {
	return func(c *Client) {
		if !enable {
			c.telegramOpts.Logger = nil
			return
		}
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		c.telegramOpts.Logger = zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		))
	}
}

func New(appID int, appHash string, opts ...Option) (*Client, error) {
	// Client with the default parameters
	var c = Client{
		cache:   gcache.New(defCacheSz).LFU().Expiration(defCacheEvict).Build(),
		storage: NewMemStorage(),

		auth:   authflow.TermAuth{}, // default is the terminal authentication
		waiter: floodwait.NewSimpleWaiter(),

		telegramOpts: telegram.Options{},
	}

	for _, opt := range opts {
		opt(&c)
	}

	c.telegramOpts.Middlewares = append(c.telegramOpts.Middlewares, c.waiter)
	if (appID == 0 || appHash == "") && c.creds.IsAvailable() {
		var err error
		appID, appHash, err = c.loadCredentials()
		if err != nil {
			return nil, err
		}
	}

	c.cl = telegram.NewClient(appID, appHash, c.telegramOpts)

	return &c, nil
}

func (c *Client) loadCredentials() (int, string, error) {
	var err error
	apiID, apiHash, err := c.creds.Load()
	if err == nil && apiID > 0 && apiHash != "" {
		return apiID, apiHash, nil
	}
	dlog.Debugf("warning: error loading credentials file, requesting manual input: %s", err)
	apiID, apiHash, err = c.auth.GetAPICredentials(context.Background())
	if err != nil {
		fmt.Println()
		if errors.Is(io.EOF, err) {
			return 0, "", errors.New("exit")
		}
		return 0, "", err
	}
	if err := c.creds.Save(apiID, apiHash); err != nil {
		// not a fatal error
		dlog.Debugf("failed to save credentials: %s", err)
	}
	return apiID, apiHash, nil
}

// Start starts the telegram session in goroutine
func (c *Client) Start(ctx context.Context) error {
	if c.stop != nil {
		return ErrAlreadyRunning
	}

	stop, err := bg.Connect(c.cl)
	if err != nil {
		return err
	}
	c.stop = stop

	flow := auth.NewFlow(c.auth, c.sendcodeOpts)
	if err := c.cl.Auth().IfNecessary(ctx, flow); err != nil {
		if err := c.Stop(); err != nil {
			dlog.Debugf("error stopping: %s", err)
		}
		return err
	}
	dlog.Debug("auth success")

	return nil
}

func (c *Client) Stop() error {
	if c.stop != nil {
		if c.waiterStop != nil {
			defer c.waiterStop()
		}
		return c.stop()
	}
	return nil
}

// Run runs an arbitrary telegram session.
func (c *Client) Run(ctx context.Context, fn func(context.Context, *telegram.Client) error) error {
	if c.stop != nil {
		return ErrAlreadyRunning
	}
	return c.cl.Run(ctx, func(ctx context.Context) error {
		return fn(ctx, c.cl)
	})
}
