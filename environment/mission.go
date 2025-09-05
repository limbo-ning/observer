package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/mission"

	"github.com/gin-gonic/gin"
)

func loadMission() {

	authorized.GET("mission/module", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW), func(c *gin.Context) {
		if missionModule, err := mission.GetModule(c.GetString("site")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "missionModule": missionModule})
		}
	})

	authorized.POST("mission/module/edit/save", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		var param mission.MissionModule

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

	sites.GET("mission/mission", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		skipCheck, _ := strconv.ParseBool("skipCheck")

		missionID := make([]int, 0)
		if idlist := c.Query("missionID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					missionID = append(missionID, id)
				}
			}
		}

		var relateID map[string]string
		if idlist, exists := c.GetQuery("relateID"); exists && strings.TrimSpace(idlist) != "" {
			if err := json.Unmarshal([]byte(idlist), &relateID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		var cids []int
		if idlist := c.Query("categoryID"); idlist != "" {
			cids = make([]int, 0)
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					cids = append(cids, id)
				}
			}
		}

		empowerID := make([]string, 0)
		if idlist, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(idlist) != "" {
			empowerID = strings.Split(idlist, ",")
		}

		targetTime := new(time.Time)
		*targetTime = time.Now()

		if c.Query("targetTime") != "" {
			if c.Query("targetTime") == "nil" {
				if actionAuth.(authority.ActionAuthSet).CheckAction(mission.ACTION_ADMIN_VIEW) {
					targetTime = nil
				}
			} else {
				ts, err := util.ParseDateTime(c.Query("targetTime"))
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				targetTime = &ts
			}
		}

		pageNo, _ := strconv.Atoi(c.Query("pageNo"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize"))

		if missionList, total, err := mission.GetMissions(siteID, actionAuth.(authority.ActionAuthSet), skipCheck, relateID, cids, c.Query("type"), c.Query("status"), c.Query("q"), targetTime, pageNo, pageSize, missionID, c.Query("authType"), c.Query("empower"), empowerID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "missionList": missionList, "total": total})
		}
	})

	authorized.POST("mission/mission/edit/:method", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param mission.Mission
		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "add":
			if err := param.Add(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "mission": param})
			}
		case "update":
			if err := param.Update(siteID); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0, "mission": param})
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

	authorized.POST("mission/mission/category/:method", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			MissionID  int `json:"missionID"`
			CategoryID int `json:"categoryID"`
		}

		err := c.ShouldBindJSON(&param)
		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		switch c.Param("method") {
		case "bind":
			err = mission.AddMissionCategory(siteID, param.MissionID, param.CategoryID)
		case "unbind":
			err = mission.DeleteMissionCategory(siteID, param.MissionID, param.CategoryID)
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

	sites.GET("mission/mission/empower", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_EDIT, mission.ACTION_EDIT, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW, mission.ACTION_ADMIN_COMPLETE, mission.ACTION_COMPLETE, mission.ACTION_ADMIN_REVIEW, mission.ACTION_ADMIN_REVIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		missionIDs := make([]string, 0)
		if str, exists := c.GetQuery("missionID"); exists && strings.TrimSpace(str) != "" {
			missionIDs = strings.Split(str, ",")
		}

		actionAuth, _ := c.Get("actionAuth")

		if missionEmpowers, err := authority.GetEmpowers(siteID, "mission", actionAuth.(authority.ActionAuthSet), mission.AdminActions, missionIDs...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "missionEmpowers": missionEmpowers})
		}
	})

	authorized.GET("mission/mission/empower/detail", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		missionIDs := make([]string, 0)
		if str, exists := c.GetQuery("missionID"); exists && strings.TrimSpace(str) != "" {
			missionIDs = strings.Split(str, ",")
		}

		var empowerIDs []string
		if str, exists := c.GetQuery("empowerID"); exists && strings.TrimSpace(str) != "" {
			empowerIDs = strings.Split(str, ",")
		}

		if missionEmpowerDetails, err := authority.GetEmpowerDetails(siteID, c.Query("empower"), empowerIDs, "mission", missionIDs, c.Query("groupBy")); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "missionEmpowerDetails": missionEmpowerDetails})
		}
	})

	authorized.POST("mission/mission/empower/:method", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_EDIT), func(c *gin.Context) {
		siteID := c.GetString("site")

		switch c.Param("method") {
		case "add":
			var param struct {
				MissionID int      `json:"missionID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}

			if err := mission.AddMissionEmpower(siteID, param.MissionID, param.Empower, param.EmpowerID, param.AuthList); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		case "delete":
			var param struct {
				MissionID int      `json:"missionID"`
				Empower   string   `json:"empower"`
				EmpowerID []string `json:"empowerID"`
				AuthList  []string `json:"authList"`
			}

			if err := c.ShouldBindJSON(&param); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			if err := mission.DeleteMissionEmpower(siteID, param.MissionID, param.Empower, param.EmpowerID, param.AuthList...); err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			} else {
				c.Set("json", map[string]interface{}{"retCode": 0})
			}
		default:
			c.AbortWithError(404, errors.New("invalid method"))
		}
	})

	authorized.POST("mission/mission/complete", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_COMPLETE, mission.ACTION_COMPLETE, mission.ACTION_ADMIN_REVIEW, mission.ACTION_REVIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		var param struct {
			MissionID int                    `json:"missionID"`
			Result    map[string]interface{} `json:"result"`
		}
		if err := c.ShouldBindJSON(&param); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
			return
		}

		actionAuth, _ := c.Get("actionAuth")

		completeTime := time.Now()
		if c.Query("completeTime") != "" && actionAuth.(authority.ActionAuthSet).CheckAction(mission.ACTION_ADMIN_COMPLETE, mission.ACTION_ADMIN_REVIEW) {
			ts, err := util.ParseDateTime(c.Query("completeTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			completeTime = ts
		}

		if complete, err := mission.CompleteMission(siteID, actionAuth.(authority.ActionAuthSet), param.MissionID, param.Result, completeTime); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "complete": complete})
		}
	})

	authorized.GET("mission/complete", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		UID := actionAuth.(authority.ActionAuthSet).GetUID()
		if uid := c.Query("UID"); uid != "" {
			var err error
			UID, err = strconv.Atoi(uid)
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
		}

		missionID := make([]int, 0)
		if idlist := c.Query("missionID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					missionID = append(missionID, id)
				}
			}
		}

		targetTime := new(time.Time)
		*targetTime = time.Now()
		if c.Query("targetTime") != "" {
			if c.Query("targetTime") == "nil" {
				targetTime = nil
			} else {
				ts, err := util.ParseDateTime(c.Query("targetTime"))
				if err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				}
				*targetTime = ts
			}
		}

		var status []string
		if str, exists := c.GetQuery("status"); exists && strings.TrimSpace(str) != "" {
			status = strings.Split(str, ",")
		}

		if completes, err := mission.GetCompletes(siteID, UID, status, targetTime, c.Query("type"), missionID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "completes": completes})
		}
	})

	sites.GET("mission/complete/count", checkAuth(mission.MODULE_MISSION, mission.ACTION_ADMIN_VIEW, mission.ACTION_VIEW), func(c *gin.Context) {
		siteID := c.GetString("site")

		actionAuth, _ := c.Get("actionAuth")

		UID := actionAuth.(authority.ActionAuthSet)[0].UID
		if uid := c.Query("UID"); uid == "-1" {
			UID = -1
		}

		missionID := make([]int, 0)
		if idlist := c.Query("missionID"); idlist != "" {
			parts := strings.Split(idlist, ",")
			for _, idstr := range parts {
				if id, err := strconv.Atoi(idstr); err != nil {
					c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
					return
				} else {
					missionID = append(missionID, id)
				}
			}
		}

		targetTime := time.Now()
		if c.Query("targetTime") != "" {
			ts, err := util.ParseDateTime(c.Query("targetTime"))
			if err != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
				return
			}
			targetTime = ts
		}

		includeBeforeSection, _ := strconv.ParseBool(c.Query("includeBeforeSection"))

		var status []string
		if str, exists := c.GetQuery("status"); exists && strings.TrimSpace(str) != "" {
			status = strings.Split(str, ",")
		}

		if completeCounts, err := mission.CountCompletes(siteID, UID, status, targetTime, includeBeforeSection, c.Query("type"), missionID...); err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0, "completeCounts": completeCounts})
		}
	})
}
