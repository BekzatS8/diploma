package mail

import "fmt"

// PasswordResetEmail builds HTML and plain-text bodies for password reset.
func PasswordResetEmail(appName, resetURL string, ttlHours int) (html, text string) {
	if appName == "" {
		appName = "BuhPro"
	}
	if ttlHours <= 0 {
		ttlHours = 1
	}

	text = fmt.Sprintf(
		"Вы запросили сброс пароля в %s.\n\nПерейдите по ссылке (действует %d ч.):\n%s\n\nЕсли вы не запрашивали сброс, проигнорируйте это письмо.",
		appName,
		ttlHours,
		resetURL,
	)

	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; line-height: 1.5; color: #111;">
  <p>Вы запросили сброс пароля в <strong>%s</strong>.</p>
  <p><a href="%s" style="display:inline-block;padding:12px 20px;background:#2563eb;color:#fff;text-decoration:none;border-radius:6px;">Сбросить пароль</a></p>
  <p style="font-size:14px;color:#555;">Ссылка действует %d ч. Если кнопка не открывается, скопируйте адрес:<br><a href="%s">%s</a></p>
  <p style="font-size:13px;color:#888;">Если вы не запрашивали сброс, просто удалите это письмо.</p>
</body>
</html>`,
		appName,
		resetURL,
		ttlHours,
		resetURL,
		resetURL,
	)

	return html, text
}
