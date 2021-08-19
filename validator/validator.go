// [Golang使用validator进行数据校验及自定义翻译器](https://wangyangyangisme.github.io/2020/10/20/golang-Golang%E4%BD%BF%E7%94%A8validator%E8%BF%9B%E8%A1%8C%E6%95%B0%E6%8D%AE%E6%A0%A1%E9%AA%8C%E5%8F%8A%E8%87%AA%E5%AE%9A%E4%B9%89%E7%BF%BB%E8%AF%91%E5%99%A8/)
package validator

import (
	"regexp"
	"strings"

	"github.com/meilihao/golib/v2/log"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"go.uber.org/zap"
)

type TranslationRegisterElem struct {
	Trans         ut.Translator
	RegisterFn    validator.RegisterTranslationsFunc
	TranslationFn validator.TranslationFunc
}
type TranslationRegister struct {
	Name       string
	ValidateFn validator.Func            // 自定义验证方法
	Langs      []TranslationRegisterElem // 需要Langs语言翻译器注入Name信息
}

var (
	Uni       *ut.UniversalTranslator
	mobileReg = regexp.MustCompile(`^1[3456789]\d{9}$`)
)

func InitGin() {
	Uni = ut.New(en.New(), en.New(), zh.New()) // Uni万能翻译器，保存所有的语言环境和翻译数据, 第一参数为兜底lang
	en, zh := "en", "zh"

	enTrans, _ := Uni.GetTranslator(en) // en翻译器
	zhTrans, _ := Uni.GetTranslator(zh)

	vd, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		log.Glog.Fatal("validator lib not right")
	}

	en_translations.RegisterDefaultTranslations(vd, enTrans)
	zh_translations.RegisterDefaultTranslations(vd, zhTrans) // 向zhTrans注入validator tag关联的校验信息

	tr := []TranslationRegister{
		{
			Name: "checkMobile",
			ValidateFn: func(fl validator.FieldLevel) bool {
				return mobileReg.MatchString(fl.Field().String())
			},
			Langs: []TranslationRegisterElem{
				{
					Trans: zhTrans,
					RegisterFn: func(ut ut.Translator) error {
						return ut.Add("checkMobile", Arg.Mobile.Map[zh], true) // see universal-translator for details, validator的formt占位符与golang fmt不一致
					},
					TranslationFn: func(ut ut.Translator, fe validator.FieldError) string {
						t, _ := ut.T("checkMobile", fe.Field(), fe.Field()) // fe.Field() 为上面`{N}`的实参, 参数个数必须一致

						return t
					},
				},
				{
					Trans: enTrans,
					RegisterFn: func(ut ut.Translator) error {
						return ut.Add("checkMobile", Arg.Mobile.Map[en], true) // see universal-translator for details
					},
					TranslationFn: func(ut ut.Translator, fe validator.FieldError) string {
						t, _ := ut.T("checkMobile", fe.Field(), fe.Field())

						return t
					},
				},
			},
		},
	}

	var err error
	for _, v := range tr {
		if err = vd.RegisterValidation(v.Name, v.ValidateFn); err != nil {
			log.Glog.Fatal("validator register fn", zap.Error(err))
		}

		for _, vv := range v.Langs {
			if err = vd.RegisterTranslation(v.Name, vv.Trans, vv.RegisterFn, vv.TranslationFn); err != nil {
				log.Glog.Fatal("validator register lang", zap.String("lang", vv.Trans.Locale()), zap.Error(err))
			}
		}
	}
}

func Translate(errs validator.ValidationErrors, trans ut.Translator) string {
	var errList []string
	for _, e := range errs {
		// can translate each error one at a time.
		errList = append(errList, e.Translate(trans))
	}
	return strings.Join(errList, "; ")
}
