package mail

type Message struct {
	To      string
	Subject string
	Text    string
}

func otpMessage(cfg Config, to, code string) Message {
	fromName := cfg.FromName
	if fromName == "" {
		fromName = "Authx"
	}
	return Message{
		To:      to,
		Subject: fromName + " verification code",
		Text:    "Your verification code is " + code + ".\n\nIf you did not request this, you can ignore this email.\n",
	}
}

func magicLinkMessage(cfg Config, to, link string) Message {
	fromName := cfg.FromName
	if fromName == "" {
		fromName = "Authx"
	}
	return Message{
		To:      to,
		Subject: fromName + " sign-in link",
		Text:    "Sign in using this link:\n\n" + link + "\n\nIf you did not request this, you can ignore this email.\n",
	}
}
