package mta

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/textproto"
	"strings"
	"time"
	"github.com/atlanssia/sophon/internal/conf"
)

const (
	// The client has connected, and is awaiting our first response
	clientGreeting = iota
	// We have responded to the client's connection and are awaiting a command
	clientCmd
	// We have received the sender and recipient information
	clientData
	// We have agreed with the client to secure the connection over TLS
	clientStartTLS
	// Server will shutdown, client to shutdown on next command turn
	clientShutdown
)

type session struct {
	sessionId  uint64
	connection net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	scanner    *bufio.Scanner
	option     *conf.Option
	envelope   *envelope
}

type command struct {
	line   string
	action string
	fields []string
	params []string
}

// new session instance
func newSession(conn net.Conn, sessionId uint64, option *conf.Option) *session {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(conn)
	instance := &session{sessionId, conn, reader, writer, scanner, option, nil}
	return instance
}

// TODO
func (session *session) upgradeToTLS(tlsConfig *tls.Config) error {
	return nil
}

// send response to client
func (session *session) sendResponse(code int, msg string) error {
	fmt.Fprintf(session.writer, "%d %s\r\n", code, msg)
	session.writer.Flush()
	return nil
}

func (session *session) close() {
	session.writer.Flush()
	time.Sleep(200 * time.Millisecond)
	session.connection.Close()
}

func (session *session) parseLine(line string) (cmd command) {

	cmd.line = line
	cmd.fields = strings.Fields(line)

	if len(cmd.fields) > 0 {
		cmd.action = strings.ToUpper(cmd.fields[0])
		if len(cmd.fields) > 1 {
			cmd.params = strings.Split(cmd.fields[1], ":")
		}
	}

	return
}

func (session *session) parseAddress(src string) (string, error) {

	if len(src) == 0 || src[0] != '<' || src[len(src)-1] != '>' {
		return "", fmt.Errorf("Ill-formatted e-mail address: %s", src)
	}

	if strings.Count(src, "@") > 1 {
		return "", fmt.Errorf("Ill-formatted e-mail address: %s", src)
	}

	return src[1 : len(src)-1], nil
}

func (session *session) handle() {
	// the welcoming message
	greeting := fmt.Sprintf("%s - Session id: %d, Time: %s", session.option.Welcoming, session.sessionId, time.Now().Format(time.RFC3339))

	session.sendResponse(220, greeting)
	log.Debugln("sent greeting...")
	for {
		for session.scanner.Scan() {
			line := session.scanner.Text()
			log.Debugf("line scanned: %s", line)
			session.handleLine(line)
		}

		err := session.scanner.Err()
		if err == bufio.ErrTooLong {
			session.sendResponse(500, "Line too long")

			// Advance reader to the next newline
			session.reader.ReadString('\n')
			session.scanner = bufio.NewScanner(session.reader)
			session.reset()
			continue
		}
		break
	}
}

func (session *session) handleLine(line string) {
	cmd := session.parseLine(line)
	switch cmd.action {

	case "HELO":
		session.handleHELO(cmd)
		return

	case "EHLO":
		session.handleEHLO(cmd)
		return

	case "MAIL":
		session.handleMAIL(cmd)
		return

	case "RCPT":
		session.handleRCPT(cmd)
		return

	case "STARTTLS":
		session.handleSTARTTLS(cmd)
		return

	case "DATA":
		session.handleDATA(cmd)
		return

	case "RSET":
		session.handleRSET(cmd)
		return

	case "NOOP":
		session.handleNOOP(cmd)
		return

	case "QUIT":
		session.handleQUIT(cmd)
		return

	case "AUTH":
		session.handleAUTH(cmd)
		return

	case "XCLIENT":
		session.handleXCLIENT(cmd)
		return

	}

	session.sendResponse(502, "Unsupported command.")

}

func (session *session) handleHELO(cmd command) {
	if len(cmd.fields) < 2 {
		session.sendResponse(502, "Missing parameters")
		return
	}

	session.sendResponse(250, "Go ahead")
}

func (session *session) handleEHLO(cmd command) {
	if len(cmd.fields) < 2 {
		session.sendResponse(502, "Missing parameters")
		return
	}

	extensions := []string{}
	extensions = append(extensions, "250-PIPELINING",
		"250-SIZE 10240000",
		"250-ENHANCEDSTATUSCODES",
		"250-8BITMIME",
		"250 DSN")
	for _, ext := range extensions {
		fmt.Fprintf(session.writer, "%s\r\n", ext)
	}
	session.writer.Flush()
}

func (session *session) handleMAIL(cmd command) {
	if len(cmd.params) != 2 || strings.ToUpper(cmd.params[0]) != "FROM" {
		session.sendResponse(502, "Invalid syntax.")
		return
	}

	//ttt

	if session.envelope != nil {
		session.sendResponse(502, "Duplicate MAIL")
		return
	}

	addr, err := session.parseAddress(cmd.params[1])

	if err != nil {
		session.sendResponse(502, "Ill-formatted e-mail address")
		return
	}

	// TODO sender valid?

	session.envelope = &envelope{
		sender: addr,
	}

	session.sendResponse(250, "Go ahead")

	return

}

func (session *session) handleRCPT(cmd command) {
	if len(cmd.params) != 2 || strings.ToUpper(cmd.params[0]) != "TO" {
		session.sendResponse(502, "Invalid syntax.")
		return
	}

	if session.envelope == nil {
		session.sendResponse(502, "Missing MAIL FROM command.")
		return
	}

	//TODO check if max recipients

	addr, err := session.parseAddress(cmd.params[1])

	if err != nil {
		session.sendResponse(502, "Ill-formatted e-mail address")
		return
	}

	session.envelope.recipients = append(session.envelope.recipients, addr)

	session.sendResponse(250, "Go ahead")

	return
}

// TODO
func (session *session) handleSTARTTLS(cmd command) {

}

func (session *session) handleDATA(cmd command) {

	if session.envelope == nil || len(session.envelope.recipients) == 0 {
		session.sendResponse(502, "Missing RCPT TO command.")
		return
	}

	log.Debugln("data go")
	session.sendResponse(354, "Go ahead. End your data with <CR><LF>.<CR><LF>")
	//session.conn.SetDeadline(time.Now().Add(session.server.DataTimeout))

	data := &bytes.Buffer{}
	reader := textproto.NewReader(session.reader).DotReader()

	_, err := io.CopyN(data, reader, int64(session.option.MaxMessageSize))

	if err == io.EOF {

		session.envelope.data = data.Bytes()

		deliver(session.envelope, session.option.Hostname)

		// response 250 if deliver finished
		session.sendResponse(250, "Thank you.")

		session.reset()
	}

	if err != nil {
		// Network error, ignore
		return
	}

	// Discard the rest and report an error.
	_, err = io.Copy(ioutil.Discard, reader)

	if err != nil {
		// Network error, ignore
		return
	}

	session.sendResponse(552, fmt.Sprintf(
		"Message exceeded max message size of %d bytes",
		session.option.MaxMessageSize,
	))

	session.reset()

	return
}

func (session *session) handleNOOP(cmd command) {
	session.sendResponse(250, "Go ahead")
	return
}

func (session *session) handleRSET(cmd command) {
	session.reset()
	session.sendResponse(250, "Go ahead")
	return
}

func (session *session) handleQUIT(cmd command) {
	session.sendResponse(221, "OK, bye")
	session.close()
	return
}

func (session *session) reset() {
	session.envelope = nil
}

func (session *session) reject() {
	session.sendResponse(421, "Too busy. Try again later.")
	session.close()
}
