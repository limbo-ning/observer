package main

import (
	"errors"
	"strconv"
	"strings"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/role"
	"obsessiontech/environment/user"

	"github.com/gin-gonic/gin"
)

func loadRole() {

	authorized.GET("role/module", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		if roleModule, err := role.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "roleModule": roleModule})
		}
	})

	authorized.POST("role/module/edit/save", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param role.RoleModule

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

	sites.GET("role", func(c *gin.Context) {
		roleIDs := make([]int, 0)
		roleID := c.Query("roleID")
		if roleID != "" {
			for _, idStr := range strings.Split(roleID, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				roleIDs = append(roleIDs, id)
			}
		}

		if roleList, err := role.GetRoles(c.GetString("site"), c.Query("series"), roleIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "roleList": roleList})
		}
	})

	sites.GET("role/point", func(c *gin.Context) {
		pointIDs := make([]int, 0)
		pointID := c.Query("pointID")
		if pointID != "" {
			for _, idStr := range strings.Split(pointID, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				pointIDs = append(pointIDs, id)
			}
		}

		if pointList, milestones, err := role.GetPoints(c.GetString("site"), pointIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "pointList": pointList, "milestones": milestones})
		}
	})

	authorized.GET("role/authority/template", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_AUTHORITY, role.ACTION_ADMIN_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		if templateList, err := role.GetRoleAuthorityTemplates(siteID); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "templateList": templateList})
			}
		}
	})

	authorized.POST("role/authority/template/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_AUTHORITY), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param role.RoleAuthorityTemplate

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

	authorized.GET("role/authority", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_AUTHORITY, role.ACTION_ADMIN_VIEW), func(c *gin.Context) {

		var moduleIDs []string
		if ids, exists := c.GetQuery("moduleID"); exists && strings.TrimSpace(ids) != "" {
			moduleIDs = strings.Split(ids, ",")
		}

		roleID := make([]int, 0)
		if roleIDs, exists := c.GetQuery("roleID"); exists && strings.TrimSpace(roleIDs) != "" {
			for _, idStr := range strings.Split(roleIDs, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				roleID = append(roleID, id)
			}
		}

		if roleAuthorities, err := role.GetRoleAuthority(c.GetString("site"), moduleIDs, roleID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "roleAuthorities": roleAuthorities})
		}
	})

	authorized.GET("role/user", func(c *gin.Context) {
		userRoles, userRoleExpires, err := role.GetUserRoleAuthority(c.GetString("site"), c.GetInt("uid"))
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		result := make(map[string]interface{})

		roleMap := userRoles[c.GetInt("uid")]
		if roleMap == nil {
			roleMap = make(map[int][]*role.RoleAuthority)
		}

		expires := userRoleExpires[c.GetInt("uid")]
		if expires == nil {
			expires = make(map[int]int)
		}

		result["userRoles"] = roleMap
		result["userRoleExpires"] = expires

		withPoints, _ := strconv.ParseBool(c.Query("withPoints"))
		if withPoints {
			userPoints, err := role.GetUserPoints(c.GetString("site"), c.GetInt("uid"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			pointMap := userPoints[c.GetInt("uid")]
			if pointMap == nil {
				pointMap = make(map[int]int)
			}
			result["userPoints"] = pointMap
		}

		result["retCode"] = 0

		c.Set("json", result)
	})

	authorized.GET("role/user/role", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_VIEW, role.ACTION_ADMIN_VIEW_USERROLE), func(c *gin.Context) {
		uids := make([]int, 0)
		if c.Query("UID") != "" {
			for _, idStr := range strings.Split(c.Query("UID"), ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uids = append(uids, id)
			}
		}
		if userRoles, userRoleExpires, err := role.GetUserRoleAuthority(c.GetString("site"), uids...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userRoles": userRoles, "userRoleExpires": userRoleExpires})
		}
	})

	authorized.GET("role/user/checkAuth", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_VIEW, role.ACTION_ADMIN_VIEW_USERROLE), func(c *gin.Context) {
		uids := make([]int, 0)
		if c.Query("UID") != "" {
			for _, idStr := range strings.Split(c.Query("UID"), ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uids = append(uids, id)
			}
		}
		if userChecked, err := role.CheckUserRoleAuthority(c.GetString("site"), c.Query("moduleID"), c.Query("action"), c.Query("roleType"), uids...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userChecked": userChecked})
		}
	})

	authorized.GET("role/role/user", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_VIEW, role.ACTION_ADMIN_VIEW_USERROLE), func(c *gin.Context) {
		roleIDs := make([]int, 0)
		roleID := c.Query("roleID")

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		if roleID != "" {
			for _, idStr := range strings.Split(roleID, ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				roleIDs = append(roleIDs, id)
			}
		}

		if userList, total, err := user.GetRoleUsers(c.GetString("site"), c.Query("roleType"), c.Query("match"), c.Query("q"), c.Query("status"), pageNo, pageSize, c.Query("orderBy"), roleIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userList": userList, "total": total})
		}
	})

	authorized.GET("role/user/point", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_VIEW, role.ACTION_ADMIN_VIEW_USERROLE), func(c *gin.Context) {
		uids := make([]int, 0)
		if c.Query("UID") != "" {
			for _, idStr := range strings.Split(c.Query("UID"), ",") {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				uids = append(uids, id)
			}
		}
		if userPoints, err := role.GetUserPoints(c.GetString("site"), uids...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "userPoints": userPoints})
		}
	})

	authorized.POST("role/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param role.Role
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
		case "sync":
			if err := param.SyncAuth(siteID); err != nil {
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

	authorized.POST("role/point/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param role.Point
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

	authorized.POST("role/point/milestone/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param role.PointMilestone
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

	authorized.GET("role/authority/grantable", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_AUTHORITY), func(c *gin.Context) {
		actionAuth, _ := c.Get("actionAuth")
		if authorityList, err := role.GetGrantableAuthorities(c.GetString("site"), c.GetInt("uid"), actionAuth.(authority.ActionAuthSet)); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "authorityList": authorityList})
		}
	})

	authorized.POST("role/authority/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_AUTHORITY), func(c *gin.Context) {

		var param struct {
			RoleID   int    `json:"roleID"`
			ModuleID string `json:"moduleID"`
			Action   string `json:"action"`
			RoleType string `json:"roleType"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "grant":
			if err := role.GrantRoleModule(c.GetString("site"), param.RoleID, param.ModuleID, param.Action, param.RoleType, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "withdraw":
			if err := role.WithdrawRoleModule(c.GetString("site"), param.RoleID, param.ModuleID, param.Action, param.RoleType, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		}
	})

	authorized.POST("role/user/edit/:method", checkAuth(role.MODULE_ROLE, role.ACTION_GRANT_ALL, role.ACTION_GRANT_ROLE), func(c *gin.Context) {

		var param struct {
			UID     int `json:"UID"`
			RoleID  int `json:"roleID"`
			Expires int `json:"expires"`
		}

		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		actionAuth, _ := c.Get("actionAuth")

		switch c.Param("method") {
		case "grant":
			if err := role.GrantUserRole(c.GetString("site"), param.UID, param.RoleID, param.Expires, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "withdraw":
			if err := role.WithdrawUserRole(c.GetString("site"), param.UID, param.RoleID, actionAuth.(authority.ActionAuthSet)); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		}
	})
}
