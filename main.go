package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	conffile = flag.String("c", "./conf.json", "server's configuration file")
)

func main() {
	// create the parent context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set up signals to cancel the context
	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() { <-sig; cancel() }()

	// run Main command
	if err := Main(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		fmt.Println("Terminating server...")
		os.Exit(1)
	}
}

func Main(ctx context.Context) (err error) {
	// Parse flags
	flag.Parse()

	err = checkFlags()
	if err != nil {
		return err
	}

	// Read provided configuration params
	conf, err := readConfigFile(*conffile)
	if err != nil {
		return err
	}
	err = checkParams(conf)
	if err != nil {
		return err
	}

	// Run server
	server := NewServer(conf.Addr, conf.RestAddr, conf.LogFile)
	return server.Run(ctx)
}

func checkFlags() error {
	if *conffile == "" {
		return errors.New("server's configuration file is required")
	}
	return nil
}
