package main

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/user"
	"obsessiontech/environment/user/auth"
	"obsessiontech/environment/user/auth/engine"

	"github.com/gin-gonic/gin"
)

func loadUser() {

	sites.GET("user/auth/method", func(c *gin.Context) {
		siteID := c.GetString("site")

		authMethods := engine.GetAuthMethod(siteID)

		c.Set("json", map[string]interface{}{"retCode": 0, "authMethods": authMethods})
	})

	sites.POST("user/register/:method", func(c *gin.Context) {
		siteID := c.GetString("site")
		postData, err := c.GetRawData()
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		cookie, ret, err := auth.Register(siteID, c.GetString("requestID"), c.ClientIP(), c.Param("method"), postData)
		if err != nil {
			if ret == nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				if _, exists := ret["retCode"]; !exists {
					ret["retCode"] = 500
				}
				if _, exists := ret["retMsg"]; !exists {
					ret["retMsg"] = err.Error()
				}
				c.Set("json", ret)
			}
			return
		}
		if cookie != "" {
			c.Set("session", cookie)
			c.Set("json", map[string]interface{}{"retCode": 0})
		} else {
			c.Set("json", ret)
		}
	})

	sites.POST("user/login/:method", func(c *gin.Context) {
		siteID := c.GetString("site")
		postData, err := c.GetRawData()
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		cookie, ret, err := auth.Login(siteID, c.GetString("requestID"), c.ClientIP(), c.Param("method"), postData)
		if err != nil {
			if ret == nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				if _, exists := ret["retCode"]; !exists {
					ret["retCode"] = 500
				}
				if _, exists := ret["retMsg"]; !exists {
					ret["retMsg"] = err.Error()
				}
				c.Set("json", ret)
			}
			return
		}
		if cookie != "" {
			log.Println("login set cookie: ", siteID+"-"+Config.CookieAuthUserName, cookie, Config.CookieDomain)
			c.Set("session", cookie)
			if ret == nil {
				c.Set("json", map[string]interface{}{"retCode": 0})
			} else {
				if _, exist := ret["retCode"]; !exist {
					ret["retCode"] = 0
				}
				c.Set("json", ret)
			}
		} else {
			if _, exist := ret["retCode"]; !exist {
				ret["retCode"] = 1
			}
			c.Set("json", ret)
		}
	})

	authorized.GET("user/isLogined", func(c *gin.Context) {
		c.Set("json", map[string]interface{}{"retCode": 0})
	})

	authorized.GET("user/info", func(c *gin.Context) {
		siteID := c.GetString("site")
		if userInfo, err := user.GetUser(siteID, "id", c.GetInt("uid")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userInfo": userInfo})
		}
	})

	authorized.POST("user/logout", func(c *gin.Context) {
		siteID := c.GetString("site")
		auth.Logout(siteID, c.GetInt("uid"))
		c.Set("session", "expired")
		c.Set("json", map[string]interface{}{"retCode": 0})
	})

	authorized.POST("user/deleteAccount", func(c *gin.Context) {
		siteID := c.GetString("site")

		var u user.User
		u.UserID = c.GetInt("uid")

		if err := u.Delete(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		c.Set("session", "expired")
		c.Set("json", map[string]interface{}{"retCode": 0})
	})

	authorized.POST("user/bind/:method", checkAuthSafe(user.MODULE_USER, user.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")
		postData, err := c.GetRawData()
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		UID := c.GetInt("uid")

		if str, exists := c.GetQuery("UID"); exists {
			checked := false

			actionAuth, _ := c.Get("actionAuth")
			for _, a := range actionAuth.(authority.ActionAuthSet) {
				switch a.Action {
				case user.ACTION_ADMIN_EDIT:
					checked = true
				default:
				}
				if checked {
					break
				}
			}

			if checked {
				UID, _ = strconv.Atoi(str)
			}
		}

		if ret, err := auth.Bind(siteID, c.Param("method"), UID, postData); err != nil {
			if ret == nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				if _, exists := ret["retCode"]; !exists {
					ret["retCode"] = 500
				}
				c.Set("json", ret)
			}
		} else if ret != nil {
			if _, exists := ret["retCode"]; !exists {
				ret["retCode"] = 0
			}
			c.Set("json", ret)
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.POST("user/unbind/:method", checkAuthSafe(user.MODULE_USER, user.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")
		postData, err := c.GetRawData()
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		UID := c.GetInt("uid")

		if str, exists := c.GetQuery("UID"); exists {
			checked := false

			actionAuth, _ := c.Get("actionAuth")
			for _, a := range actionAuth.(authority.ActionAuthSet) {
				switch a.Action {
				case user.ACTION_ADMIN_EDIT:
					checked = true
				}

				if checked {
					break
				}
			}

			if checked {
				UID, _ = strconv.Atoi(str)
			}
		}

		if ret, err := auth.UnBind(siteID, c.Param("method"), UID, postData); err != nil {
			if ret == nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				if _, exists := ret["retCode"]; !exists {
					ret["retCode"] = 500
				}
				c.Set("json", ret)
			}
		} else if ret != nil {
			if _, exists := ret["retCode"]; !exists {
				ret["retCode"] = 0
			}
			c.Set("json", ret)
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	authorized.POST("user/edit/:method", checkAuthFunc(func(c *gin.Context) (string, []string) {
		switch c.Param("method") {
		case "create":
			fallthrough
		case "activate":
			fallthrough
		case "inactivate":
			return user.MODULE_USER, []string{user.ACTION_ADMIN_EDIT}
		default:
			return user.MODULE_USER, []string{user.ACTION_ADMIN_EDIT, user.ACTION_EDIT}
		}
	}), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param user.User
		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}
		switch c.Param("method") {
		case "create":
			if err := auth.Create(siteID, c.GetString("requestID"), &param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "userInfo": param})
			}
		case "activate":
			if err := param.Activate(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "userInfo": param})
			}
		case "deactivate":
			if err := param.Deactivate(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "userInfo": param})
			}
		case "updateInfo":
			actionAuth, _ := c.Get("actionAuth")

			if err := auth.UpdateInfo(siteID, c.GetString("requestID"), actionAuth.(authority.ActionAuthSet), &param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "userInfo": param})
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

	sites.GET("user/brief", func(c *gin.Context) {
		siteID := c.GetString("site")

		var uidList []int
		if str := c.Query("UID"); str != "" {
			for _, idStr := range strings.Split(str, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uidList = append(uidList, id)
			}
		}

		if userList, err := user.GetUserBrief(siteID, uidList...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userList": userList})
		}
	})

	authorized.GET("user/user/info", func(c *gin.Context) {
		siteID := c.GetString("site")

		var uidList []int
		if str := c.Query("UID"); str != "" {
			for _, idStr := range strings.Split(str, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uidList = append(uidList, id)
			}
		}

		if userList, err := user.GetUserInfo(siteID, uidList...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userList": userList})
		}
	})

	authorized.GET("user", checkAuth(user.MODULE_USER, user.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		var uidList []int
		if c.Query("UID") != "" {
			list := strings.Split(c.Query("UID"), ",")
			for _, idStr := range list {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uidList = append(uidList, id)
			}
		}

		if userList, total, err := user.GetUsers(siteID, c.Query("match"), c.Query("q"), c.Query("status"), pageNo, pageSize, c.Query("orderBy"), uidList...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userList": userList, "total": total})
		}
	})

	authorized.GET("user/module", checkAuth(user.MODULE_USER, user.ACTION_ADMIN_VIEW_MODULE), func(c *gin.Context) {
		if userModule, err := user.GetUserModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userModule": userModule})
		}
	})

	authorized.POST("user/module/edit/save", checkAuth(user.MODULE_USER, user.ACTION_ADMIN_EDIT_MODULE), func(c *gin.Context) {
		var param user.UserModule

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

}
