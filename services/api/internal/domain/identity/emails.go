package identity

import (
	"fmt"

	"github.com/dealo/dealo/services/api/internal/platform/mail"
)

// emailCopy holds the translated mail text. Templates live in Go because the API sends the mail;
// the web bundle in packages/i18n is not reachable from the server.
type emailCopy struct {
	verifySubject string
	verifyBody    string
	resetSubject  string
	resetBody     string
}

var emailCopies = map[string]emailCopy{
	"uz": {
		verifySubject: "%s: elektron pochtangizni tasdiqlang",
		verifyBody: "Assalomu alaykum!\n\nElektron pochtangizni tasdiqlash uchun havolani oching:\n%s\n\n" +
			"Havola 24 soat davomida amal qiladi. Agar bu siz bo'lmasangiz, xabarni e'tiborsiz qoldiring.\n",
		resetSubject: "%s: parolni tiklash",
		resetBody: "Assalomu alaykum!\n\nYangi parol o'rnatish uchun havolani oching:\n%s\n\n" +
			"Havola 1 soat davomida amal qiladi. Agar bu so'rovni siz yubormagan bo'lsangiz, xabarni e'tiborsiz qoldiring.\n",
	},
	"ru": {
		verifySubject: "%s: подтвердите адрес электронной почты",
		verifyBody: "Здравствуйте!\n\nЧтобы подтвердить адрес электронной почты, перейдите по ссылке:\n%s\n\n" +
			"Ссылка действует 24 часа. Если вы не регистрировались, просто проигнорируйте письмо.\n",
		resetSubject: "%s: восстановление пароля",
		resetBody: "Здравствуйте!\n\nЧтобы задать новый пароль, перейдите по ссылке:\n%s\n\n" +
			"Ссылка действует 1 час. Если вы не запрашивали восстановление, проигнорируйте письмо.\n",
	},
	"en": {
		verifySubject: "%s: confirm your email address",
		verifyBody: "Hello!\n\nConfirm your email address by opening this link:\n%s\n\n" +
			"The link is valid for 24 hours. If you did not sign up, you can ignore this email.\n",
		resetSubject: "%s: reset your password",
		resetBody: "Hello!\n\nTo choose a new password, open this link:\n%s\n\n" +
			"The link is valid for 1 hour. If you did not request this, you can ignore this email.\n",
	},
}

func copyFor(locale string) emailCopy {
	if copy, ok := emailCopies[locale]; ok {
		return copy
	}
	return emailCopies["uz"]
}

func verificationMail(brand, locale, to, link string) mail.Message {
	copy := copyFor(locale)
	return mail.Message{
		To:      to,
		Subject: fmt.Sprintf(copy.verifySubject, brand),
		Body:    fmt.Sprintf(copy.verifyBody, link),
	}
}

func resetMail(brand, locale, to, link string) mail.Message {
	copy := copyFor(locale)
	return mail.Message{
		To:      to,
		Subject: fmt.Sprintf(copy.resetSubject, brand),
		Body:    fmt.Sprintf(copy.resetBody, link),
	}
}
