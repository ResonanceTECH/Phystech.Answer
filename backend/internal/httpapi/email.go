package httpapi

import (
	"fmt"
	"html"
	"time"
)

// resetEmailContent — текстовая и HTML-версии письма восстановления.
// Стиль повторяет продукт: белая карточка, фирменная фиолетовая кнопка.
func resetEmailContent(link string, ttl time.Duration) (text, htmlBody string) {
	minutes := int(ttl.Minutes())
	safeLink := html.EscapeString(link)

	text = fmt.Sprintf(`Здравствуйте!

Мы получили запрос на восстановление пароля от вашего аккаунта Физтех.Ответ.

Чтобы создать новый пароль, перейдите по ссылке:
%s

Ссылка действует %d минут.

Если вы не запрашивали восстановление, просто проигнорируйте это письмо.`,
		link, minutes)

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<body style="margin:0;padding:0;background:#f4f2fb;font-family:'Segoe UI',Arial,sans-serif;">
  <div style="max-width:560px;margin:0 auto;padding:32px 16px;">
    <div style="text-align:center;padding-bottom:24px;">
      <span style="font-size:20px;font-weight:800;color:#2a1656;">Физтех<span style="color:#8b5cf6;">.</span>Ответ</span>
    </div>
    <div style="background:#ffffff;border-radius:16px;padding:32px;border:1px solid #e6e0f5;">
      <h1 style="margin:0 0 16px;font-size:22px;color:#17233d;">Восстановление пароля</h1>
      <p style="margin:0 0 24px;font-size:15px;line-height:1.6;color:#4a4763;">
        Мы получили запрос на восстановление пароля от вашего аккаунта.
        Чтобы создать новый пароль, нажмите кнопку ниже.
      </p>
      <div style="text-align:center;padding-bottom:24px;">
        <a href="%s" style="display:inline-block;background:#8b5cf6;color:#ffffff;text-decoration:none;font-size:15px;font-weight:700;padding:14px 32px;border-radius:999px;">Восстановить пароль</a>
      </div>
      <p style="margin:0 0 8px;font-size:13px;line-height:1.6;color:#8a86a3;">
        Ссылка действует %d минут. Если кнопка не работает, скопируйте адрес в браузер:
      </p>
      <p style="margin:0;font-size:12px;word-break:break-all;color:#8b5cf6;">%s</p>
    </div>
    <p style="text-align:center;font-size:12px;color:#8a86a3;padding-top:24px;">
      Если вы не запрашивали восстановление, просто проигнорируйте это письмо.
    </p>
  </div>
</body>
</html>`, safeLink, minutes, safeLink)

	return text, htmlBody
}
