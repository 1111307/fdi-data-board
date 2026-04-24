package log

import (
	"context"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

var DefaultCaller = mCaller(4)

func mCaller(depth int) log.Valuer {
	return func(context.Context) interface{} {
		_, file, line, _ := runtime.Caller(depth)
		idx := strings.Split(file, "/")
		return strings.Join(idx[len(idx)-2:], "/") + ":" + strconv.Itoa(line)
	}
}
