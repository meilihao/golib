package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/meilihao/golib/v2/log"

	"go.uber.org/zap"
)

type Option struct {
	IgnoreErr   bool
	Timeout     time.Duration
	Stdin       io.Reader
	SysProcAttr *syscall.SysProcAttr
}

func CmdCombinedWithCtx(ctx context.Context, opt *Option, name string, args ...string) ([]byte, error) {
	if opt == nil {
		opt = &Option{}
	}

	now := time.Now()

	if opt.Timeout.Seconds() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LANG=POSIX")
	if opt.SysProcAttr != nil {
		cmd.SysProcAttr = opt.SysProcAttr
	}
	if opt.Stdin != nil {
		cmd.Stdin = opt.Stdin
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) && len(out) > 0 {
			err = errors.New(string(out))
		}

		if !opt.IgnoreErr {
			log.Glog.Error("exec", zap.String("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))), zap.Error(err), zap.Int64("duration", time.Since(now).Milliseconds()))
		} else {
			log.Glog.Debug("exec", zap.String("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))), zap.Error(err), zap.Int64("duration", time.Since(now).Milliseconds()))
		}
	} else {
		log.Glog.Debug("exec", zap.String("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))), zap.Int64("duration", time.Since(now).Milliseconds()))
	}

	return bytes.TrimSpace(out), err
}

func CmdCombined(opt *Option, name string, args ...string) ([]byte, error) {
	if opt == nil {
		opt = &Option{}
	}

	return CmdCombinedWithCtx(context.TODO(), opt, name, args...)
}

func CmdCombinedBashWithCtx(ctx context.Context, opt *Option, in string) ([]byte, error) {
	if opt == nil {
		opt = &Option{}
	}

	now := time.Now()

	if opt.Timeout.Seconds() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", in)
	cmd.Env = append(os.Environ(), "LANG=POSIX")
	if opt.SysProcAttr != nil {
		cmd.SysProcAttr = opt.SysProcAttr
	}
	if opt.Stdin != nil {
		cmd.Stdin = opt.Stdin
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) && len(out) > 0 {
			err = errors.New(string(out))
		}

		if !opt.IgnoreErr {
			log.Glog.Error("exec", zap.String("cmd", in), zap.Error(err), zap.Int64("duration", time.Since(now).Milliseconds()))
		} else {
			log.Glog.Debug("exec", zap.String("cmd", in), zap.Error(err), zap.Int64("duration", time.Since(now).Milliseconds()))
		}
	} else {
		log.Glog.Debug("exec", zap.String("cmd", in), zap.Int64("duration", time.Since(now).Milliseconds()))
	}

	return bytes.TrimSpace(out), err
}

func CmdCombinedBash(opt *Option, in string) ([]byte, error) {
	if opt == nil {
		opt = &Option{}
	}

	return CmdCombinedBashWithCtx(context.TODO(), opt, in)
}

func CmdCombinedWithStdin(opt *Option, reader io.Reader, name string, args ...string) ([]byte, error) {
	now := time.Now()

	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "LANG=POSIX")

	cmd.Stdin = reader
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Glog.Error("exec", zap.String("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))), zap.Error(err), zap.Int64("duration", time.Since(now).Milliseconds()))
	} else {
		log.Glog.Debug("exec", zap.String("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))), zap.Int64("duration", time.Since(now).Milliseconds()))
	}

	return bytes.TrimSpace(out), err
}

type CmdStreamControl struct {
	cmd  *exec.Cmd
	kill context.CancelFunc

	StdoutReader io.ReadCloser
}

func CmdStdoutStreamWithBash(cmdIn string) (*CmdStreamControl, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdIn)
	cmd.Env = append(os.Environ(), "LANG=POSIX")

	stderr := bytes.NewBuffer(make([]byte, 0, 1024))
	cmd.Stderr = stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		log.Glog.Error("exec stream", zap.String("cmd", cmdIn), zap.Bool("byBash", true), zap.Error(err))

		return nil, err
	}

	if err = cmd.Start(); err != nil {
		cancel()
		log.Glog.Error("exec stream start failed", zap.String("cmd", cmdIn), zap.Bool("byBash", true), zap.Error(err))

		return nil, err
	}

	log.Glog.Debug("exec stream start success", zap.String("cmd", cmdIn), zap.Bool("byBash", true))

	return &CmdStreamControl{
		cmd:          cmd,
		kill:         cancel,
		StdoutReader: stdout,
	}, nil
}

func (c *CmdStreamControl) Close() error {
	c.kill()

	waitErr := c.cmd.Wait()
	// distinguish between ExitError (which is actually a non-problem for us)
	// vs failed wait syscall (for which we give upper layers the chance to retyr)
	{
		var buf *bytes.Buffer
		var ok bool

		if buf, ok = c.cmd.Stderr.(*bytes.Buffer); ok && buf.Len() > 0 {
			log.Glog.Debug("CmdStreamControl stderr", zap.String("err", buf.String()))
		}
	}

	var exitErr *exec.ExitError
	if ee, ok := waitErr.(*exec.ExitError); ok {
		exitErr = ee
	}

	if waitErr != nil {
		return waitErr
	}

	if exitErr != nil {
		return exitErr
	}

	return nil
}
