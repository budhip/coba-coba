package ddd_notification

type DataMessage struct {
	Operation string `json:"Operation"`
	Message   string `json:"Message"`
}

type PayloadNotification struct {
	Title        string      `json:"title"`
	Service      string      `json:"service"`
	SlackChannel string      `json:"slackChannel"`
	Data         DataMessage `json:"data"`
}

type ResponseSendMessage struct {
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RequestEmail struct {
	From        string        `json:"from"`
	FromName    string        `json:"fromName"`
	To          string        `json:"to"`
	ToName      string        `json:"toName"`
	BCC         string        `json:"bcc"`
	BCCname     string        `json:"bccName"`
	CC          []Cc          `json:"cc"`
	ReplyTo     string        `json:"replyTo"`
	Template    string        `json:"template"`
	Subject     string        `json:"subject"`
	Bucket      bool          `json:"bucket"`
	Subs        []interface{} `json:"subs"`
	Attachments []Attachment  `json:"attachments"`
}

type Attachment struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Cc struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
