package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/logging"

	"github.com/gin-gonic/gin"
)

var logger = func(moduleID, source, action string) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Next()
		doLog(c, moduleID, source, action)
	}
}

var loggerFunc = func(checkFunc func(c *gin.Context) (string, string, string)) func(*gin.Context) {
	return func(c *gin.Context) {
		moduleID, source, action := checkFunc(c)
		c.Next()
		doLog(c, moduleID, source, action)
	}
}

func doLog(c *gin.Context, moduleID, source, action string) {
	loggingID, exists := c.Get("loggingID")
	if !exists {
		return
	}
	payload, _ := c.Get("loggingPayload")

	siteID := c.GetString("site")
	uid := c.GetInt("uid")

	var sourceID string

	switch v := loggingID.(type) {
	case int:
		sourceID = fmt.Sprintf("%d", v)
	case string:
		sourceID = v
	default:
		log.Println("error incorrect loggingID:", loggingID)
		return
	}

	go logging.Log(siteID, uid, moduleID, source, sourceID, action, payload)
}

func loadLogging() {

	authorized.GET("logging/module", checkAuth(logging.MODULE_LOGGING, logging.ACTION_ADMIN_VIEW_MODULE), func(c *gin.Context) {
		if module, err := logging.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "loggingModule": module})
		}
	})

	authorized.POST("logging/module/edit/save", checkAuth(logging.MODULE_LOGGING, logging.ACTION_ADMIN_EDIT_MODULE), func(c *gin.Context) {
		var param logging.LoggingModule

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

	authorized.GET("logging/logger/registrant", checkAuth(logging.MODULE_LOGGING, logging.ACTION_ADMIN_VIEW_MODULE), func(c *gin.Context) {
		if loggerList, err := logging.GetLoggerRegistrants(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "loggerList": loggerList})
		}
	})

	authorized.GET("logging/logger", checkAuth(logging.MODULE_LOGGING, logging.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if logger, err := logging.GetLogger(c.GetString("site"), c.Query("moduleID"), c.Query("source")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "logger": logger})
		}
	})

	authorized.GET("logging", checkAuth(logging.MODULE_LOGGING, logging.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		uid, _ := strconv.Atoi(c.Query("UID"))
		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

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

		if loggingList, userMap, total, err := logging.GetLoggings(siteID, c.Query("moduleID"), c.Query("source"), c.Query("sourceID"), c.Query("action"), uid, &beginTime, &endTime, pageNo, pageSize); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "loggingList": loggingList, "userMap": userMap, "total": total})
		}
	})

}
