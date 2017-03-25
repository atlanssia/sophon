package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/atlanssia/brynhild"
)

func main() {
	conf, err := brynhild.Load()
	if err != nil {
		log.Panicln(err)
	}

	s, err := brynhild.NewServer(conf)
	err = s.Start()
	if err != nil {
		log.Panicln(err)
	}
}
