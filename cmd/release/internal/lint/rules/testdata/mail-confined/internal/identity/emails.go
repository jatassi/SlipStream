package identity

import (
	"net/mail"

	"github.com/wneessen/go-mail/smtp"
)

var _ mail.Address
var _ smtp.Auth
