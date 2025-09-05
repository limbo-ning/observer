package main

import (
	"errors"
	"strconv"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/site/page"
	"obsessiontech/environment/site/page/template"

	"github.com/gin-gonic/gin"
)

func loadPage() {

	authorized.GET("page/page", checkAuth(page.MODULE_PAGE, page.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if componentList, err := page.GetPages(c.GetString("site"), c.Query("status")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "componentList": componentList})
		}
	})

	authorized.GET("page/page/component", checkAuth(page.MODULE_PAGE, page.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if componentList, models, err := page.GetPageComponents(c.GetString("site"), c.Query("pageID"), c.Query("status")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "componentList": componentList, "models": models})
		}
	})

	authorized.POST("page/page/compose", checkAuth(page.MODULE_PAGE, page.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param struct {
			PageID        string                `json:"pageID"`
			ComponentList []*page.PageComponent `json:"componentList"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		if err := page.SavePageComponents(c.GetString("site"), param.PageID, param.ComponentList); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("page/page/render", checkAuth(page.MODULE_PAGE, page.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		if componentList, err := page.RenderPage(c.GetString("site"), c.Query("pageID"), c.Query("status")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "componentList": componentList})
		}
	})

	authorized.POST("page/page/export", checkAuth(page.MODULE_PAGE, page.ACTION_ADMIN_PUBLISH), func(c *gin.Context) {
		if err := page.ExportSite(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("page/model", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_VIEW_MODEL), func(c *gin.Context) {
		siteID := c.GetString("site")

		modelIDs := make([]string, 0)
		if modelID, exists := c.GetQuery("modelID"); exists && strings.TrimSpace(modelID) != "" {
			modelIDs = strings.Split(modelID, ",")
		}

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

		if modelList, total, err := page.GetSiteModelList(siteID, cids, c.Query("type"), c.Query("moduleID"), c.Query("q"), pageNo, pageSize, modelIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "total": total, "modelList": modelList})
		}
	})

	authorized.GET("page/model/detail", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_VIEW_MODEL), func(c *gin.Context) {
		siteID := c.GetString("site")

		if models, err := page.GetSiteModels(siteID, strings.Split(c.Query("modelID"), ",")...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "models": models})
		}
	})

	authorized.GET("page/model/child", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_VIEW_MODEL), func(c *gin.Context) {
		siteID := c.GetString("site")

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

		if modelRelationList, models, total, err := page.GetChildModels(siteID, c.Query("parentModelID"), cids, c.Query("type"), c.Query("moduleID"), c.Query("q"), pageNo, pageSize); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "total": total, "modelRelationList": modelRelationList, "models": models})
		}
	})

	authorized.GET("page/model/relation", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_VIEW_MODEL), func(c *gin.Context) {
		siteID := c.GetString("site")

		var parentModelID []string
		if IDStr, exists := c.GetQuery("parentModelID"); exists && strings.TrimSpace(IDStr) != "" {
			parentModelID = strings.Split(IDStr, ",")
		}

		var childModelID []string
		if IDStr, exists := c.GetQuery("childModelID"); exists && strings.TrimSpace(IDStr) != "" {
			childModelID = strings.Split(IDStr, ",")
		}

		relationID := make([]int, 0)
		if relationIDStr, exists := c.GetQuery("relationID"); exists && strings.TrimSpace(relationIDStr) != "" {
			for _, str := range strings.Split(relationIDStr, ",") {
				id, err := strconv.Atoi(str)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				relationID = append(relationID, id)
			}
		}

		if modelRelationList, err := page.GetModelRelations(siteID, parentModelID, childModelID, relationID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "modelRelationList": modelRelationList})
		}
	})

	authorized.POST("page/model/edit/:method", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_EDIT_MODEL), func(c *gin.Context) {

		siteID := c.GetString("site")
		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "add":
			var param page.Model
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Add(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "update":
			var param page.Model
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Update(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			var param page.Model
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Delete(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "addChild":
			var param page.ModelRelation
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Add(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "updateChild":
			var param page.ModelRelation
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Update(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "deleteChild":
			var param page.ModelRelation
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Delete(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.POST("page/model/assemble/:method", checkAuth(page.MODULE_MODEL, page.ACTION_ADMIN_EDIT_MODEL), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "add":
			var param struct {
				page.Model
				ComponentList []*template.BaseComponent    `json:"componentList"`
				ParamAlias    map[string]map[string]string `json:"paramAlias"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Model.AddAssembleModel(siteID, actionAuth.(authority.ActionAuthSet), param.ComponentList, param.ParamAlias); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "update":
			var param struct {
				page.Model
				ComponentList []*template.BaseComponent    `json:"componentList"`
				ParamAlias    map[string]map[string]string `json:"paramAlias"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := param.Model.UpdateAssembleModel(siteID, actionAuth.(authority.ActionAuthSet), param.ComponentList, param.ParamAlias); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		}
	})
}
