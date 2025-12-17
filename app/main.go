package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/umputun/secrets/app/email"
	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/server"
	"github.com/umputun/secrets/app/store"
)

var opts struct {
	Engine         string        `short:"e" long:"engine" env:"ENGINE" description:"storage engine" choice:"MEMORY" choice:"BOLT" default:"MEMORY"` // nolint
	SignKey        string        `short:"k" long:"key" env:"SIGN_KEY" description:"sign key" required:"true"`
	PinSize        int           `long:"pinsize" env:"PIN_SIZE" default:"5" description:"pin size"`
	MaxExpire      time.Duration `long:"expire" env:"MAX_EXPIRE" default:"24h" description:"max lifetime"`
	MaxPinAttempts int           `long:"pinattempts" env:"PIN_ATTEMPTS" default:"3" description:"max attempts to enter pin"`
	BoltDB         string        `long:"bolt" env:"BOLT_FILE" default:"/tmp/secrets.bd" description:"boltdb file"`
	WebRoot        string        `long:"web" env:"WEB" description:"web ui location (dev mode, uses embedded files if not set)"`
	Branding       string        `long:"branding" env:"BRANDING" default:"Safe Secrets" description:"application branding/title"`
	Dbg            bool          `long:"dbg" description:"debug mode"`
	Domain         []string      `short:"d" long:"domain" env:"DOMAIN" env-delim:"," description:"site domain(s)" required:"true"`
	Protocol       string        `short:"p" long:"protocol" env:"PROTOCOL" description:"site protocol" choice:"http" choice:"https" default:"https" required:"true"` // nolint

	Files struct {
		Enabled bool  `long:"enabled" env:"ENABLED" description:"enable file uploads"`
		MaxSize int64 `long:"max-size" env:"MAX_SIZE" default:"1048576" description:"max file size in bytes (default 1MB)"`
	} `group:"files" namespace:"files" env-namespace:"FILES"`

	Auth struct {
		Hash       string        `long:"hash" env:"HASH" description:"bcrypt hash of password (enables auth if set)"`
		SessionTTL time.Duration `long:"session-ttl" env:"SESSION_TTL" default:"168h" description:"session lifetime"`
	} `group:"auth" namespace:"auth" env-namespace:"AUTH"`

	Email struct {
		Enabled  bool          `long:"enabled" env:"ENABLED" description:"enable email sharing"`
		Host     string        `long:"host" env:"HOST" description:"SMTP server host"`
		Port     int           `long:"port" env:"PORT" default:"587" description:"SMTP server port"`
		Username string        `long:"username" env:"USERNAME" description:"SMTP auth username"`
		Password string        `long:"password" env:"PASSWORD" description:"SMTP auth password"`
		From     string        `long:"from" env:"FROM" description:"sender address, format: 'Display Name <email>' or just 'email'"`
		TLS      bool          `long:"tls" env:"TLS" description:"use TLS (not STARTTLS)"`
		Timeout  time.Duration `long:"timeout" env:"TIMEOUT" default:"30s" description:"connection timeout"`
		Template string        `long:"template" env:"TEMPLATE" description:"path to custom email template file"`
	} `group:"email" namespace:"email" env-namespace:"EMAIL"`
}

var revision string

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	fmt.Printf("secrets %s\n", revision)

	setupLog(opts.Dbg)

	dataStore := getEngine(opts.Engine, opts.BoltDB)
	crypter := messager.Crypt{Key: messager.MakeSignKey(opts.SignKey, opts.PinSize)}
	params := messager.Params{MaxDuration: opts.MaxExpire, MaxPinAttempts: opts.MaxPinAttempts, MaxFileSize: opts.Files.MaxSize}

	if opts.Auth.Hash != "" {
		log.Printf("[INFO]  authentication enabled (session TTL: %v)", opts.Auth.SessionTTL)
	}

	// create email sender if enabled
	var emailSender email.Sender
	if opts.Email.Enabled {
		log.Printf("[INFO]  email sharing enabled (host: %s, from: %s)", opts.Email.Host, opts.Email.From)
		var emailErr error
		emailSender, emailErr = email.NewSender(email.Config{
			Enabled:  opts.Email.Enabled,
			Host:     opts.Email.Host,
			Port:     opts.Email.Port,
			Username: opts.Email.Username,
			Password: opts.Email.Password,
			From:     opts.Email.From,
			TLS:      opts.Email.TLS,
			Timeout:  opts.Email.Timeout,
			Template: opts.Email.Template,
		}, opts.Branding)
		if emailErr != nil {
			log.Fatalf("[ERROR] can't create email sender, %v", emailErr)
		}
	}

	srv, err := server.New(messager.New(dataStore, crypter, params), revision, server.Config{
		Domain:         opts.Domain,
		Protocol:       opts.Protocol,
		PinSize:        opts.PinSize,
		MaxPinAttempts: opts.MaxPinAttempts,
		MaxExpire:      opts.MaxExpire,
		WebRoot:        opts.WebRoot,
		Branding:       opts.Branding,
		EnableFiles:    opts.Files.Enabled,
		MaxFileSize:    opts.Files.MaxSize,
		AuthHash:       opts.Auth.Hash,
		SessionTTL:     opts.Auth.SessionTTL,
		EmailEnabled:   opts.Email.Enabled,
	})
	if err != nil {
		log.Fatalf("[ERROR] can't create server, %v", err)
	}
	if emailSender != nil {
		srv = srv.WithEmail(emailSender)
	}

	if err = srv.Run(context.Background()); err != nil {
		log.Printf("[ERROR] failed, %+v", err)
	}
}

func getEngine(engineType, boltFile string) messager.Engine {
	switch engineType {
	case "MEMORY":
		return store.NewInMemory(time.Minute * 5)
	case "BOLT":
		boltStore, err := store.NewBolt(boltFile, time.Minute*5)
		if err != nil {
			log.Fatalf("[ERROR] can't open db, %v", err)
		}
		return boltStore
	}
	log.Fatalf("[ERROR] unknown engine type %s", engineType)
	return nil
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
