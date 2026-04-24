package main

import (
	"flag"
	"github.com/go-kratos/kratos/v2/transport"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"

	"fdi_data_board/internal/conf"
	mlog "fdi_data_board/utils/log"
	"fdi_data_board/utils/transport/simple"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/config-dev.yaml", "config path, eg: -conf config-dev.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hss []*http.Server,
	ss *simple.Server) *kratos.App {
	servers := make([]transport.Server, 0)
	for _, v := range hss {
		servers = append(servers, v)
	}
	servers = append(servers, gs)
	servers = append(servers, ss)

	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			servers...,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", mlog.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"serice.version", Version,
	)
	log.SetLogger(log.NewFilter(logger, log.FilterLevel(log.LevelInfo)))

	c := config.New(
		config.WithSource(
			env.NewSource(),
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	log.Info(bc)

	app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
