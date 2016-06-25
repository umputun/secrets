package main

import (
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/umputun/secrets/app/crypt"
	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/rest"
	"github.com/umputun/secrets/app/store"
)

var opts struct {
	SignKey        string `short:"k" long:"key" env:"SIGN_KEY" description:"JWT sign key" required:"true"`
	PinSize        int    `long:"pinszie" env:"PIN_SIZE" default:"5" description:"pin size"`
	MaxExpSecs     int    `long:"expire" env:"MAX_EXPIRE" default:"86400" description:"max token's lifetime, in seconds"`
	MaxPinAttempts int    `long:"pinattempts" env:"PIN_ATTEMPTS" default:"3" description:"max attempts to enter pin"`
	Dbg            bool   `long:"dbg" description:"debug mode"`
}

var revision string

func main() {

	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime)
	if opts.Dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}
	log.Printf("secrets %s", revision)

	store := store.NewInMemory(time.Second*time.Duration(opts.MaxExpSecs), time.Minute*5)
	crypt := crypt.Crypt{Key: crypt.MakeSignKey(opts.SignKey, opts.PinSize)}
	params := messager.Params{MaxDuration: time.Second * time.Duration(opts.MaxExpSecs), MaxPinAttempts: opts.MaxPinAttempts}
	server := rest.Server{
		Messager: messager.New(store, crypt, params),
		PinSize:  opts.PinSize,
	}
	server.Run()
}
