package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/umputun/secrets/backend/app/messager"
	"github.com/umputun/secrets/backend/app/server"
	"github.com/umputun/secrets/backend/app/store"
)

var opts struct {
	Engine         string        `short:"e" long:"engine" env:"ENGINE" description:"storage engine" choice:"MEMORY" choice:"BOLT" default:"MEMORY"` // nolint
	SignKey        string        `short:"k" long:"key" env:"SIGN_KEY" description:"sign key" required:"true"`
	PinSize        int           `long:"pinszie" env:"PIN_SIZE" default:"5" description:"pin size"`
	MaxExpire      time.Duration `long:"expire" env:"MAX_EXPIRE" default:"24h" description:"max lifetime"`
	MaxPinAttempts int           `long:"pinattempts" env:"PIN_ATTEMPTS" default:"3" description:"max attempts to enter pin"`
	BoltDB         string        `long:"bolt" env:"BOLT_FILE" default:"/tmp/secrets.bd" description:"boltdb file"`
	Dbg            bool          `long:"dbg" description:"debug mode"`
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
	params := messager.Params{MaxDuration: opts.MaxExpire, MaxPinAttempts: opts.MaxPinAttempts}
	srv := server.Server{
		Messager:       messager.New(dataStore, crypter, params),
		PinSize:        opts.PinSize,
		MaxExpire:      opts.MaxExpire,
		MaxPinAttempts: opts.MaxPinAttempts,
		Version:        revision,
	}
	if err := srv.Run(); err != nil {
		log.Printf("[ERROR] failed, %+v", err)
	}
}

func getEngine(engineType, boltFile string) store.Engine {
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
