package misc

import (
	"encoding/json"

	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
)

func MarshalAny(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Glog.Error("marshal", zap.Error(err))
	}
	return data
}

func MarshalAnyString(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Glog.Error("marshal", zap.Error(err))
	}
	return string(data)
}
