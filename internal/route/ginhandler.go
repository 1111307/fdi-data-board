package route

import (
	"github.com/gin-gonic/gin"
	"github.com/tbaehler/gin-keycloak/pkg/ginkeycloak"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"
	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gerror"
	umkc "devops.momenta.works/Momenta/FDI/_git/gotools.git/um/keycloak"

	"fdi_data_board/api"
)

func KeycloakMiddleware(cloakUrl, cloakRealm string) gin.HandlerFunc {
	return ginkeycloak.Auth(ginkeycloak.AuthCheck(), ginkeycloak.KeycloakConfig{
		Url:           cloakUrl,
		Realm:         cloakRealm,
		FullCertsPath: nil,
	})
}

func UMAuthMiddleware(cloakUrl, cloakRealm string) gin.HandlerFunc {
	return ginkeycloak.Auth(umkc.AuthCheck(), ginkeycloak.KeycloakConfig{
		Url:           cloakUrl,
		Realm:         cloakRealm,
		FullCertsPath: nil,
	})
}

// 用户组+权限RBAC
func UMUserResourcesMiddleware(funcScope, endpoint string) gin.HandlerFunc {
	return umkc.MiddlewareUserResources(funcScope, endpoint)
}

const (
	GET    = "GET"
	POST   = "POST"
	DELETE = "DELETE"
	PUT    = "PUT"
	ANY    = "ANY"
	WS     = "WS"
)

type GroupUrl struct {
	GroupAddr  string
	Urls       []Url
	Middleware []gin.HandlerFunc
}

type Url struct {
	JsonHandlerFunc  func(c *gin.Context) (api.HttpResponse, error)
	EmptyHandlerFunc func(c *gin.Context)
	Path             string
	Method           string
	Middleware       []gin.HandlerFunc
}

func MKHandler(gEngine *gin.Engine, gurl GroupUrl) {
	var route gin.IRoutes
	if gurl.GroupAddr != "" {
		route = gEngine.Group(gurl.GroupAddr, gurl.Middleware...)
	} else {
		route = gEngine
	}
	for _, url := range gurl.Urls {
		middleware := append(url.Middleware, decoratorNormal(url))
		if url.Method == GET || url.Method == WS {
			route.GET(url.Path, middleware...)
		} else if url.Method == POST {
			route.POST(url.Path, middleware...)
		} else if url.Method == PUT {
			route.PUT(url.Path, middleware...)
		} else if url.Method == DELETE {
			route.DELETE(url.Path, middleware...)
		} else if url.Method == ANY {
			route.Any(url.Path, middleware...)
		} else {
			panic("invalid http method")
		}
	}
}

func decoratorNormal(url Url) gin.HandlerFunc {
	if url.JsonHandlerFunc != nil {
		return decoratorJson(url)
	} else if url.EmptyHandlerFunc != nil {
		return decoratorEmpty(url)
	} else {
		panic("param error in decorator")
	}
}

func decoratorJson(url Url) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := url.JsonHandlerFunc(c)

		code := gerror.Code(err)
		if err != nil && gerror.Code(err) == gcode.CodeNil {
			code = gcode.CodeInternalError
		}
		c.JSON(int(code.HttpCode()), resp)
	}
}

func decoratorEmpty(url Url) gin.HandlerFunc {
	return func(c *gin.Context) {
		url.EmptyHandlerFunc(c)
	}
}
