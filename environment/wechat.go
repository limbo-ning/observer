package main

import (
	"errors"
	"log"
	"strconv"

	"obsessiontech/environment/wechat"
	"obsessiontech/wechat/util"

	"github.com/gin-gonic/gin"
)

func loadWechat() {
	router.POST("wechat/platform/authorization/:method", func(c *gin.Context) {
		switch c.Param("method") {
		case "authorize":
			code := c.Query("code")
			siteID := c.Query("csite")
			if c.Query("siteID") != "" {
				siteID = c.Query("siteID")
			}

			if err := wechat.Authorize(siteID, code); err != nil {
				c.PureJSON(200, map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.PureJSON(200, map[string]interface{}{"retCode": 0})
			}
		case "push":
			timestamp, _ := strconv.Atoi(c.Query("timestamp"))
			msgSignature := c.Query("msg_signature")
			nonce := c.Query("nonce")
			encryptType := c.Query("encrypt_type")

			data, err := c.GetRawData()
			if err != nil {
				log.Println("error get raw data from wechat verify ticket push: ", err)
				c.AbortWithError(400, err)
				return
			}

			if err := wechat.ReceiveAuthorizationPush(timestamp, msgSignature, nonce, encryptType, data); err != nil {
				log.Println("error get wechat verify ticket push: ", err)
				c.AbortWithError(400, err)
			} else {
				c.String(200, "success")
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	router.POST("wechat/platform/appPush/:appID", func(c *gin.Context) {
		appID := c.Param("appID")

		timestamp, _ := strconv.Atoi(c.Query("timestamp"))
		msgSignature := c.Query("msg_signature")
		nonce := c.Query("nonce")
		encryptType := c.Query("encrypt_type")

		data, err := c.GetRawData()
		if err != nil {
			log.Println("error get raw data from wechat verify ticket push: ", err)
			c.AbortWithError(400, err)
			return
		}

		if contentType, response, err := wechat.ReceiveMessagePush(appID, timestamp, msgSignature, nonce, encryptType, data); err != nil {
			log.Println("error process wechat platform message push: ", err)
			c.AbortWithError(400, err)
		} else {
			c.Data(200, contentType, response)
		}
	})

	internal.GET("wechat/jsapi", func(c *gin.Context) {

		referer, exists := c.GetQuery("referer")
		if !exists {
			referer = c.Request.Referer()
		}

		c.PureJSON(200, util.GetWxConfig(referer))
	})

	sites.POST("wechat/miniapp/code/:type", func(c *gin.Context) {
		var param wechat.MiniAppCodeParam

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("type") {
		case "permanent":
			if data, err := wechat.GetMiniAppCodePermanent(&param); err != nil {
				c.AbortWithError(400, err)
			} else {
				c.Data(200, "image/png", data)
			}
		case "unlimit":
			if data, err := wechat.GetMiniAppCodeUnlimit(&param); err != nil {
				c.AbortWithError(400, err)
			} else {
				c.Data(200, "image/png", data)
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("wechat/agent", checkAuth(wechat.MODULE_AGENT, wechat.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if agents, err := wechat.GetSiteAgents(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "wechatAgentList": agents})
		}
	})

	authorized.GET("wechat/agent/openServiceAccount", checkAuth(wechat.MODULE_AGENT, wechat.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if openAppID, err := wechat.GetOpenServiceAccount(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "openAppID": openAppID})
		}
	})

	authorized.POST("wechat/agent/:method", checkAuth(wechat.MODULE_AGENT, wechat.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		appID := c.Query("appID")

		switch c.Param("method") {
		case "bind":
			if agent, err := wechat.BindSiteAgent(siteID, appID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "wechatAgent": agent})
			}
		case "unbind":
			if err := wechat.UnbindSiteAgent(siteID, appID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "grant":
			if pcLink, wechatLink, err := wechat.Grant(appID, c.Query("redirectURL")); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "pcLink": pcLink, "wechatLink": wechatLink})
			}
		case "refresh":
			if agent, err := wechat.GetAgent(appID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else if err := agent.RefreshInfo(); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "wechatAgent": agent})
			}
		case "createOpenServiceAccount":
			if openAppID, err := wechat.CreateOpenServiceAccount(appID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "openAppID": openAppID})
			}
		case "bindOpenServiceAccount":
			if err := wechat.BindOpenServiceAccount(appID, c.Query("openAppID")); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "unbindOpenServiceAccount":
			if err := wechat.UnbindOpenServiceAccount(appID, c.Query("openAppID")); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	sites.GET("wechat/agent/material", func(c *gin.Context) {
		appID := c.Query("appID")

		if material, err := wechat.DownloadMaterial(appID, c.Query("mediaID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "material": material})
		}
	})

}
