package main

import (
	"errors"
	"strconv"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/peripheral"
	"obsessiontech/environment/peripheral/speaker"

	"github.com/gin-gonic/gin"
)

func loadPeripheral() {
	authorized.GET("peripheral/device", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_VIEW, peripheral.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		empowerID := make([]string, 0)
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerID = strings.Split(str, ",")
		}

		actionAuth, _ := c.Get("actionAuth")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		if deviceList, total, err := peripheral.GetDevices(siteID, actionAuth.(authority.ActionAuthSet), c.Query("serial"), c.Query("type"), c.Query("q"), pageNo, pageSize, c.Query("auth"), c.Query("empower"), empowerID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "deviceList": deviceList, "total": total})
		}
	})

	authorized.POST("peripheral/device/edit/:method", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_EDIT, peripheral.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param peripheral.Device
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

	authorized.GET("peripheral/device/empower", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_VIEW, peripheral.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		deviceID := strings.Split(c.Query("deviceID"), ",")

		actionAuth, _ := c.Get("actionAuth")

		if deviceEmpowers, err := authority.GetEmpowers(siteID, "device", actionAuth.(authority.ActionAuthSet), peripheral.AdminActions, deviceID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "deviceEmpowers": deviceEmpowers})
		}
	})

	authorized.GET("peripheral/device/empower/detail", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_VIEW, peripheral.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		var deviceID []string
		if str, exists := c.GetQuery("deviceID"); exists && strings.TrimSpace(str) != "" {
			deviceID = strings.Split(str, ",")
		}

		var empowerIDs []string
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerIDs = strings.Split(str, ",")
		}

		if deviceEmpowerDetails, err := authority.GetEmpowerDetails(siteID, c.Query("empower"), empowerIDs, "device", deviceID, c.Query("groupBy")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "deviceEmpowerDetails": deviceEmpowerDetails})
		}
	})

	authorized.POST("peripheral/device/empower/:method", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_EDIT, peripheral.ACTION_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "add":
			var param struct {
				DeviceID  int      `json:"deviceID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := peripheral.AddDeviceEmpower(siteID, actionAuth.(authority.ActionAuthSet), param.DeviceID, param.Empower, param.EmpowerID, param.AuthList); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			var param struct {
				DeviceID  int      `json:"deviceID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := peripheral.DeleteDeviceEmpower(siteID, actionAuth.(authority.ActionAuthSet), param.DeviceID, param.Empower, param.EmpowerID, param.AuthList...); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("peripheral/speaker/status", checkAuth(peripheral.MODULE_PEREPHERAL, peripheral.ACTION_ADMIN_VIEW, peripheral.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		deviceID := make([]int, 0)
		for _, idstr := range strings.Split(c.Query("deviceID"), ",") {
			id, err := strconv.Atoi(idstr)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			deviceID = append(deviceID, id)
		}

		if speakerStatus, err := speaker.GetSpeakerStatus(siteID, deviceID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "speakerStatus": speakerStatus})
		}
	})
}
