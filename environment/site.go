package main

import (
	"errors"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/site"

	"github.com/gin-gonic/gin"
)

func loadSite() {

	sites.GET("site/series", func(c *gin.Context) {
		if siteList, err := site.GetSiteSeries(c.GetString("site"), c.Query("status"), c.Query("type"), c.Query("geoType")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "siteList": siteList})
		}
	})

	internal.GET("site/checkCname", func(c *gin.Context) {
		if siteID, err := site.GetSiteIDByCName(c.Query("cname")); err != nil {
			c.PureJSON(200, map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.PureJSON(200, map[string]interface{}{"retCode": 0, "siteID": siteID})
		}
	})

	projectc.GET("site", func(c *gin.Context) {
		if siteList, err := site.MySites(c.GetInt("uid"), c.Query("geoType")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "siteList": siteList})
		}
	})

	authorized.POST("site/edit/update", checkAuth(site.MODULE_SITE, site.ACTION_C_EDIT_SITE, site.ACTION_ADMIN_EDIT_SITE), func(c *gin.Context) {

		actionAuth, _ := c.Get("actionAuth")
		var param site.Site

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		param.SiteID = c.GetString("site")

		if err := param.Update(actionAuth.(authority.ActionAuthSet)); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("site/cname", checkAuth(site.MODULE_SITE, site.ACTION_C_EDIT_SITE, site.ACTION_ADMIN_EDIT_SITE), func(c *gin.Context) {
		siteID := c.GetString("site")

		if cnames, err := site.GetCNames(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "cnames": cnames})
		}
	})

	authorized.POST("site/cname/edit/:method", checkAuth(site.MODULE_SITE, site.ACTION_C_EDIT_SITE, site.ACTION_ADMIN_EDIT_SITE), func(c *gin.Context) {
		var param struct {
			CName string `json:"cname"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "bind":
			if err := site.BindCName(c.GetString("site"), param.CName); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "unbind":
			if err := site.UnbindCName(c.GetString("site"), param.CName); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("site/module", func(c *gin.Context) {
		siteID := c.GetString("site")

		moduleIDs := make([]string, 0)
		if c.Query("moduleID") != "" {
			moduleIDs = strings.Split(c.Query("moduleID"), ",")
		}

		if moduleList, err := site.GetSiteModuleList(siteID, moduleIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "moduleList": moduleList})
		}
	})

	authorized.GET("site/siteModule", checkAuth(site.MODULE_MODULE, site.ACTION_ADMIN_VIEW_MODULE), func(c *gin.Context) {
		if siteModuleList, err := site.GetSiteModules(c.GetString("site"), c.Query("moduleID"), c.Query("prefix")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "siteModuleList": siteModuleList})
		}
	})
}
