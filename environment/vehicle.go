package main

import (
	"errors"
	"obsessiontech/environment/peripheral/vehicle"
	"strconv"

	"github.com/gin-gonic/gin"
)

func loadVehicle() {
	authorized.GET("peripheral/vehicle/module", checkAuth(vehicle.MODULE_VEHICLE, vehicle.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if vehicleModule, err := vehicle.GetModule(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "vehicleModule": vehicleModule})
		}
	})

	authorized.POST("peripheral/vehicle/module/edit/save", checkAuth(vehicle.MODULE_VEHICLE, vehicle.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param vehicle.VehicleModule

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

	authorized.GET("peripheral/vehicle", checkAuth(vehicle.MODULE_VEHICLE, vehicle.ACTION_ADMIN_VIEW, vehicle.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		status, _ := strconv.Atoi(c.Query("status"))

		if vehicleList, err := vehicle.GetVehicleList(siteID, c.Query("type"), status, c.Query("search")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "vehicleList": vehicleList})
		}
	})

	authorized.POST("peripheral/vehicle/edit/:method", checkAuth(vehicle.MODULE_VEHICLE, vehicle.ACTION_ADMIN_EDIT), func(c *gin.Context) {

		siteID := c.GetString("site")

		var param vehicle.Vehicle
		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "add":
			if err := param.Add(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "vehicle": param})
			}
		case "update":
			if err := param.Update(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "vehicle": param})
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
