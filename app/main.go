package main

import (
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/umputun/secrets/app/crypt"
	"github.com/umputun/secrets/app/rest"
	"github.com/umputun/secrets/app/store"
)

var opts struct {
	SignKey    string `short:"k" long:"key" env:"SIGN_KEY" description:"JWT sign key" required:"true"`
	PinSize    int    `long:"pinszie" env:"PIN_SIZE" default:"5" description:"pin size"`
	MaxExpMins int    `long:"expire" env:"MAX_EXPIRE" default:"600" description:"max token's lifetime, in days"`
	Dbg        bool   `long:"dbg" description:"debug mode"`
}

var revision string

func main() {

	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	if opts.Dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}
	log.Printf("secrets %s", revision)

	server := rest.Server{
		Store: store.NewInMemory(crypt.Crypt{Key: opts.SignKey}, time.Minute*time.Duration(opts.MaxExpMins)),
	}
	server.Run()
}
