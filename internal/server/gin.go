package server

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/docs"
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/route"
)

//	@title			Backend API
//	@version		1.0
//	@description	后端 api接口定义
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@BasePath					/
//	@schemes					http https
//	@query.collection.format	multi

//	@securitydefinitions.oauth2.password	OAuth2Password
//	@tokenUrl								[[TokenUrl]]

// NewGinServer new a Gin server.
func NewGinServer(c *conf.Server, d *conf.Data, logger log.Logger,
	urls []route.GroupUrl) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	}
	if c.Gin.Network != "" {
		opts = append(opts, http.Network(c.Gin.Network))
	}
	if c.Gin.Addr != "" {
		opts = append(opts, http.Address(c.Gin.Addr))
	}
	if c.Gin.Timeout != nil {
		opts = append(opts, http.Timeout(c.Gin.Timeout.AsDuration()))
	}

	gEngine := gin.Default()
	gin.SetMode(gin.DebugMode)
	if c.GetGin().RunMode == "prd" {
		gin.SetMode(gin.ReleaseMode)
	}
	DecoratorRouter(gEngine, d, urls)

	srv := http.NewServer(opts...)
	srv.HandlePrefix("/", gEngine)
	return srv
}

func DecoratorRouter(gEngine *gin.Engine, cd *conf.Data, urls []route.GroupUrl) {
	//解决跨域问题
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowBrowserExtensions = true
	corsConfig.AddAllowHeaders("Authorization")
	gEngine.Use(cors.New(corsConfig))

	//Swagger
	docs.SwaggerInfo.SwaggerTemplate = strings.ReplaceAll(docs.SwaggerInfo.SwaggerTemplate,
		"[[TokenUrl]]", cd.GetKeycloak().GetTokenTpl())
	gEngine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	//设置健康检查
	gEngine.GET("/healthy", func(c *gin.Context) {
		c.JSON(200,
			struct {
				IsAlive bool `json:"is_alive"`
			}{IsAlive: true})
	})

	for _, gurl := range urls {
		route.MKHandler(gEngine, gurl)
	}

	gEngine.NoRoute(func(c *gin.Context) {
		c.JSON(int(gcode.CodeNotFound.HttpCode()), nil)
	})
}
