package main

import (
	"errors"
	"strconv"

	"obsessiontech/environment/peripheral/surveillance"

	"github.com/gin-gonic/gin"
)

func loadSurveillance() {
	authorized.GET("peripheral/surveillance/module", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if surveillanceModule, err := surveillance.GetModule(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "surveillanceModule": surveillanceModule})
		}
	})

	authorized.POST("peripheral/surveillance/module/edit/save", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param surveillance.SurveillanceModule

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

	authorized.GET("peripheral/surveillance/ezviz/device", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		if deviceList, total, err := surveillance.GetDeviceList(siteID, pageNo, pageSize); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "deviceList": deviceList, "total": total})
		}
	})
	authorized.GET("peripheral/surveillance/ezviz/device/camera", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_VIEW, surveillance.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if cameraList, err := surveillance.GetDeviceCameraList(siteID, c.Query("deviceSerial")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "cameraList": cameraList})
		}
	})

	authorized.GET("peripheral/surveillance/ezviz/device/camera/live", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_VIEW, surveillance.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		channelNo, _ := strconv.Atoi(c.Query("channelNo"))

		if liveURL, accessToken, err := surveillance.GetLiveURL(siteID, c.Query("deviceSerial"), c.Query("code"), channelNo); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "liveURL": liveURL, "accessToken": accessToken})
		}
	})

	authorized.POST("peripheral/surveillance/ezviz/device/edit/:method", checkAuth(surveillance.MODULE_SURVEILLANCE, surveillance.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		switch c.Param("method") {
		case "add":
			var param struct {
				DeviceSerial string `json:"deviceSerial"`
				ValidateCode string `json:"validateCode"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := surveillance.AddDevice(siteID, param.DeviceSerial, param.ValidateCode); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			var param struct {
				DeviceSerial string `json:"deviceSerial"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := surveillance.DeleteDevice(siteID, param.DeviceSerial); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "updateName":
			var param struct {
				DeviceSerial string `json:"deviceSerial"`
				DeviceName   string `json:"deviceName"`
			}
			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := surveillance.UpdateDeviceName(siteID, param.DeviceSerial, param.DeviceName); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})
}
