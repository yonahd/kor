package common

type Opts struct {
	DeleteFlag    bool
	NoInteractive bool
	Verbose       bool
	WebhookURL    string
	Channel       string
	Token         string
	GroupBy       string
	ShowReason    bool
}
