package mail

import (
	gomail "github.com/wneessen/go-mail"
	"github.com/wneessen/go-mail/smtp"
)

var _ gomail.Msg
var _ smtp.Auth
