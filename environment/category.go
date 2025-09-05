package main

import (
	"errors"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/category"
	"obsessiontech/environment/category/module"

	"github.com/gin-gonic/gin"
)

func loadCategory() {

	sites.GET("category/module", func(c *gin.Context) {
		siteID := c.GetString("site")

		if categoryModule, err := module.GetModule(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "categoryModule": categoryModule})
		}
	})

	authorized.POST("category/module/edit/save", checkAuth(module.MODULE_CATEGORY, module.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param module.CategoryModule

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

	// authorized.POST("category/key/edit/:method", checkAuth(module.MODULE_CATEGORY, module.ACTION_ADMIN_EDIT), func(c *gin.Context) {
	// 	siteID := c.GetString("site")

	// 	var param category.CategoryKey

	// 	if err := c.ShouldBindJSON(&param); err != nil {
	// 		c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		return
	// 	}
	// 	switch c.Param("method") {
	// 	case "add":
	// 		if err := param.Add(siteID); err != nil {
	// 			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		} else {
	// 			c.Set("json", map[string]interface{}{"retCode": 0})
	// 		}
	// 	case "update":
	// 		if err := param.Update(siteID); err != nil {
	// 			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		} else {
	// 			c.Set("json", map[string]interface{}{"retCode": 0})
	// 		}
	// 	case "delete":
	// 		if err := param.Delete(siteID); err != nil {
	// 			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		} else {
	// 			c.Set("json", map[string]interface{}{"retCode": 0})
	// 		}
	// 	default:
	// 		c.AbortWithError(404, errors.New("invalid method"))
	// 	}
	// })

	// sites.GET("category/key/clientAgent", func(c *gin.Context) {
	// 	siteID := c.GetString("site")

	// 	cids := make([]int, 0)
	// 	if c.Query("categoryID") != "" {
	// 		for _, idStr := range strings.Split(c.Query("categoryID"), ",") {
	// 			id, err := strconv.Atoi(idStr)
	// 			if err != nil {
	// 				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 				return
	// 			}

	// 			cids = append(cids, id)
	// 		}
	// 	}

	// 	if keyClientAgents, err := category.GetCategoryKeyClientAgents(siteID, cids...); err != nil {
	// 		c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 	} else {
	// 		c.Set("json", map[string]interface{}{"retCode": 0, "categoryKeyClientAgents": keyClientAgents})
	// 	}

	// })

	// authorized.POST("category/key/clientAgent/:method", checkAuth(module.MODULE_CATEGORY, module.ACTION_ADMIN_EDIT), func(c *gin.Context) {
	// 	siteID := c.GetString("site")

	// 	var param struct {
	// 		CategoryKeyID int    `json:"categoryKeyID"`
	// 		ClientAgent   string `json:"clientAgent"`
	// 	}

	// 	if err := c.ShouldBindJSON(&param); err != nil {
	// 		c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		return
	// 	}

	// 	switch c.Param("method") {
	// 	case "bind":
	// 		if err := category.AddCategoryKeyClientAgentMapping(siteID, param.CategoryKeyID, param.ClientAgent); err != nil {
	// 			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		} else {
	// 			c.Set("json", map[string]interface{}{"retCode": 0})
	// 		}
	// 	case "unbind":
	// 		if err := category.DeleteCategoryKeyClientAgentMapping(siteID, param.CategoryKeyID, param.ClientAgent); err != nil {
	// 			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 		} else {
	// 			c.Set("json", map[string]interface{}{"retCode": 0})
	// 		}
	// 	default:
	// 		c.AbortWithError(404, errors.New("invalid method"))
	// 	}
	// })

	sites.GET("category/category", checkAuthSafe(module.MODULE_CATEGORY, module.ACTION_ADMIN_VIEW, module.ACTION_VIEW_TYPE, module.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		if len(actionAuth.(authority.ActionAuthSet)) == 0 {
			c.Set("json", map[string]interface{}{"retCode": 0, "categoryKeyList": []interface{}{}, "categories": map[string]interface{}{}})
			return
		}

		types := make([]string, 0)
		if str, exists := c.GetQuery("type"); exists && strings.TrimSpace(str) != "" {
			types = strings.Split(strings.TrimSpace(str), ",")
		}

		if categories, err := category.GetCategories(siteID, c.Query("source"), c.Query("clientAgent"), actionAuth.(authority.ActionAuthSet), types...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "categories": categories})
		}
	})

	sites.GET("category/object", checkAuthSafe(module.MODULE_CATEGORY, module.ACTION_ADMIN_VIEW, module.ACTION_VIEW_TYPE, module.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		if len(actionAuth.(authority.ActionAuthSet)) == 0 {
			c.Set("json", map[string]interface{}{"retCode": 0, "categories": map[string]interface{}{}})
			return
		}

		if c.Query("objectID") == "" || c.Query("source") == "" {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "objectID|source 不能为空"})
			return
		}

		if categoris, err := category.GetObjectCategories(siteID, c.Query("source"), strings.Split(c.Query("objectID"), ","), actionAuth.(authority.ActionAuthSet)); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "categories": categoris})
		}

	})

	sites.GET("category/objectCategory", checkAuthSafe(module.MODULE_CATEGORY, module.ACTION_ADMIN_VIEW, module.ACTION_VIEW_TYPE, module.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		if len(actionAuth.(authority.ActionAuthSet)) == 0 {
			c.Set("json", map[string]interface{}{"retCode": 0, "categories": map[string]interface{}{}})
			return
		}

		if c.Query("objectID") == "" || c.Query("source") == "" {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "objectID|source 不能为空"})
			return
		}

		if categoris, err := category.GetObjectCategories(siteID, c.Query("source"), strings.Split(c.Query("objectID"), ","), actionAuth.(authority.ActionAuthSet)); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "categories": categoris})
		}

	})

	authorized.POST("category/category/edit/:method", checkAuth(module.MODULE_CATEGORY, module.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param category.Category

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "add":
			if err := param.Add(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "update":
			if err := param.Update(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			if err := param.Delete(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

}
