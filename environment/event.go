package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/event"

	"github.com/gin-gonic/gin"
)

func loadEvent() {

	event.ScheduleEvents()

	authorized.GET("event/event", checkAuth(event.MODULE_EVENT, event.ACTION_ADMIN_VIEW, event.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		var subRelateID map[string]string
		if keys, exists := c.GetQueryArray("subRelateID"); exists {
			subRelateID = make(map[string]string)
			for _, key := range keys {
				subRelateID[key] = c.Query("subRelateID_" + key)
			}
		}

		empowerID := make([]string, 0)
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerID = strings.Split(str, ",")
		}

		eventID := make([]int, 0)
		if str, exists := c.GetQuery("eventID"); exists && strings.TrimSpace(str) != "" {
			for _, idstr := range strings.Split(str, ",") {
				id, err := strconv.Atoi(idstr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				eventID = append(eventID, id)
			}
		}

		var beginTime, endTime, effectTime time.Time
		if str, exists := c.GetQuery("beginTime"); exists && strings.TrimSpace(str) != "" {
			ts, err := util.ParseDateTime(str)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			beginTime = ts
		}
		if str, exists := c.GetQuery("endTime"); exists && strings.TrimSpace(str) != "" {
			ts, err := util.ParseDateTime(str)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			endTime = ts
		}
		if str, exists := c.GetQuery("effectTime"); exists && strings.TrimSpace(str) != "" {
			ts, err := util.ParseDateTime(str)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			effectTime = ts
		}

		if eventList, total, err := event.GetEvents(siteID, actionAuth.(authority.ActionAuthSet), c.Query("type"), c.Query("status"), &beginTime, &endTime, &effectTime, pageNo, pageSize, c.Query("mainRelateID"), subRelateID, c.Query("authType"), c.Query("empower"), empowerID, eventID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "eventList": eventList, "total": total})
		}
	})

	authorized.POST("event/event/edit/:method", checkAuth(event.MODULE_EVENT, event.ACTION_ADMIN_EDIT, event.ACTION_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param event.Event

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "add":
			if err := param.Add(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "event": param})
			}
		case "clone":
			if result, err := event.CloneEvent(siteID, actionAuth.(authority.ActionAuthSet), param.ID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "event": result})
			}
		case "delete":
			if err := param.Delete(siteID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.GET("event/scheduler", checkAuth(event.MODULE_EVENT, event.ACTION_ADMIN_VIEW, event.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		schedulerID := make([]int, 0)
		if str, exists := c.GetQuery("schedulerID"); exists && strings.TrimSpace(str) != "" {
			for _, idstr := range strings.Split(str, ",") {
				id, err := strconv.Atoi(idstr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				schedulerID = append(schedulerID, id)
			}
		}

		if schedulerList, total, err := event.GetSchedulers(siteID, c.Query("type"), c.Query("mainRelateID"), c.Query("q"), pageNo, pageSize, schedulerID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "schedulerList": schedulerList, "total": total})
		}
	})

	authorized.POST("event/scheduler/edit/:method", checkAuth(event.MODULE_EVENT, event.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param event.Scheduler

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
