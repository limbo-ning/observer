package main

import (
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/config"
	_ "obsessiontech/common/context"
	myHttp "obsessiontech/common/http"
	"obsessiontech/common/random"
	"obsessiontech/common/random/serial"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/role"
	"obsessiontech/environment/user/auth"
	"obsessiontech/environment/websocket"

	"github.com/gin-gonic/gin"
)

const PROJECT_C = "c"

var Config struct {
	InternalSecret     string
	CookieAuthUserName string
	CookieDomain       string
}
var server *gin.Engine
var router, internal, sites, authorized, projectc *gin.RouterGroup

func init() {
	config.GetConfig("config.yaml", &Config)
}

func main() {
	server = myHttp.GetObEngine()
	prefix := myHttp.GetPrefix()
	prefix = strings.Replace(prefix, "v1", ":serverVersion", 1)

	var logConfig gin.LoggerConfig
	logConfig.SkipPaths = []string{fmt.Sprintf("/%s/internal/ping", prefix)}
	server.Use(gin.LoggerWithConfig(logConfig))

	router = server.Group(prefix)

	internal = router.Group("internal", func(c *gin.Context) {
		if c.Query("secret") != Config.InternalSecret {
			c.AbortWithStatus(400)
		} else {
			c.Next()

			json, exists := c.Get("json")
			if exists && json != nil {
				c.PureJSON(200, json)
			}
		}
	})

	internal.HEAD("ping", func(c *gin.Context) {
		c.Status(200)
	})

	sites = router.Group("/", func(c *gin.Context) {
		if site, exists := c.GetQuery("csite"); !exists {
			c.AbortWithStatus(400)
		} else if strings.TrimSpace(site) == "" {
			c.AbortWithStatus(400)
		} else {

			c.Set("requestID", fmt.Sprintf("%d-%s-%s", serial.Config.MachineCode, random.GenerateNonce(4), random.GenerateNonce(4)))
			c.Set("csite", site)

			log.Println("request accept: ", site, c.GetString("requestID"))

			token := c.GetHeader("Authorization")
			if token == "" {
				token, _ = c.Cookie(site + "-" + Config.CookieAuthUserName)
			}

			referer := c.Request.Referer()

			userID, token := auth.IsLogined(site, c.ClientIP(), token, referer)

			c.Set("uid", userID)
			c.Set("session", token)

			if site == PROJECT_C && c.Query("siteID") != "" {
				site = c.Query("siteID")
			}
			c.Set("site", site)

			clientAgent := c.Query("ca")
			c.Set("clientAgent", clientAgent)

			c.Next()

			session := c.GetString("session")
			if session != "" {
				c.Header("Session", session)
			}

			json, exists := c.Get("json")
			if exists && json != nil {
				if session != "" {
					json.(map[string]interface{})["session"] = session
				}
				c.PureJSON(200, json)
			}

			log.Println("request done: ", site, c.GetString("requestID"))
		}
	})

	sites.GET("websocket", func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.GetHeader("X-Cookie-" + c.GetString("csite") + "-" + Config.CookieAuthUserName)
		}
		websocket.Handle(c.Writer, c.Request, c.GetString("site"), c.ClientIP(), token)
	})

	authorized = sites.Group("/", func(c *gin.Context) {
		if c.GetInt("uid") > 0 {
			c.Next()
		} else {
			c.AbortWithStatusJSON(200, map[string]interface{}{"retCode": 1, "retMsg": "请登录"})
		}
	})

	//user of projectc
	projectc = authorized.Group("/", func(c *gin.Context) {
		if c.GetString("csite") == PROJECT_C {
			c.Next()
		} else {
			c.AbortWithStatus(403)
		}
	})

	loadCadmin()
	loadClientAgent()
	loadCategory()
	loadEnvironment()
	loadEvent()
	loadLogging()
	loadMission()
	loadPage()
	loadPush()
	loadRole()
	loadPeripheral()
	loadResource()
	loadSite()
	loadSurveillance()
	loadUser()
	loadVehicle()
	loadWechat()
	loadMigrate()
	server.Run(":" + myHttp.GetPort())
}

func doAuth(c *gin.Context, mandatory bool, moduleID string, action ...string) {
	actionAuth, err := role.GetAuthorityActions(c.GetString("site"), moduleID, c.GetString("session"), c.GetString("clientAgent"), c.GetInt("uid"), action...)
	if err != nil {
		c.AbortWithStatusJSON(200, map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		return
	}
	if len(actionAuth) == 0 && mandatory {
		c.AbortWithStatus(403)
		return
	}

	c.Set("actionAuth", actionAuth)

	c.Next()

	if authList, exists := c.Get("authorizing"); exists {
		args, exists := c.Get("authorizingArgs")
		if !exists {
			args = make([]interface{}, 0)
		}
		for _, auth := range authList.([]authority.IAuth) {
			if err := auth.CheckAuth(c.GetString("site"), actionAuth, args.([]interface{})...); err != nil {
				c.AbortWithStatus(403)
				c.Set("json", nil)
				return
			}
		}
	}
}

var checkAuth = func(moduleID string, action ...string) func(*gin.Context) {
	return func(c *gin.Context) {
		doAuth(c, true, moduleID, action...)
	}
}

var checkAuthSafe = func(moduleID string, action ...string) func(*gin.Context) {
	return func(c *gin.Context) {
		doAuth(c, false, moduleID, action...)
	}
}

var checkAuthFunc = func(checkFunc func(c *gin.Context) (string, []string)) func(*gin.Context) {
	return func(c *gin.Context) {
		moduleID, action := checkFunc(c)

		if len(action) == 0 {
			c.AbortWithStatus(403)
			return
		}

		doAuth(c, true, moduleID, action...)
	}
}

var checkAuthFuncSafe = func(checkFunc func(c *gin.Context) (string, []string)) func(*gin.Context) {
	return func(c *gin.Context) {
		moduleID, action := checkFunc(c)

		if len(action) == 0 {
			c.AbortWithStatus(403)
			return
		}

		doAuth(c, false, moduleID, action...)
	}
}
