package links

import "fmt"

type SlackLink interface {
	fmt.Stringer
	SlackFormat() string
}

type slackLink struct {
	url    string
	pretty string
}

func (sl slackLink) String() string {
	return sl.url
}
func (sl slackLink) SlackFormat() string {
	return "<" + sl.url + "|" + sl.pretty + ">"
}

func NewSlackLink(url string, prettyText string) SlackLink {
	return slackLink{
		url:    url,
		pretty: prettyText,
	}
}
