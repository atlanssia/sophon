package mta

type envelope struct {
	sender     string
	recipients []string
	data       []byte
}
