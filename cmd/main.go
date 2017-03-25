package main

import (
	"github.com/atlanssia/sophon/internal/conf"
	"github.com/atlanssia/sophon/internal/mta"
	"log"
	"os"
	"os/signal"
)

func main() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)
	go func() {
		sig := <-sc
		log.Printf("got signal %s, I will exit.", sig)
		// TODO clean thins and exit
	}()

	conf, err := conf.Load()
	if err != nil {
		log.Panicln(err)
	}

	s, err := mta.NewServer(conf)
	err = s.Start()
	if err != nil {
		log.Panicln(err)
	}
}
