package time

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInt64(t *testing.T) {
	type My struct {
		T Int64 `json：“t”`
	}

	t1 := My{
		T: Int64(time.Now().Unix()),
	}

	data, err := json.Marshal(t1)
	log.Println(string(data), err)
	assert.Nil(t, err)

	t2 := new(My)
	err = json.Unmarshal(data, t2)
	assert.Nil(t, err)

	assert.Equal(t, t1.T, t2.T)
}

func TestMytime(t *testing.T) {
	type My struct {
		T Mytime `json：“t”`
	}

	t1 := My{
		T: Mytime(time.Now()),
	}

	data, err := json.Marshal(t1)
	log.Println(string(data), err)
	assert.Nil(t, err)

	t2 := new(My)
	err = json.Unmarshal(data, t2)
	assert.Nil(t, err)

	assert.Equal(t, t1.T.String(), t2.T.String())
}
