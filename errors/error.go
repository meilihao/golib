package errors

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/meilihao/goi18n/v2"
	"github.com/meilihao/water"
	"google.golang.org/grpc/metadata"
)

type CodeErr struct {
	Num      int32             `json:"num"`
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata"`
}

func (e CodeErr) Error() string {
	return fmt.Sprintf("code=%s, message=%s", e.Code, e.Message)
}

func I18nError0(c *water.Context, e *goi18n.Elem, args ...interface{}) error {
	l := c.Environ.GetString("i18n")
	if len(args) == 0 {
		return CodeErr{
			Code:    e.Key,
			Message: e.Map[l],
		}
	} else {
		return CodeErr{
			Code:    e.Key,
			Message: fmt.Sprintf(e.Map[l], args...),
		}
	}
}

func I18nError(c *gin.Context, e *goi18n.Elem, args ...interface{}) error {
	l := c.MustGet("i18n").(string)
	if len(args) == 0 {
		return CodeErr{
			Code:    e.Key,
			Message: e.Map[l],
		}
	} else {
		return CodeErr{
			Code:    e.Key,
			Message: fmt.Sprintf(e.Map[l], args...),
		}
	}
}

// for grpc server
func I18nError2(ctx context.Context, e *goi18n.Elem, args ...interface{}) error {
	l := GetLangFromCtx(ctx)

	if len(args) == 0 {
		return CodeErr{
			Code:    e.Key,
			Message: e.Map[l],
		}
	} else {
		return CodeErr{
			Code:    e.Key,
			Message: fmt.Sprintf(e.Map[l], args...),
		}
	}
}

func GetLangFromCtx(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "en"
	}

	if len(md["lang"]) > 0 && md["lang"][0] == "zh" {
		return "zh"
	}

	return "en"
}
