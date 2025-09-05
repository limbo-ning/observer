package main

import (
	"errors"
	"log"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment"
	"obsessiontech/environment/site/clientAgent"

	"github.com/gin-gonic/gin"
)

func loadClientAgent() {

	sites.GET("clientAgent", func(c *gin.Context) {
		if clientAgentList, err := clientAgent.GetClientAgentList(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "clientAgentList": clientAgentList})
		}
	})

	authorized.GET("clientAgent/module", checkAuth(clientAgent.MODULE_CLIENTAGENT, clientAgent.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if clientAgentModule, err := clientAgent.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "clientAgentModule": clientAgentModule})
		}
	})

	authorized.POST("clientAgent/module/edit/save", checkAuth(clientAgent.MODULE_CLIENTAGENT, clientAgent.ACTION_ADMIN_EDIT), logger(environment.MODULE_ENVIRONMENT, "site_module", "save"), func(c *gin.Context) {
		var param clientAgent.ClientAgentModule

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if err := param.Save(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	sites.GET("clientAgent/setting/typeKeys", checkAuthFuncSafe(func(c *gin.Context) (string, []string) {
		return c.Query("clientAgent"), []string{clientAgent.ACTION_ADMIN_VIEW, clientAgent.ACTION_VIEW}
	}), func(c *gin.Context) {
		module, err := clientAgent.GetModule(c.GetString("site"))
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			typeKeys := module.SettingTypeKeys
			if typeKeys == nil {
				typeKeys = make(map[string][]string)
			}
			c.Set("json", map[string]interface{}{"retCode": 0, "typeKeys": typeKeys})
		}
	})

	sites.GET("clientAgent/setting", checkAuthFuncSafe(func(c *gin.Context) (string, []string) {
		return c.Query("clientAgent"), []string{clientAgent.ACTION_ADMIN_VIEW, clientAgent.ACTION_VIEW}
	}), func(c *gin.Context) {
		actionAuth, _ := c.Get("actionAuth")
		if settingList, err := clientAgent.GetSettings(c.GetString("site"), actionAuth.(authority.ActionAuthSet), c.Query("clientAgent"), c.Query("key"), c.Query("type"), c.Query("q")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "settingList": settingList})
		}
	})

	authorized.POST("clientAgent/setting/edit/:method", checkAuthFunc(func(c *gin.Context) (string, []string) {
		var param clientAgent.Setting
		if err := c.ShouldBindJSON(&param); err != nil {
			return "", nil
		}
		c.Set("param", &param)
		return param.ClientAgent, []string{clientAgent.ACTION_ADMIN_EDIT}
	}), func(c *gin.Context) {
		siteID := c.GetString("site")
		param, _ := c.Get("param")

		setting := param.(*clientAgent.Setting)

		switch c.Param("method") {
		case "add":
			if err := setting.Add(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "update":
			if err := setting.Update(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			if err := setting.Delete(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "clear":
			if err := clientAgent.ClearSetting(siteID, setting.ClientAgent, setting.Key, setting.Type); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.POST("clientAgent/setting/upload/excel", checkAuthFunc(func(c *gin.Context) (string, []string) {
		return c.Query("clientAgent"), []string{clientAgent.ACTION_ADMIN_EDIT}
	}), func(c *gin.Context) {
		siteID := c.GetString("site")

		form, err := c.MultipartForm()
		if err != nil {
			log.Println("error not multipart-form: ", err)
			c.String(500, "not multipart-form")
			return
		}

		if uploadedList, err := clientAgent.UploadExcel(siteID, form.File, form.Value); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "uploadedList": uploadedList})
		}
	})
}
