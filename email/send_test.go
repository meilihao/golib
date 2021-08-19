package email

import (
	"log"
	"testing"
)

func TestEmail(t *testing.T) {
	toers := []string{"li@latelee.org", "latelee@163.com"} // 逗号隔开
	ccers := []string{}                                    //"readchy@163.com"

	subject := "这是主题"
	body := `这是正文<br>
            <h3>这是标题</h3>
             Hello <a href = "http://www.latelee.org">主页</a><br>`
	// 结构体赋值
	e := &Emailer{
		Config: &Config{
			Host:       "smtp.exmail.qq.com",
			Port:       465,
			FromEmail:  "cicd@latelee.org",
			FromPasswd: "1qaz@WSX",
		},
	}
	t.Logf("init email.\n")

	if err := e.Send(subject, body, toers, ccers); err != nil {
		log.Fatal("send email failed: ", err)
	}
}
