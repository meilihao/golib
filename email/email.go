package email

import (
	"crypto/tls"

	"gopkg.in/gomail.v2"
)

type Config struct {
	// Host 邮箱服务器地址，如腾讯企业邮箱为smtp.exmail.qq.com
	Host string
	// Port 邮箱服务器端口，如腾讯企业邮箱为465
	Port int
	// FromEmail　发件人邮箱地址
	FromEmail string
	// FromPasswd 发件人邮箱密码（注意，这里是明文形式）
	FromPasswd         string
	InsecureSkipVerify bool
}

type Emailer struct {
	*Config
}

// toers 接收者邮件
// ccers 抄送者邮件
func (e *Emailer) Send(subject, body string, toers, ccers []string) error {
	m := gomail.NewMessage()
	// 发件人
	// 第三个参数为发件人别名，如"李大锤"，可以为空（此时则为邮箱名称）
	m.SetAddressHeader("From", e.FromEmail, "")
	// 主题
	m.SetHeader("Subject", subject)
	// 正文
	m.SetBody("text/html", body)
	// 收件人可以有多个，故用此方式
	m.SetHeader("To", toers...)
	//抄送列表
	if len(ccers) != 0 {
		m.SetHeader("Cc", toers...)
	}

	d := gomail.NewDialer(e.Host, e.Port, e.FromEmail, e.FromPasswd)
	if d.SSL && e.InsecureSkipVerify {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// 发送
	return d.DialAndSend(m)
}
