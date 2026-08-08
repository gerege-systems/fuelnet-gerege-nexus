/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The message a recipient actually reads.
 */

package emailverify

import (
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/mailer"
)

// The mail is plain text on purpose.
//
// It carries exactly one thing — a URL — and every mail client makes a URL
// clickable on its own. An HTML body would add a template to escape, a second
// rendering path to keep in step, and a reason for a spam filter to look
// harder, in exchange for a button.
//
// Wording exists in all seven platform languages. Server-owned text is the half
// of the interface the client cannot translate for itself, and this half is
// read outside the product entirely — by somebody who may never have seen it.
type wording struct {
	subject string
	intro   string
	expiry  string // carries {deadline}
	ignore  string
}

var wordings = map[string]wording{
	"mn": {
		subject: "И-мэйл хаягаа баталгаажуулна уу",
		intro:   "И-мэйл хаягаа баталгаажуулахын тулд доорх холбоос дээр дарна уу:",
		expiry:  "Холбоос {deadline} хүртэл хүчинтэй бөгөөд ганц удаа ажиллана.",
		ignore:  "Хэрэв та энэ хүсэлтийг илгээгээгүй бол энэ захидлыг үл тоомсорлоно уу.",
	},
	"en": {
		subject: "Confirm your email address",
		intro:   "Follow this link to confirm your email address:",
		expiry:  "The link works once and stops working after {deadline}.",
		ignore:  "If you did not ask for this, you can ignore this message.",
	},
	"ar": {
		subject: "أكّد عنوان بريدك الإلكتروني",
		intro:   "اتبع هذا الرابط لتأكيد عنوان بريدك الإلكتروني:",
		expiry:  "يعمل الرابط مرة واحدة ويتوقف عن العمل بعد {deadline}.",
		ignore:  "إذا لم تطلب ذلك، يمكنك تجاهل هذه الرسالة.",
	},
	"zh": {
		subject: "确认您的电子邮件地址",
		intro:   "请点击以下链接确认您的电子邮件地址：",
		expiry:  "该链接仅可使用一次，并在 {deadline} 后失效。",
		ignore:  "如果这不是您本人的请求，请忽略此邮件。",
	},
	"fr": {
		subject: "Confirmez votre adresse e-mail",
		intro:   "Suivez ce lien pour confirmer votre adresse e-mail :",
		expiry:  "Le lien fonctionne une seule fois et expire après le {deadline}.",
		ignore:  "Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.",
	},
	"ru": {
		subject: "Подтвердите адрес электронной почты",
		intro:   "Перейдите по ссылке, чтобы подтвердить адрес электронной почты:",
		expiry:  "Ссылка сработает один раз и перестанет действовать после {deadline}.",
		ignore:  "Если вы этого не запрашивали, просто проигнорируйте это письмо.",
	},
	"es": {
		subject: "Confirme su dirección de correo electrónico",
		intro:   "Siga este enlace para confirmar su dirección de correo electrónico:",
		expiry:  "El enlace funciona una sola vez y caduca después del {deadline}.",
		ignore:  "Si no solicitó esto, puede ignorar este mensaje.",
	},
}

// composeMessage builds the verification mail. The deadline is written in UTC
// with the zone named: the recipient is outside the product and has no session
// to carry a timezone, and an unqualified local time is the one that gets read
// wrong.
func composeMessage(to, link string, expiresAt time.Time, locale string) mailer.EmailMessage {
	text, ok := wordings[strings.ToLower(strings.TrimSpace(locale))]
	if !ok {
		text = wordings["mn"]
	}
	deadline := expiresAt.UTC().Format("2006-01-02 15:04 MST")
	body := strings.Join([]string{
		text.intro,
		"",
		link,
		"",
		strings.ReplaceAll(text.expiry, "{deadline}", deadline),
		text.ignore,
		"",
		"— Gerege Nexus",
	}, "\r\n")

	return mailer.EmailMessage{To: to, Subject: text.subject, Body: body}
}
