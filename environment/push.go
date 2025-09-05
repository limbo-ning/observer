package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"obsessiontech/environment/push"

	"github.com/gin-gonic/gin"
)

func loadPush() {
	authorized.GET("push/subscription", checkAuth(push.MODULE_PUSH, push.ACTION_ADMIN_SUBSCRIBE, push.ACTION_SUBSCRIBE), func(c *gin.Context) {
		siteID := c.GetString("site")

		subscriberType := c.Query("subscriberType")
		var subscriberID int

		if idstr, exists := c.GetQuery("subscriberID"); exists && strings.TrimSpace(idstr) != "" {
			var err error
			subscriberID, err = strconv.Atoi(idstr)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		} else {
			if subscriberType == "" {
				subscriberType = "user"
				subscriberID = c.GetInt("uid")
			}
		}

		var ext map[string][]any
		if str, exists := c.GetQuery("ext"); exists && strings.TrimSpace(str) != "" {
			if err := json.Unmarshal([]byte(str), &ext); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		if subscriptionList, err := push.GetSubscriptionList(siteID, subscriberType, subscriberID, c.Query("subscriptionType"), c.Query("push"), ext); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "subscriptionList": subscriptionList})
		}
	})

	authorized.POST("push/subscription/edit/:method", checkAuth(push.MODULE_PUSH, push.ACTION_ADMIN_SUBSCRIBE, push.ACTION_SUBSCRIBE), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param push.Subscription

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		if param.SubscriberType == "" {
			param.SubscriberType = "user"
		}
		if param.SubscriberType == "user" {
			param.SubscriberID = c.GetInt("uid")
		}
		switch c.Param("method") {
		case "add":
			err = param.Add(siteID, nil)
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
}
