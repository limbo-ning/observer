package main

import (
	"github.com/gin-gonic/gin"

	"obsessiontech/common/config"
	"obsessiontech/common/http"
	"obsessiontech/wechat/util"
)

var Config routerConfig

type routerConfig struct {
	Sign string
}

func init() {
	config.GetConfig("config.yaml", &Config)
}

func main() {

	server := gin.Default()
	router := server.Group(http.GetPrefix())

	router.GET("receive", func(c *gin.Context) {
		c.String(200, c.Query("echostr"))
	})

	auth := router.Group("auth", func(c *gin.Context) {
		if c.Query("sign") != Config.Sign {
			c.AbortWithStatus(403)
			return
		} else {
			c.Next()
		}
	})

	auth.GET("jsapi", func(c *gin.Context) {

		referer, exists := c.GetQuery("referer")
		if !exists {
			referer = c.Request.Referer()
		}

		c.PureJSON(200, util.GetWxConfig(referer))
	})

	auth.GET("media/download", func(c *gin.Context) {

		if err := util.DownloadMedia(c.Writer, c.Query("mediaID"), c.Query("amrConvertTo"), func(contentType string) {
			c.Header("Content-Type", contentType)
		}, func(filename string) {
			c.Header("Content-Dispositon", "attachment;filename="+filename)
		}); err != nil {
			c.AbortWithError(500, err)
		}
	})

	auth.GET("voice/download", func(c *gin.Context) {
		c.Header("Accept-Ranges", "bytes")
		c.Header("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		if err := util.DownloadVoice(c.Writer, c.Query("mediaID"), func(contentType string) {
			c.Header("Content-Type", contentType)
		}, func(filename string) {
			c.Header("Content-Dispositon", "attachment;filename="+filename)
		}); err != nil {
			c.AbortWithError(500, err)
		}
	})

	server.Run(":" + http.GetPort())
}
