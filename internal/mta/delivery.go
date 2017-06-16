package mta
import (
	"bufio"
	"crypto/tls"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/smtp"
	"os"
	"strings"
)

func deliver(envelope *envelope, localName string) error {

	// TODO split recipients before deliver(only outer recipients goes here)
	for _, rcp := range envelope.recipients {
		domain := rcp[strings.Index(rcp, "@")+1:]
		//mxs, err := net.LookupMX(domain)
		//if err != nil {
		//	fmt.Println(err)
		//	return err
		//}
		//for _, mx := range mxs {
		//	fmt.Println(mx)
		//}

		addr := domain + ":25"

		// local mail, deliver to local
		if strings.TrimSpace(domain) == strings.TrimSpace(localName) {
			w := bufio.NewWriter(os.Stdout)
			fmt.Fprintln(w, envelope.sender)
			fmt.Fprintln(w, envelope.recipients)
			fmt.Fprintln(w, string(envelope.data))
			w.Flush()
			return nil
		} else {
			return sendMail(addr, envelope.sender, []string{rcp}, envelope.data)
		}
	}
	return nil
}

func sendMail(addr string, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	//TODO fill localName
	if err = c.Hello("local.name.tld"); err != nil {
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
