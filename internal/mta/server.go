package mta

import (
	"crypto/tls"
<<<<<<< HEAD
	"log"
=======
	"github.com/atlanssia/sophon/internal/conf"
	log "github.com/sirupsen/logrus"
>>>>>>> 01932adf86bdbecbfe2ddce77f4763475ae98360
	"net"
	"net/smtp"
	"time"

	"github.com/atlanssia/sophon/internal/conf"
)

type server struct {
	option       *conf.Option
	relay        []string
	tlsConfig    *tls.Config
	sessionCount uint
}

// new server instance
func NewServer(conf *conf.Option) (*server, error) {
	if !conf.StartTLS {
		return &server{conf, nil, nil, 0}, nil
	}

	cert, err := tls.LoadX509KeyPair(conf.PublicKeyFile, conf.PrivateKeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   conf.Hostname,
	}
	return &server{conf, nil, tlsConfig, 0}, nil
}

// start a server
func (s *server) Start() error {
	listener, err := net.Listen("tcp", s.option.ListenInterface)
	if err != nil {
		log.Panic(err)
	}

	err = s.configTLS()
	if err != nil {
		// TODO disable TLS support
		log.Println(err)
	}

	var sessionID uint64
	sessionID = 0
	for {
		conn, err := listener.Accept()
		sessionID++
		s.sessionCount++ // count active sessions

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				time.Sleep(time.Second)
				log.Println(err, "- will continue")
				continue
			}
			return err
		}

		// handle a connection
		go s.handleSession(conn, sessionID)
	}
}

// TODO signal to shutdown
func (s *server) Shutdown() error {
	return nil
}

// handle a connection (in new goroutine)
func (s *server) handleSession(conn net.Conn, sessionID uint64) {
	defer conn.Close()

	session := newSession(conn, sessionID, s.option)
	defer session.close()

	session.handle()
}

func (s *server) configTLS() error {
	cert, err := tls.LoadX509KeyPair(s.option.PublicKeyFile, s.option.PrivateKeyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ServerName:   s.option.Hostname,
	}

	s.tlsConfig = tlsConfig
	return nil
}

func (s *server) sendMail(addr string, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello(s.option.Hostname); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{InsecureSkipVerify: true}

		if err = c.StartTLS(config); err != nil {
			return err
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
