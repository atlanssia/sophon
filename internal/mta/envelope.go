package brynhild

type envelope struct {
	sender     string
	recipients []string
	data       []byte
}
