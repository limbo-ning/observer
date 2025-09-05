package main

import (
	"errors"
	"strconv"
	"strings"

	"obsessiontech/environment/site"
	"obsessiontech/environment/site/module"
	"obsessiontech/environment/wechat"
	"obsessiontech/wechat/util"

	"github.com/gin-gonic/gin"
)

func loadCadmin() {

	projectc.GET("module", checkAuth(site.MODULE_MODULE, site.ACTION_C_VIEW_MODULE), func(c *gin.Context) {
		cids := make([]int, 0)
		cidlist, exists := c.GetQuery("categoryIDs")
		if !exists {
			cidlist, exists = c.GetQuery("categoryID")
		}
		if exists && cidlist != "" {
			cl := strings.Split(cidlist, ",")
			for _, cidstr := range cl {
				if cid, err := strconv.Atoi(cidstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					cids = append(cids, cid)
				}
			}
		}

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		moduleIDs := make([]string, 0)
		if moduleID, exists := c.GetQuery("moduleID"); exists && strings.TrimSpace(moduleID) != "" {
			moduleIDs = strings.Split(moduleID, ",")
		}

		if moduleList, total, err := module.GetModuleList(cids, c.Query("q"), pageNo, pageSize, moduleIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "total": total, "moduleList": moduleList})
		}
	})

	projectc.POST("module/edit/:method", checkAuth(site.MODULE_MODULE, site.ACTION_C_EDIT_MODULE), func(c *gin.Context) {
		var param site.Module

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "add":
			if err := param.Add(); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "update":
			if err := param.Update(); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			if err := param.Delete(); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	projectc.POST("module/category/:method", checkAuth(site.MODULE_MODULE, site.ACTION_C_EDIT_MODULE), func(c *gin.Context) {
		var param struct {
			ModuleID   string `json:"moduleID"`
			CategoryID int    `json:"categoryID"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "bind":
			if err := module.AddModuleCategory(param.ModuleID, param.CategoryID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "unbind":
			if err := module.DeleteModuleCategory(param.ModuleID, param.CategoryID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	projectc.GET("wechat/admin/agent", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_VIEW), func(c *gin.Context) {
		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		if wechatAgentList, total, err := wechat.GetAgentList(c.Query("siteID"), c.Query("appID"), c.Query("appType"), c.Query("status"), c.Query("q"), pageNo, pageSize, c.Query("orderBy")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "wechatAgentList": wechatAgentList, "total": total})
		}
	})

	projectc.POST("wechat/admin/upload/:method", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		data, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatus(500)
			return
		}

		switch c.Param("method") {
		case "media":
			if mediaID, err := wechat.UploadMedia(c.Query("appID"), c.Query("mediaType"), data); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "mediaID": mediaID})
			}
		default:
			c.AbortWithError(403, errors.New("not support"))
		}
	})

	projectc.POST("wechat/admin/download/:target", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		switch c.Param("target") {
		case "media":
			if err := wechat.DownloadMedia(c.Query("appID"), c.Writer, c.Query("mediaID"), c.Query("amrConvertTo"), func(contentType string) {
				c.Header("Content-Type", contentType)
			}, func(filename string) {
				c.Header("Content-Dispositon", "attachment;filename="+filename)
			}); err != nil {
				c.AbortWithError(500, err)
			}
		case "material":
			if material, err := wechat.DownloadMaterial(c.Query("appID"), c.Query("mediaID")); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "material": material})
			}
		default:
			c.AbortWithError(403, errors.New("not support"))
		}
	})

	projectc.GET("wechat/admin/miniapp/templateList", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if templateList, err := util.PlatformGetMiniAppTemplateList(); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "templateList": templateList})
		}
	})

	projectc.GET("wechat/admin/miniapp/pageList", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if pageList, err := wechat.GetMiniAppPageList(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "pageList": pageList})
		}
	})

	projectc.GET("wechat/admin/miniapp/categoryList", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if categoryList, err := wechat.GetMiniAppPageCategoryList(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "categoryList": categoryList})
		}
	})

	projectc.GET("wechat/admin/miniapp/submitStatus", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if status, err := wechat.QueryMiniAppSubmit(c.Query("appID"), c.Query("auditID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "status": status})
		}
	})

	projectc.GET("wechat/admin/miniapp/domain", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if domain, err := wechat.GetMiniAppDomain(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "domain": domain})
		}
	})

	projectc.GET("wechat/admin/miniapp/webviewdomain", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if domain, err := wechat.GetMiniAppWebviewDomain(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "domain": domain})
		}
	})

	projectc.GET("wechat/admin/miniapp/privacySetting", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		var param util.QueryPrivacySettingReq
		ver, exists := c.GetQuery("privacyVer")
		if exists {
			param.PrivacyVer, _ = strconv.Atoi(ver)
		}
		if privacySetting, err := wechat.QueryMiniAppPrivacySetting(c.Query("appID"), &param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "privacySetting": privacySetting})
		}
	})

	projectc.GET("wechat/admin/miniapp/apiSetting", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if apiSetting, err := wechat.GetMiniAppApiSetting(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "apiSetting": apiSetting})
		}
	})

	projectc.GET("wechat/admin/miniapp/apiReview", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		if apiReview, err := wechat.GetMiniAppApiReview(c.Query("appID")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "apiReview": apiReview})
		}
	})

	projectc.POST("wechat/admin/miniapp/:method", checkAuth(wechat.MODULE_C_AGENT, wechat.ACTION_C_EDIT), func(c *gin.Context) {
		switch c.Param("method") {
		case "upload":
			var param struct {
				AppID           string                 `json:"appID"`
				TemplateID      string                 `json:"templateID"`
				UserVersion     string                 `json:"userVersion"`
				UserDescription string                 `json:"userDescription"`
				ExtJSON         map[string]interface{} `json:"extJSON"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := wechat.UploadMiniAppTemplateCode(param.AppID, param.TemplateID, param.UserVersion, param.UserDescription, param.ExtJSON); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "preview":
			var param struct {
				AppID string `json:"appID"`
				Path  string `json:"path"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if data, err := wechat.PreviewMiniApp(param.AppID, param.Path); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Data(200, "image/jpeg", data)
			}
		case "submitMedia":
			data, err := c.GetRawData()
			if err != nil {
				c.AbortWithStatus(500)
				return
			}

			if mediaType, mediaID, err := wechat.SubmitMiniAppMedia(c.Query("appID"), data); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "mediaType": mediaType, "mediaID": mediaID})
			}
		case "submit":
			var param struct {
				util.SubmitMiniAppReq
				AppID string `json:"appID"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if auditID, err := wechat.SubmitMiniApp(param.AppID, &param.SubmitMiniAppReq); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "auditID": auditID})
			}
		case "retreat":
			var param struct {
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := wechat.RetreatMiniAppSubmit(param.AppID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "speedup":
			var param struct {
				AppID   string `json:"appID"`
				AuditID string `json:"auditID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := wechat.SpeedupMiniAppSubmit(param.AppID, param.AuditID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "release":
			var param struct {
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := wechat.ReleaseMiniApp(param.AppID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "setDomain":
			var param struct {
				util.DomainReq
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := wechat.SetMiniAppDomain(param.AppID, &param.DomainReq); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "setWebviewDomain":
			var param struct {
				util.WebviewDomainReq
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := wechat.SetMiniAppWebviewDomain(param.AppID, &param.WebviewDomainReq); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "setPrivacySetting":
			var param struct {
				util.SetPrivacySettingReq
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := wechat.SetMiniAppPrivacySetting(param.AppID, &param.SetPrivacySettingReq); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "applyApi":
			var param struct {
				util.ApplyApiParam
				AppID string `json:"appID"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := wechat.ApplyMiniAppApi(param.AppID, &param.ApplyApiParam); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(403, errors.New("invalid method"))
		}
	})
}
