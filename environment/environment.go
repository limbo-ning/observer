package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/data/operation"
	"obsessiontech/environment/environment/data/recent"
	"obsessiontech/environment/environment/data/upload"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/externalsource"
	"obsessiontech/environment/environment/ipcclient"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/environment/protocol"
	"obsessiontech/environment/environment/stats"
	"obsessiontech/environment/environment/subscription"
	"obsessiontech/environment/logging"

	"github.com/gin-gonic/gin"

	_ "obsessiontech/environment/environment/receiver/HJ/hjt212"
	_ "obsessiontech/environment/environment/receiver/fume"
	_ "obsessiontech/environment/environment/receiver/noise"
	_ "obsessiontech/environment/environment/receiver/odor"
	_ "obsessiontech/environment/environment/receiver/thwater"
)

func init() {
	logging.Register(environment.MODULE_ENVIRONMENT, logging.ParseRegistrant("site_module", "模块设置", [2]string{"save", "修改"}))

	logging.Register(entity.MODULE_ENTITY,
		logging.ParseRegistrant("entity", "企业", [2]string{"add", "新增"}, [2]string{"update", "修改"}, [2]string{"delete", "删除"}, [2]string{"bindCategory", "添加分类"}, [2]string{"unbindCategory", "删除分类"}),
		logging.ParseRegistrant("station", "监测点", [2]string{"add", "新增"}, [2]string{"update", "修改"}, [2]string{"delete", "删除"}, [2]string{"bindCategory", "添加分类"}, [2]string{"unbindCategory", "删除分类"}),
	)

	logging.Register(monitor.MODULE_MONITOR,
		logging.ParseRegistrant("monitor", "监测物", [2]string{"add", "新增"}, [2]string{"update", "修改"}, [2]string{"delete", "删除"}, [2]string{"bindCategory", "添加分类"}, [2]string{"unbindCategory", "删除分类"}),
		logging.ParseRegistrant("monitorCode", "监测物因子", [2]string{"add", "新增"}, [2]string{"update", "修改"}, [2]string{"delete", "删除"}),
		logging.ParseRegistrant("monitorLimit", "监测物限值", [2]string{"add", "新增"}, [2]string{"update", "修改"}, [2]string{"delete", "删除"}),
	)
}

