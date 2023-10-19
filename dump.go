package golib

import (
	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func DumpToJsonString(i any) string {
	data, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(data)
}
