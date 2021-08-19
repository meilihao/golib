package validator

import (
	"github.com/meilihao/goi18n/v2"
)

var (
	Arg = struct {
		Mobile *goi18n.Elem
	}{

		Mobile: &goi18n.Elem{
			Key: "Arg.Mobile",
			Map: map[string]string{
				"zh": `{0}长度不等于11位或{1}格式错误!`,
				"en": `{0} length need 11 or {1} format invalid!`,
			},
		},
	}
)