func loadEnvironment() {

	authorized.GET("environment/protocol", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		protocols := protocol.GetSupportedProtocols()

		c.Set("json", map[string]interface{}{"retCode": 0, "protocolList": protocols})
	})

	authorized.GET("environment/module", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_VIEW, environment.ACTION_VIEW), func(c *gin.Context) {
		if environmentModule, err := environment.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "environmentModule": environmentModule})
		}
	})

	authorized.POST("environment/module/edit/save", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT), logger(environment.MODULE_ENVIRONMENT, "site_module", "save"), func(c *gin.Context) {
		var param environment.EnvironmentModule

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if err := param.Save(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			go ipcclient.NotifyModuleChange(c.GetString("site"))
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("environment/entity", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

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

		entityIDs := make([]int, 0)
		if idlist := c.Query("entityID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					entityIDs = append(entityIDs, id)
				}
			}
		}

		empowerID := make([]string, 0)
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerID = strings.Split(str, ",")
		}

		if entityList, err := entity.GetEntityList(siteID, actionAuth.(authority.ActionAuthSet), c.Query("auth"), c.Query("empower"), empowerID, cids, c.Query("q"), entityIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			authList := make([]entity.IEntityAuth, 0)
			for _, ele := range entityList {
				authList = append(authList, ele)
			}
			authList, err := entity.FilterEntityAuthInterface(siteID, authList, actionAuth.(authority.ActionAuthSet), entity.ACTION_ENTITY_VIEW)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "entityList": authList})
			}
		}
	})

	authorized.POST("environment/entity/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return entity.MODULE_ENTITY, "entity", c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")
			actionAuth, _ := c.Get("actionAuth")

			var param entity.Entity

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			switch c.Param("method") {
			case "add":
				err = param.Add(siteID, actionAuth.(authority.ActionAuthSet))
			case "update":
				err = param.Update(siteID, actionAuth.(authority.ActionAuthSet))
			case "delete":
				err = param.Delete(siteID, actionAuth.(authority.ActionAuthSet))
			default:
				c.AbortWithStatus(404)
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("loggingID", param.ID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0, "entity": param})
			}
		},
	)

	authorized.POST("environment/entity/category/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return entity.MODULE_ENTITY, "entity", c.Param("method") + "Category"
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			var param struct {
				EntityID   int `json:"entityID"`
				CategoryID int `json:"categoryID"`
			}

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			switch c.Param("method") {
			case "bind":
				err = entity.AddEntityCategory(siteID, param.EntityID, param.CategoryID)
			case "unbind":
				err = entity.DeleteEntityCategory(siteID, param.EntityID, param.CategoryID)
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("loggingID", param.EntityID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		},
	)

	authorized.GET("environment/entity/empower", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ADMIN_EXPORT, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_VIEW, entity.ACTION_ENTITY_EXPORT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		entityIDs := strings.Split(c.Query("entityID"), ",")
		actionAuth, _ := c.Get("actionAuth")

		if entityEmpowers, err := authority.GetEmpowers(siteID, "entity", actionAuth.(authority.ActionAuthSet), entity.AdminActions, entityIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "entityEmpowers": entityEmpowers})
		}
	})

	authorized.GET("environment/entity/empower/detail", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		var entityIDs []string
		if str, exists := c.GetQuery("entityID"); exists && strings.TrimSpace(str) != "" {
			entityIDs = strings.Split(str, ",")
		}

		var empowerIDs []string
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerIDs = strings.Split(str, ",")
		}

		if entityEmpowerDetails, err := authority.GetEmpowerDetails(siteID, c.Query("empower"), empowerIDs, "entity", entityIDs, c.Query("groupBy")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "entityEmpowerDetails": entityEmpowerDetails})
		}
	})

	authorized.POST("environment/entity/empower/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "add":
			var param struct {
				EntityID  int      `json:"entityID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := entity.AddEntityEmpower(siteID, actionAuth.(authority.ActionAuthSet), param.EntityID, param.Empower, param.EmpowerID, param.AuthList); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			var param struct {
				EntityID  int      `json:"entityID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := entity.DeleteEntityEmpower(siteID, actionAuth.(authority.ActionAuthSet), param.EntityID, param.Empower, param.EmpowerID, param.AuthList...); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("environment/station", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

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

		entityIDs := make([]int, 0)
		if idlist := c.Query("entityID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					entityIDs = append(entityIDs, id)
				}
			}
		}

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		if stationList, err := entity.GetStations(siteID, cids, entityIDs, c.Query("status"), c.Query("protocol"), c.Query("q"), stationIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			authList := make([]entity.IEntityAuth, 0)
			for _, ele := range stationList {
				authList = append(authList, ele)
			}
			authList, err := entity.FilterEntityAuthInterface(siteID, authList, actionAuth.(authority.ActionAuthSet), entity.ACTION_ENTITY_VIEW)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "stationList": authList})
			}
		}
	})

	authorized.GET("environment/station/status", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		stationStatus := ipcclient.RequestStationStatus(siteID, stationIDs...)
		c.Set("json", map[string]interface{}{"retCode": 0, "stationStatus": stationStatus})
	})

	authorized.GET("environment/station/status/history", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		stationStatusHistory := ipcclient.GetStationStatusHistory(siteID, stationIDs...)
		c.Set("json", map[string]interface{}{"retCode": 0, "stationStatusHistory": stationStatusHistory})
	})

	authorized.POST("environment/station/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return entity.MODULE_ENTITY, "station", c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")
			actionAuth, _ := c.Get("actionAuth")

			var param entity.Station

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			switch c.Param("method") {
			case "add":
				err = param.Add(siteID, actionAuth.(authority.ActionAuthSet))
			case "update":
				err = param.Update(siteID, actionAuth.(authority.ActionAuthSet))
			case "delete":
				err = param.Delete(siteID, actionAuth.(authority.ActionAuthSet))
				if err != nil {
					recent.ClearCache(siteID, param.ID)
				}
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				done, fail, timeout := ipcclient.NotifyStationChange(c.GetString("site"), param.ID)
				c.Set("loggingID", param.ID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0, "station": param, "done": done, "fail": fail, "timeout": timeout})
			}
		},
	)

	authorized.POST("environment/station/category/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return entity.MODULE_ENTITY, "station", c.Param("method") + "Category"
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			var param struct {
				StationID  int `json:"stationID"`
				CategoryID int `json:"categoryID"`
			}

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			switch c.Param("method") {
			case "bind":
				err = entity.AddStationCategory(siteID, param.StationID, param.CategoryID)
			case "unbind":
				err = entity.DeleteStationCategory(siteID, param.StationID, param.CategoryID)
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("loggingID", param.StationID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		},
	)

	authorized.GET("environment/station/log", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")
		stationID, err := strconv.Atoi(c.Query("stationID"))
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		lines, err := strconv.Atoi(c.Query("line"))
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if log, err := ipcclient.GetStationLog(siteID, stationID, lines); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "log": log})
		}
	})

	authorized.GET("environment/station/monitor", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		if stationMonitors, err := monitor.GetStationMonitors(siteID, actionAuth.(authority.ActionAuthSet), stationIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "stationMonitors": stationMonitors})
		}
	})

	authorized.GET("environment/monitor/module", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_VIEW, environment.ACTION_VIEW), func(c *gin.Context) {
		if monitorModule, err := monitor.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "monitorModule": monitorModule})
		}
	})

	authorized.POST("environment/monitor/module/edit/save", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT), logger(environment.MODULE_ENVIRONMENT, "site_module", "save"), func(c *gin.Context) {
		var param monitor.MonitorModule

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if err := param.Save(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			go ipcclient.NotifyModuleChange(c.GetString("site"))
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("environment/monitor", func(c *gin.Context) {
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

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorIDs = append(monitorIDs, id)
				}
			}
		}

		var types []int
		if str, exists := c.GetQuery("type"); exists && strings.TrimSpace(str) != "" {
			for _, idstr := range strings.Split(str, ",") {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					types = append(types, id)
				}
			}
		}

		if monitorList, err := monitor.GetMonitors(siteID, cids, types, monitorIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "monitorList": monitorList})
		}
	})

	authorized.POST("environment/monitor/edit/:method", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return monitor.MODULE_MONITOR, "monitor", c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			var param monitor.Monitor

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			switch c.Param("method") {
			case "add":
				err = param.Add(siteID)
			case "update":
				err = param.Update(siteID)
			case "delete":
				err = param.Delete(siteID)
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				done, fail, timout := ipcclient.NotifyMonitorChange(c.GetString("site"))
				c.Set("loggingID", param.ID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0, "done": done, "fail": fail, "timeout": timout})
			}
		},
	)

	authorized.POST("environment/monitor/category/:method", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return monitor.MODULE_MONITOR, "monitor", c.Param("method") + "Category"
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			var param struct {
				MonitorID  int `json:"monitorID"`
				CategoryID int `json:"categoryID"`
			}

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			switch c.Param("method") {
			case "bind":
				err = monitor.AddMonitorCategory(siteID, param.MonitorID, param.CategoryID)
			case "unbind":
				err = monitor.DeleteMonitorCategory(siteID, param.MonitorID, param.CategoryID)
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("loggingID", param.MonitorID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		},
	)

	authorized.GET("environment/monitor/code/template", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if templateList, err := monitor.GetMonitorCodeTemplates(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "templateList": templateList})
			}
		}
	})

	authorized.POST("environment/monitor/code/template/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param monitor.MonitorCodeTemplate

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "add":
			err = param.Add(siteID)
		case "update":
			err = param.Update(siteID)
		case "delete":
			err = param.Delete(siteID)
		default:
			c.AbortWithError(404, errors.New("invalid method"))
			return
		}

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "template": param})
		}
	})

	authorized.GET("environment/monitor/code", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		monitorID, _ := strconv.Atoi(c.Query("monitorID"))

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		if monitorCodeList, err := monitor.GetMonitorCodes(siteID, monitorID, c.Query("q"), stationIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			authList := make([]entity.IEntityStationAuth, 0)
			for _, ele := range monitorCodeList {
				authList = append(authList, ele)
			}
			authList, err := entity.FilterEntityStationAuthInterface(siteID, authList, actionAuth.(authority.ActionAuthSet), entity.ACTION_ENTITY_VIEW)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "monitorCodeList": authList})
			}
		}
	})

	authorized.POST("environment/monitor/code/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return monitor.MODULE_MONITOR, "monitorCode", c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			actionAuth, _ := c.Get("actionAuth")

			var param monitor.MonitorCode

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			switch c.Param("method") {
			case "add":
				err = param.Add(siteID, actionAuth.(authority.ActionAuthSet))
			case "update":
				err = param.Update(siteID, actionAuth.(authority.ActionAuthSet))
			case "delete":
				err = param.Delete(siteID, actionAuth.(authority.ActionAuthSet))
				if err != nil && param.StationID > 0 {
					recent.ClearCache(siteID, param.StationID)
				}
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				done, fail, timeout := ipcclient.NotifyMonitorCodeChange(c.GetString("site"))
				c.Set("loggingID", param.ID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0, "done": done, "fail": fail, "timeout": timeout})
			}
		},
	)

	authorized.GET("environment/monitor/limit/template", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if templateList, err := monitor.GetMonitorLimitTemplates(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "templateList": templateList})
			}
		}
	})

	authorized.POST("environment/monitor/limit/template/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param monitor.MonitorLimitTemplate

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "add":
			err = param.Add(siteID)
		case "update":
			err = param.Update(siteID)
		case "delete":
			err = param.Delete(siteID)
		default:
			c.AbortWithError(404, errors.New("invalid method"))
			return
		}

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "template": param})
		}
	})

	authorized.GET("environment/monitor/limit", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				id, err := strconv.Atoi(idstr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				monitorIDs = append(monitorIDs, id)
			}
		}

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				id, err := strconv.Atoi(idstr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				stationIDs = append(stationIDs, id)
			}
		}

		flags := make([]string, 0)
		if list := c.Query("flag"); strings.TrimSpace(list) != "" {
			flags = strings.Split(list, ",")
		}

		if monitorFlagLimitList, err := monitor.GetFlagLimits(siteID, stationIDs, monitorIDs, flags); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			authList := make([]entity.IEntityStationAuth, 0)
			for _, ele := range monitorFlagLimitList {
				authList = append(authList, ele)
			}
			authList, err := entity.FilterEntityStationAuthInterface(siteID, authList, actionAuth.(authority.ActionAuthSet), entity.ACTION_ENTITY_VIEW)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "monitorFlagLimitList": authList})
			}
		}
	})

	authorized.POST("environment/monitor/limit/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return monitor.MODULE_MONITOR, "monitorFlagLimit", c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			actionAuth, _ := c.Get("actionAuth")

			var param monitor.FlagLimit

			err := c.ShouldBindJSON(&param)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			switch c.Param("method") {
			case "add":
				err = param.Add(siteID, actionAuth.(authority.ActionAuthSet))
			case "update":
				err = param.Update(siteID, actionAuth.(authority.ActionAuthSet))
			case "delete":
				err = param.Delete(siteID, actionAuth.(authority.ActionAuthSet))
			default:
				c.AbortWithError(404, errors.New("invalid method"))
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				done, fail, timeout := ipcclient.NotifyFlagLimitChange(c.GetString("site"))
				c.Set("loggingID", param.ID)
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0, "done": done, "fail": fail, "timeout": timeout})
			}
		})

	authorized.GET("environment/data/module", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if dataModule, err := data.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "dataModule": dataModule})
		}
	})

	authorized.POST("environment/data/module/edit/save", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT), logger(data.MODULE_DATA, "site_module", "save"), func(c *gin.Context) {
		var param data.DataModule

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

	authorized.GET("environment/data/range/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {

		if rangeList, err := data.FetchableTables(c.GetString("site"), c.Param("dataType")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "rangeList": rangeList})
		}

	})

	authorized.POST("environment/data/rotate", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		data.TriggerRotation(siteID, true)
		c.Set("json", map[string]interface{}{"retCode": 0})

	})

	authorized.POST("environment/data/activateArchive/:dataType", checkAuth(environment.MODULE_ENVIRONMENT, environment.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			Table string `json:"table"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if err := data.ActivateArchive(siteID, c.Param("dataType"), param.Table); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("environment/data/byTime/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth.(authority.ActionAuthSet), stationIDs, entity.ACTION_ENTITY_VIEW)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		for _, sid := range stationIDs {
			if !filtered[sid] {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": fmt.Sprintf("无权限查看【%d】", sid)})
				return
			}
		}

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorIDs = append(monitorIDs, id)
				}
			}
		}

		monitorCodeIDs := make([]int, 0)
		if idlist := c.Query("monitorCodeID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorCodeIDs = append(monitorCodeIDs, id)
				}
			}
		}

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}

		var criterias data.Criterias
		if cs, exists := c.GetQuery("criteria"); exists && cs != "" {
			if err := json.Unmarshal([]byte(cs), &criterias); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		withOriginData, _ := strconv.ParseBool(c.Query("withOriginData"))
		withReiewed, _ := strconv.ParseBool(c.Query("withReviewed"))

		if timeDateList, total, err := data.GetDataByTime(siteID, dataType, stationIDs, criterias, beginTime, endTime, withOriginData, withReiewed, c.Query("order"), pageNo, pageSize, monitorCodeIDs, monitorIDs); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "timeDataList": timeDateList, "total": total})
		}
	})

	authorized.GET("environment/data/count/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth.(authority.ActionAuthSet), stationIDs, entity.ACTION_ENTITY_VIEW)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		for _, sid := range stationIDs {
			if !filtered[sid] {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "无权限"})
				return
			}
		}

		monitorCodeIDs := make([]int, 0)
		if idlist := c.Query("monitorCodeID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorCodeIDs = append(monitorCodeIDs, id)
				}
			}
		}

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorIDs = append(monitorIDs, id)
				}
			}
		}

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}

		var criterias data.Criterias
		if cs, exists := c.GetQuery("criteria"); exists && cs != "" {
			if err := json.Unmarshal([]byte(cs), &criterias); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		groupByTime, _ := strconv.ParseBool(c.Query("groupByTime"))
		groupByStation, _ := strconv.ParseBool(c.Query("groupByStation"))
		groupByMonitor, _ := strconv.ParseBool(c.Query("groupByMonitor"))
		groupByFlag, _ := strconv.ParseBool(c.Query("groupByFlag"))

		var flags []string
		if flag, exists := c.GetQuery("flag"); exists {
			flags = strings.Split(flag, ",")
		}

		if count, err := data.CountData(siteID, dataType, stationIDs, monitorIDs, monitorCodeIDs, beginTime, endTime, flags, criterias, groupByTime, groupByStation, groupByMonitor, groupByFlag); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "count": count})
		}
	})

	authorized.GET("environment/data/list/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth.(authority.ActionAuthSet), stationIDs, entity.ACTION_ENTITY_VIEW)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		for _, sid := range stationIDs {
			if !filtered[sid] {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "无权限"})
				return
			}
		}

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorIDs = append(monitorIDs, id)
				}
			}
		}

		monitorCodeIDs := make([]int, 0)
		if idlist := c.Query("monitorCodeID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorCodeIDs = append(monitorCodeIDs, id)
				}
			}
		}

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}

		var flags []string
		if flag, exists := c.GetQuery("flag"); exists {
			flags = strings.Split(flag, ",")
		}

		var criterias data.Criterias
		if cs, exists := c.GetQuery("criteria"); exists && cs != "" {
			if err := json.Unmarshal([]byte(cs), &criterias); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		if dataList, err := data.GetData(siteID, dataType, stationIDs, monitorIDs, monitorCodeIDs, criterias, beginTime, endTime, flags); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "dataList": dataList})
		}
	})

	authorized.GET("environment/data/vacancy/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth.(authority.ActionAuthSet), stationIDs, entity.ACTION_ENTITY_VIEW)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		for _, sid := range stationIDs {
			if !filtered[sid] {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "无权限"})
				return
			}
		}

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}

		if dataVacancies, err := data.GetDataVacancy(siteID, dataType, stationIDs, beginTime, endTime); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "dataVacancies": dataVacancies})
		}
	})

	authorized.GET("environment/data/recent/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		if recentData, err := recent.GetRecentData(siteID, actionAuth.(authority.ActionAuthSet), dataType, stationIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "recentData": recentData})
		}
	})

	authorized.GET("environment/data/quality/:target", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW), func(c *gin.Context) {

		siteID := c.GetString("site")
		actionAuth, _ := c.Get("actionAuth")

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}
		switch c.Param("target") {
		case "station":
			stationIDs := make([]int, 0)
			if idlist := c.Query("stationID"); idlist != "" {
				parts := strings.Split(idlist, ",")
				for _, idstr := range parts {
					if id, err := strconv.Atoi(idstr); err != nil {
						c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
						return
					} else {
						stationIDs = append(stationIDs, id)
					}
				}
			}
			if stationDataQualities, err := stats.GetStationDataQuality(siteID, actionAuth.(authority.ActionAuthSet), &beginTime, &endTime, stationIDs...); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "stationDataQualities": stationDataQualities})
			}
		case "monitor":
			stationID, _ := strconv.Atoi(c.Query("stationID"))
			if monitorDataQualities, err := stats.GetMonitorDataQuality(siteID, actionAuth.(authority.ActionAuthSet), stationID, &beginTime, &endTime); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "monitorDataQualities": monitorDataQualities})
			}
		case "history":
			stationIDs := make([]int, 0)
			if idlist := c.Query("stationID"); idlist != "" {
				parts := strings.Split(idlist, ",")
				for _, idstr := range parts {
					if id, err := strconv.Atoi(idstr); err != nil {
						c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
						return
					} else {
						stationIDs = append(stationIDs, id)
					}
				}
			}
			if historyStationDataQualities, err := stats.GetHistoryStats(siteID, actionAuth.(authority.ActionAuthSet), stats.HISTORY_STATS_DATA_QUALITY, stats.HISTORY_STATS_INTERVAL_DAILY, beginTime, endTime, stationIDs...); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "historyStationDataQualities": historyStationDataQualities})
			}
		default:
			c.AbortWithStatus(404)
		}
	})

	authorized.POST("environment/data/massage/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		dataType := c.Param("dataType")

		stationIDs := make([]int, 0)
		if idlist := c.Query("stationID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					stationIDs = append(stationIDs, id)
				}
			}
		}

		monitorIDs := make([]int, 0)
		if idlist := c.Query("monitorID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorIDs = append(monitorIDs, id)
				}
			}
		}

		monitorCodeIDs := make([]int, 0)
		if idlist := c.Query("monitorCodeID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					monitorCodeIDs = append(monitorCodeIDs, id)
				}
			}
		}

		var beginTime, endTime time.Time
		if c.Query("beginTime") != "" {
			ts, err := util.ParseDateTime(c.Query("beginTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if c.Query("endTime") != "" {
			ts, err := util.ParseDateTime(c.Query("endTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}

		flag := c.Query("flag")

		restoreBeforeProcess, _ := strconv.ParseBool(c.Query("restoreBeforeProcess"))
		skipNoOrigins, _ := strconv.ParseBool(c.Query("skipNoOrigins"))

		var processor dataprocess.DataProcessors
		if err := c.ShouldBindJSON(&processor); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if count, err := operation.Massage(siteID, actionAuth.(authority.ActionAuthSet), dataType, stationIDs, monitorIDs, monitorCodeIDs, beginTime, endTime, flag, restoreBeforeProcess, skipNoOrigins, processor); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "count": count, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "count": count})
		}
	})

	authorized.POST("environment/data/upload/excel", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")
		actionAuth, _ := c.Get("actionAuth")

		save, _ := strconv.ParseBool(c.Query("save"))

		form, err := c.MultipartForm()
		if err != nil {
			log.Println("error not multipart-form: ", err)
			c.String(500, "not multipart-form")
			return
		}

		if timeDataList, err := upload.UploadExcel(siteID, actionAuth.(authority.ActionAuthSet), form.File, form.Value); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			if save {
				for _, td := range timeDataList {
					for _, list := range td.Data {
						for _, d := range list {
							if err := data.AddUpdate(siteID, d); err != nil {
								c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
							}
						}
					}
				}
				c.Set("json", map[string]interface{}{"retCode": 0, "count": len(timeDataList)})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "timeDataList": timeDataList})
			}
		}
	})

	authorized.GET("environment/data/uploader", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		if dataUploaderList, err := upload.GetDataUploader(siteID, c.Query("q")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "dataUploaderList": dataUploaderList})
		}
	})

	authorized.POST("environment/data/uploader/edit/:method", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param upload.DataUploader

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "add":
			err = param.Add(siteID)
		case "update":
			err = param.Update(siteID)
		case "delete":
			err = param.Delete(siteID)
		default:
			c.AbortWithError(404, errors.New("invalid method"))
			return
		}

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.POST("environment/data/edit/:method/:dataType", checkAuth(entity.MODULE_ENTITY, entity.ACTION_ADMIN_EDIT, entity.ACTION_ENTITY_EDIT),
		loggerFunc(func(c *gin.Context) (string, string, string) {
			return data.MODULE_DATA, c.Param("dataType"), c.Param("method")
		}),
		func(c *gin.Context) {
			siteID := c.GetString("site")

			actionAuth, _ := c.Get("actionAuth")

			var param data.IData

			switch c.Param("dataType") {
			case data.REAL_TIME:
				rtd := new(data.RealTimeData)
				err := c.ShouldBindJSON(&rtd)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				param = rtd
			case data.MINUTELY:
				minutely := new(data.MinutelyData)
				err := c.ShouldBindJSON(&minutely)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				param = minutely
			case data.HOURLY:
				hourly := new(data.HourlyData)
				err := c.ShouldBindJSON(&hourly)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				param = hourly
			case data.DAILY:
				daily := new(data.DailyData)
				err := c.ShouldBindJSON(&daily)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				param = daily
			default:
				c.AbortWithError(404, errors.New("invalid data type"))
				return
			}

			filtered, err := entity.FilterEntityStationAuth(siteID, actionAuth.(authority.ActionAuthSet), []int{param.GetStationID()}, entity.ACTION_ENTITY_EDIT)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if !filtered[param.GetStationID()] {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "无权限"})
				return
			}

			switch c.Param("method") {
			case "add":
				err = data.AddUpdate(siteID, param)
			case "modify":
				fields := strings.Split(c.Query("field"), ",")
				_, err = operation.Modify(siteID, param, fields, c.GetInt("uid"))
			case "delete":
				err = data.Delete(siteID, param)
			default:
				c.AbortWithError(404, errors.New("invalid method"))
				return
			}

			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				recent.ClearCache(siteID, param.GetStationID())
				c.Set("loggingID", param.GetID())
				c.Set("loggingPayload", param)
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		},
	)

	authorized.GET("environment/subscription/module", checkAuth(subscription.MODULE_SUBSCRIPTION, subscription.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if subscriptionModule, err := subscription.GetModule(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "subscriptionModule": subscriptionModule})
		}
	})

	authorized.POST("environment/subscription/module/edit/save", checkAuth(subscription.MODULE_SUBSCRIPTION, subscription.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param subscription.SubscriptionModule

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		err = param.Save(siteID)

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.GET("environment/externalsource/HNAQIPublish/module", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if HNAQIPublishModule, err := externalsource.GetHNAQIPublishModule(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "HNAQIPublishModule": HNAQIPublishModule})
		}
	})

	authorized.POST("environment/externalsource/HNAQIPublish/module/edit/save", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param externalsource.HNAQIPublishModule

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		err = param.Save(siteID)

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.POST("environment/externalsource/HNAQIPublish/initLogin", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		if status, contentType, data, err := externalsource.InitLoginHNAQIPublish(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Data(status, contentType, data)
		}
	})

	authorized.POST("environment/externalsource/HNAQIPublish/login", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			Usr     string `json:"usr"`
			Pw      string `json:"pw"`
			ImgCode string `json:"imgCode"`
		}

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		if status, contentType, data, err := externalsource.LoginHNAQIPublish(siteID, param.Usr, param.Pw, param.ImgCode); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Data(status, contentType, data)
		}
	})

	authorized.GET("environment/externalsource/HNAQIPublish/stationTree", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		if status, contentType, data, err := externalsource.GetHNAQIPublishStationTree(siteID, c.Query("type")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Data(status, contentType, data)
		}
	})

	authorized.GET("environment/externalsource/HNAQIPublish/:rptType/:dataType", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var stations []string
		if str, exists := c.GetQuery("stn"); exists && str != "" {
			stations = strings.Split(str, ",")
		} else {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "need stn"})
			return
		}

		var beginTime, endTime time.Time
		var err error
		if str, exists := c.GetQuery("beginTime"); exists && str != "" {
			beginTime, err = util.ParseDateTime(str)
		} else {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "need beginTime"})
			return
		}
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		if str, exists := c.GetQuery("endTime"); exists && str != "" {
			endTime, err = util.ParseDateTime(str)
		} else {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "need beginTime"})
			return
		}
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("rptType") {
		case "rptData":
			if status, contentType, data, err := externalsource.GetHNAQIPublishRptData(siteID, c.Param("dataType"), stations, beginTime, endTime); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Data(status, contentType, data)
			}
		case "statsRptData":
			if status, contentType, data, err := externalsource.GetHNAQIPublishStatsData(siteID, c.Param("dataType"), stations, beginTime, endTime); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Data(status, contentType, data)
			}
		default:
			c.AbortWithError(404, errors.New("invalid rpt"))
			return
		}

	})

	authorized.POST("environment/externalsource/HNAQIPublish/sync/:rptType/:dataType", checkAuth(externalsource.MODULE_HNAQIPUBLISH, externalsource.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			SyncTime       util.Time `json:"syncTime"`
			TraceBackCount int       `json:"traceBackCount"`
		}

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("rptType") {
		case "rptData":
			if err := externalsource.SyncHNAQIPublishRptData(siteID, c.Param("dataType"), time.Time(param.SyncTime), param.TraceBackCount); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "statsRptData":
			if err := externalsource.SyncHNAQIPublishStatsData(siteID, c.Param("dataType"), time.Time(param.SyncTime), param.TraceBackCount); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid rpt"))
			return
		}

	})

}
