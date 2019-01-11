package sender

import (
	"os/exec"
	"net/smtp"
	"net/mail"
	"./email"
	"fmt"
	"os"
)

func tarReport() error {
	cmd := exec.Command("tar", "-zcf", "report.tar.gz", "report/")
	err := cmd.Run()
	return err
}

func SendReport(address, name string) error {
	const SMTP_USERNAME = "PRIVATE DATA"
	const SMTP_PASSWORD = "PRIVATE DATA"
	const SMTP_SERVER = "PRIVATE DATA"
	const PATTERN = `Уважаемый %s,
Спасибо за использование LJIR Online. Ваша заявка была обработана в той или иной степени, и разработчик выражает искреннюю надежду, что в той, а не иной.
Даже если LJIR умудрился вам что-то попортить, он  ̶п̶о̶п̶р̶о̶с̶и̶т̶ ̶п̶р̶о̶щ̶е̶н̶и̶я̶ делал резервные копии постов, так что восстановить их не составит труда. Конечно, если внезапно копии не окажутся битыми, хехехе.
К данному письму прилагается архив, в котором вы найдёте копии постов в json-формате и лог обработки вашей заявки, в котором содержится вся информация по заменам изображений во всех постах.
Если по какому-то ужасному стечению обстоятельств у вас вместо блога КРОВЬ КИШКИ РАСПИДОРАСИЛО - напишите на адрес разработчика artem@bigdan.in, и мы решим вашу проблему.
В случае успешного проведения обработки, ваша благодарность может быть выражена в денежном эквиваленте, например переводом на карточку monobank 5375414105767932. Закиньте сколько не жалко. А можете не закидывать.

С уважением,
func SendReport(address, name string)`

	err := tarReport()
	if err != nil {
		return err
	}
	defer os.Remove("report.tar.gz")

	text := fmt.Sprintf(PATTERN, name)

	auth := smtp.PlainAuth(
		"",
		SMTP_USERNAME,
		SMTP_PASSWORD,
		SMTP_SERVER,
	)

	msg := email.NewMessage("Отчёт об обработке", text)
	msg.From = mail.Address{Name: "LJIR Online", Address: "report@ljir.devnullinc.pp.ua"}
	msg.To = []string{address}
	err = msg.Attach("report.tar.gz")
	if err != nil {
		return err
	}
	err = email.Send("in-v3.mailjet.com:25", auth, msg)
	return err
}
