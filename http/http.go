package http

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	"github.com/meilihao/golib/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

var (
	DefaultClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).Dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

type Option struct {
	Req              *http.Request
	Client           *http.Client
	Timeout          time.Duration
	EnableDebug      bool
	BindRespHeader   bool
	RetryCount       int
	RetryWaitTime    time.Duration
	RetryMaxWaitTime time.Duration
	RetryFunc        func(c *gout.Context) error

	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   io.Reader
}

type Resp struct {
	Code   int
	Header http.Header
	Body   []byte
}

func Do(ctx context.Context, opt *Option) (*Resp, error) {
	if opt.RetryCount > 0 && opt.RetryWaitTime.Seconds() == 0 {
		return nil, errors.New("invalid retry")
	}

	var err error

	if opt.Req == nil {
		if opt.Method == "" || opt.Path == "" {
			return nil, errors.New("invalid req params")
		}

		if len(opt.Query) > 0 {
			opt.Req, err = http.NewRequest(opt.Method, opt.Path+"?"+opt.Query.Encode(), opt.Body)
		} else {
			opt.Req, err = http.NewRequest(opt.Method, opt.Path, opt.Body)
		}

		if err != nil {
			return nil, err
		}
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(opt.Req.Header))

	for k, v := range opt.Header {
		if len(v) == 0 {
			continue
		}
		opt.Req.Header.Set(k, v[0])
	}

	var client *dataflow.Gout
	if opt.Client != nil {
		client = gout.New(opt.Client)
	} else {
		client = gout.New(DefaultClient)
	}

	df := client.SetRequest(opt.Req)
	if opt.EnableDebug {
		df.Debug(true)
	}
	if opt.Timeout.Seconds() != 0 {
		df.SetTimeout(opt.Timeout)
	}

	resp := &Resp{}

	df.Code(&resp.Code).
		BindBody(&resp.Body)
	if opt.BindRespHeader {
		resp.Header = http.Header{}
		df.BindHeader(&resp.Header)
	}
	if opt.RetryCount > 0 {
		dr := df.F().Retry()
		dr = dr.Attempt(opt.RetryCount).WaitTime(opt.RetryWaitTime)

		if opt.RetryMaxWaitTime.Seconds() != 0 {
			dr = dr.MaxWaitTime(opt.RetryMaxWaitTime)
		}
		if opt.RetryFunc != nil {
			dr = dr.Func(opt.RetryFunc)
		}

		err = dr.Do()
	} else {
		err = df.Do()
	}

	if err != nil {
		log.Glog.Error("Do", zap.Error(err))
		return nil, err
	}

	return resp, nil
}
