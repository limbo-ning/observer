package main

import (
	"errors"
	"log"

	"obsessiontech/environment/resource"

	"github.com/gin-gonic/gin"
)

func loadResource() {
	authorized.POST("resource/upload", checkAuth(resource.MODULE_RESOURCE, resource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		form, err := c.MultipartForm()
		if err != nil {
			log.Println("error: ", err)
			c.String(500, "not multipart-form")
			return
		}

		if uploadList, err := resource.UploadResource(siteID, form.File, form.Value); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "uploadList": uploadList})
		}
	})

	authorized.POST("resource/edit/:method", checkAuth(resource.MODULE_RESOURCE, resource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			File     string `json:"file"`
			DestFile string `json:"destFile"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "move":
			if err := resource.MoveResource(siteID, param.File, param.DestFile); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			if err := resource.DeleteResource(siteID, param.File); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("resource/list", checkAuth(resource.MODULE_RESOURCE, resource.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if files, err := resource.ListResource(siteID, c.Query("folderPath")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "fileList": files})
		}
	})
}
