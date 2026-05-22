package server

import (
	"encoding/json"
	http2 "net/http"
	"strings"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"
	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gerror"

	"github.com/go-kratos/grpc-gateway/v2/protoc-gen-openapiv2/generator"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/swagger-api/openapiv2"
	"github.com/gorilla/handlers"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "fdi_data_board/idl/helloworld/v1"
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
		http.Filter(handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}),
			handlers.AllowCredentials(),
		)),
		http.ResponseEncoder(func(w http2.ResponseWriter, r *http2.Request, v interface{}) error {
			if v == nil {
				return nil
			}
			if rd, ok := v.(http.Redirector); ok {
				url, code := rd.Redirect()
				http2.Redirect(w, r, url, code)
				return nil
			}

			pbJson := func(v interface{}) ([]byte, error) {
				mOption := protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: true,
					UseEnumNumbers:  true,
				}
				if m, ok := v.(proto.Message); ok {
					return mOption.Marshal(m)
				} else {
					return json.Marshal(v)
				}
			}
			data, err := pbJson(v)
			if err != nil {
				return err
			}

			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(data)
			if err != nil {
				return err
			}
			return nil
		}),
		http.ErrorEncoder(func(w http2.ResponseWriter, r *http2.Request, err error) {
			codec, _ := http.CodecForRequest(r, "Accept")

			var code gcode.Code
			if gerror.Code(err) == gcode.CodeNil {
				code = gcode.CodeInternalError
			} else {
				code = gerror.Code(err)
			}

			log.Errorf("[http] error: code=%d err=%v", code.Code(), err)

			body, err1 := codec.Marshal(struct {
				Code    int32  `json:"code"`
				Message string `json:"message"`
			}{
				Message: "internal server error",
				Code:    int32(code.Code()),
			})
			if err1 != nil {
				w.WriteHeader(int(gcode.CodeInternalError.HttpCode()))
				return
			}
			w.Header().Set("Content-Type", strings.Join([]string{"application", codec.Name()}, "/"))
			w.WriteHeader(int(code.HttpCode()))
			_, _ = w.Write(body)
		}),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.HandlePrefix("/q/", openapiv2.NewHandler(
		openapiv2.WithGeneratorOptions(
			generator.EnumsAsInts(true),
			generator.UseJSONNamesForFields(false),
		)))
	v1.RegisterGreeterHTTPServer(srv, greeter)
	return srv
}
