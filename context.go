package routine

import (
	"context"
	"io"
	"os"
	"strings"
)

type _argments struct{}

//WithArgments inject into context
func WithArgments(ctx context.Context, args ...interface{}) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _argments{}, args)
	}
	return nil
}

//ArgumentsFrom extract from context
func ArgumentsFrom(ctx context.Context) ([]interface{}, bool) {
	if ctx != nil {
		argments := ctx.Value(_argments{})
		if argments != nil {
			return argments.([]interface{}), true
		}
	}
	return nil, false
}

type _stdin struct{}
type _stdout struct{}
type _stderr struct{}
type _env struct{}

//WithStdin set stdin
func WithStdin(ctx context.Context, in io.Reader) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _stdin{}, in)
	}
	return context.WithValue(context.TODO(), _stdin{}, in)
}

//StdinFrom get stdin
func StdinFrom(ctx context.Context) io.Reader {
	if ctx != nil {
		stdin := ctx.Value(_stdin{})
		if stdin != nil {
			return stdin.(io.Reader)
		}
	}
	return os.Stdin
}

//WithStdout set stdout
func WithStdout(ctx context.Context, out io.Writer) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _stdout{}, out)
	}
	return context.WithValue(context.TODO(), _stdout{}, out)
}

//StdoutFrom get stdout
func StdoutFrom(ctx context.Context) io.Writer {
	if ctx != nil {
		stdout := ctx.Value(_stdout{})
		if stdout != nil {
			return stdout.(io.Writer)
		}
	}
	return os.Stdout
}

//WithStderr set stderr
func WithStderr(ctx context.Context, out io.Writer) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _stderr{}, out)
	}
	return context.WithValue(context.TODO(), _stderr{}, out)
}

//StderrFrom get stderr
func StderrFrom(ctx context.Context) io.Writer {
	if ctx != nil {
		stderr := ctx.Value(_stderr{})
		if stderr != nil {
			return stderr.(io.Writer)
		}
	}
	return os.Stderr
}

//WithEnviron set env
func WithEnviron(ctx context.Context, key string, value string) context.Context {
	envs := EnvironFrom(ctx)
	envs = append(envs, strings.Join([]string{key, value}, "="))
	if ctx != nil {
		return context.WithValue(ctx, _env{}, envs)
	}
	return context.WithValue(context.TODO(), _env{}, envs)
}

//EnvironFrom get env
func EnvironFrom(ctx context.Context) []string {
	if ctx != nil {
		env := ctx.Value(_env{})
		if env != nil {
			return env.([]string)
		}
	}
	return os.Environ()
}
