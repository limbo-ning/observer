package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"obsessiontech/common/datasource"
	"obsessiontech/common/encrypt"
	"obsessiontech/common/util"
	"obsessiontech/environment/authority"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/data/operation"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/ipcclient"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/push"
	"obsessiontech/environment/role"
	"obsessiontech/environment/user"
	"obsessiontech/environment/user/auth/engine"

	"github.com/gin-gonic/gin"
)

func loadMigrate() {

	internal.POST("migrate/:target", func(c *gin.Context) {
		siteID := c.Query("siteID")
		var err error
		switch c.Param("target") {
		case "user":
			err = migrateUserPassword(siteID)
		case "monitorCode":
			err = migrateMonitorCode(siteID)
		case "monitorLimit":
			err = migrateEnvironmentMonitorLimit(siteID)
		// case "entityCategoryEmpower":
		// 	keyID, _ := strconv.Atoi(c.Query("keyID"))
		// 	authTypes := c.Query("authType")
		// 	if authTypes == "" {
		// 		c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "need auth type"})
		// 		return
		// 	}
		// 	authType := strings.Split(authTypes, ",")
		// 	err = migrateEnvironmentCategoryEmpower(siteID, keyID, c.Query("roleSeries"), authType)
		case "entityEmpower":
			authTypes := c.Query("authType")
			if authTypes == "" {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": "need auth type"})
				return
			}
			authType := strings.Split(authTypes, ",")
			err = migrateEnvironmentEntityEmpower(siteID, c.Query("roleSeries"), authType)
		case "subscription":
			err = migrateSubscription(siteID, c.Query("subscriberType"))
		case "testSubscription":
			err = migrateTestSubscription(siteID, c.Query("pushType"), c.Query("dataFlag"))
		case "testUpload":
			stationID, e := strconv.Atoi(c.Query("stationID"))
			if e != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": e.Error()})
				return
			}
			var monitorID []int
			if str, exists := c.GetQuery("monitorID"); exists {
				for _, idstr := range strings.Split(str, ",") {
					id, e := strconv.Atoi(idstr)
					if e != nil {
						c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": e.Error()})
						return
					}
					monitorID = append(monitorID, id)
				}
			}
			value, e := strconv.ParseFloat(c.Query("value"), 64)
			if e != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": e.Error()})
				return
			}
			dataTime, e := util.ParseDateTime(c.Query("dataTime"))
			if e != nil {
				c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": e.Error()})
				return
			}
			err = migrateTestUpload(siteID, c.Query("dataType"), stationID, monitorID, value, dataTime, strings.Split(c.Query("code"), ","))
		default:
			c.AbortWithStatus(404)
			return
		}

		if err != nil {
			c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
		} else {
			c.Set("json", map[string]interface{}{"retCode": 0})
		}
	})

	// projectc.POST("migrate/:target", checkAuth(site.MODULE_SITE, site.ACTION_ADMIN_EDIT_SITE), func(c *gin.Context) {
	// 	siteID := c.GetString("site")
	// 	var err error
	// 	switch c.Param("target") {
	// 	case "goods":
	// 		err = migrateGoods(siteID)
	// 	case "template":
	// 		err = migrateTemplate(siteID)
	// 	case "order":
	// 		err = migrateOrder(siteID)
	// 	case "purchase":
	// 		err = migratePurchase(siteID)
	// 	case "quantity":
	// 		err = migrateQuantity(siteID)
	// 	case "user":
	// 		err = migrateUserPassword(siteID)
	// 	default:
	// 		c.AbortWithStatus(404)
	// 		return
	// 	}

	// 	if err != nil {
	// 		c.Set("json", map[string]interface{}{"retCode": 500, "retMsg": err.Error()})
	// 	} else {
	// 		c.Set("json", map[string]interface{}{"retCode": 0})
	// 	}
	// })
}

func migrateEnvironmentMonitorLimit(siteID string) error {
	monitorLimitList, err := monitor.GetMonitorLimits(siteID, nil)
	if err != nil {
		return err
	}

	for _, l := range monitorLimitList {
		if l.Overproof != "" {
			flag, err := monitor.GetFlagByBit(siteID, monitor.FLAG_OVERPROOF)
			if err != nil {
				return err
			}

			if flag == nil {
				return errors.New("no flag overproof")
			}

			fl := new(monitor.FlagLimit)
			fl.StationID = l.StationID
			fl.MonitorID = l.MonitorID
			fl.Flag = flag.Flag

			limits := make([]string, 0)

			segments := strings.Split(l.Overproof, ";")
			for _, segment := range segments {
				if segment == "" {
					continue
				}

				var limit string

				parts := strings.Split(segment, ",")
				if len(parts) != 2 {
					return fmt.Errorf("错误的超标区间:%s", segment)
				}

				if parts[0] == "-"+monitor.NOTATION_INFINITY {
					limit += "<"
				} else {
					limit += ">" + parts[0]
				}

				if parts[1] == "+"+monitor.NOTATION_INFINITY {
					//do nothing
				} else {
					if limit == "<" {
						limit += parts[1]
					} else if limit != "" {
						limit += ","
						limit += "<" + parts[1]
					}
				}

				limits = append(limits, limit)
			}

			fl.Region = strings.Join(limits, ";")

			log.Println("migrate: ", flag.Name, l.Overproof, fl.Region)

			if err := fl.Add(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_EDIT}}); err != nil {
				log.Println("error migrade: ", fl.StationID, fl.MonitorID, fl.Flag, err)
			}
		}
		if l.InvarianceHour > 0 {
			flag, err := monitor.GetFlagByBit(siteID, monitor.FLAG_DATA_INVARIANCE)
			if err != nil {
				return err
			}

			if flag == nil {
				return errors.New("no flag di")
			}

			fl := new(monitor.FlagLimit)
			fl.StationID = l.StationID
			fl.MonitorID = l.MonitorID
			fl.Flag = flag.Flag
			fl.Region = fmt.Sprintf(">=%d", l.InvarianceHour)

			log.Println("migrate: ", flag.Name, l.InvarianceHour, fl.Region)

			if err := fl.Add(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_EDIT}}); err != nil {
				log.Println("error migrade: ", fl.StationID, fl.MonitorID, fl.Flag, err)
			}
		}
		if l.TopEffective > 0 {

			flag, err := monitor.GetFlagByBit(siteID, monitor.FLAG_TOP_LIMIT)
			if err != nil {
				return err
			}

			if flag == nil {
				return errors.New("no flag top effective")
			}

			fl := new(monitor.FlagLimit)
			fl.StationID = l.StationID
			fl.MonitorID = l.MonitorID
			fl.Flag = flag.Flag
			fl.Region = fmt.Sprintf(">%G", l.TopEffective)

			log.Println("migrate: ", flag.Name, l.TopEffective, fl.Region)

			if err := fl.Add(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_EDIT}}); err != nil {
				log.Println("error migrade: ", fl.StationID, fl.MonitorID, fl.Flag, err)
			}
		}
		if l.LowerDetection > 0 {
			flag, err := monitor.GetFlagByBit(siteID, monitor.FLAG_LOW_LIMIT)
			if err != nil {
				return err
			}

			if flag == nil {
				return errors.New("no flag low limit")
			}

			fl := new(monitor.FlagLimit)
			fl.StationID = l.StationID
			fl.MonitorID = l.MonitorID
			fl.Flag = flag.Flag
			fl.Region = fmt.Sprintf("<%G", l.LowerDetection)

			log.Println("migrate: ", flag.Name, l.LowerDetection, fl.Region)

			if err := fl.Add(siteID, authority.ActionAuthSet{{Action: entity.ACTION_ADMIN_EDIT}}); err != nil {
				log.Println("error migrade: ", fl.StationID, fl.MonitorID, fl.Flag, err)
			}
		}
	}

	return nil
}

// func migrateEnvironmentCategoryEmpower(siteID string, keyID int, roleSeries string, authType []string) error {

// 	keys, categories, err := category.GetCategories(siteID, "", "entity", "", authority.ActionAuthSet{{Action: "admin_view"}}, keyID)
// 	if err != nil {
// 		return err
// 	}

// 	roleAuthorities, err := role.GetRoleAuthority(siteID, []string{entity.MODULE_ENTITY})
// 	if err != nil {
// 		return err
// 	}

// 	categoryViewRoles := make(map[int]int)
// 	for roleID, ras := range roleAuthorities {

// 		if roleID > 0 {
// 			for _, ra := range ras {
// 				if ra.Action == "category_view" {
// 					cid, err := strconv.Atoi(ra.RoleType)
// 					if err != nil {
// 						return fmt.Errorf("cant parse category id: %s %s", ra.RoleType, err.Error())
// 					}
// 					categoryViewRoles[cid] = roleID
// 				}
// 			}
// 		}
// 	}

// 	for _, key := range keys {
// 		catIDs := make([]int, 0)
// 		for _, cat := range categories[key.ID] {
// 			catIDs = append(catIDs, cat.ID)
// 		}

// 		if len(catIDs) == 0 {
// 			continue
// 		}

// 		categoryObjectIDs, err := category.GetCategoryObjectIDs[int](siteID, "entity", catIDs...)
// 		if err != nil {
// 			return err
// 		}

// 		for _, cat := range categories[key.ID] {
// 			roleID := categoryViewRoles[cat.ID]
// 			roles, err := role.GetRoles(siteID, "", roleID)
// 			if err != nil {
// 				return err
// 			}
// 			if len(roles) == 0 {
// 				continue
// 			}
// 			r := roles[0]
// 			r.Series = roleSeries
// 			if err := r.Update(siteID); err != nil {
// 				return err
// 			}
// 			if err := r.SyncAuth(siteID); err != nil {
// 				return err
// 			}

// 			objectIDs := categoryObjectIDs[cat.ID]
// 			if len(objectIDs) == 0 {
// 				continue
// 			}

// 			for _, entityID := range objectIDs {
// 				entities, err := entity.GetEntities(siteID, entityID)
// 				if err != nil {
// 					return err
// 				}

// 				if len(entities) == 0 {
// 					continue
// 				}

// 				if err := entity.AddEntityEmpower(siteID, authority.ActionAuthSet{{Action: "admin_edit"}}, entityID, "role", []string{fmt.Sprintf("%d", r.ID)}, authType); err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	return nil
// }

func migrateEnvironmentEntityEmpower(siteID string, roleSeries string, authType []string) error {
	roleAuthorities, err := role.GetRoleAuthority(siteID, []string{entity.MODULE_ENTITY})
	if err != nil {
		return err
	}

	entityViewRoles := make(map[int][]int)
	for roleID, ras := range roleAuthorities {
		if roleID > 0 {
			for _, ra := range ras {
				if ra.Action == "view" && ra.RoleType != "" {
					eid, err := strconv.Atoi(ra.RoleType)
					if err != nil {
						return fmt.Errorf("cant parse entity id: ra[%d] roleid[%d] roletype[%s] err[%s]", ra.ID, ra.RoleID, ra.RoleType, err.Error())
					}
					if _, exists := entityViewRoles[eid]; !exists {
						entityViewRoles[eid] = make([]int, 0)
					}
					entityViewRoles[eid] = append(entityViewRoles[eid], roleID)
				}
			}
		}
	}

	for entityID, roleIDs := range entityViewRoles {

		log.Println("migrate entity empower: ", entityID, roleIDs)

		entities, err := entity.GetEntities(siteID, entityID)
		if err != nil {
			return err
		}

		if len(entities) == 0 {
			log.Println("no entity: ", entityID)
			continue
		}

		roleList, err := role.GetRoles(siteID, "", roleIDs...)
		if err != nil {
			return err
		}

		roles := make(map[int]*role.Role)

		for _, r := range roleList {
			roles[r.ID] = r
		}

		for _, rid := range roleIDs {
			r := roles[rid]

			if r == nil {
				log.Println("no role: ", rid)
				continue
			}

			r.Series = roleSeries
			if err := r.Update(siteID); err != nil {
				return err
			}
			if err := r.SyncAuth(siteID); err != nil {
				return err
			}

			if err := entity.AddEntityEmpower(siteID, authority.ActionAuthSet{{Action: "admin_edit"}}, entityID, "role", []string{fmt.Sprintf("%d", r.ID)}, authType); err != nil {
				return err
			}
		}
	}

	return nil
}

func migrateQuantity(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, blocks
		FROM
			%s_goods
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	goodsBlocks := make(map[int]string)
	for rows.Next() {
		var id int
		var blocks string

		if err := rows.Scan(&id, &blocks); err != nil {
			return err
		}

		goodsBlocks[id] = blocks
	}

	for id, b := range goodsBlocks {
		updatedBlock, err := updateBlockQuantity(b)
		if err != nil {
			return err
		}
		if updatedBlock != "" {
			if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
				UPDATE
					%s_goods
				SET
					blocks = ?
				WHERE
					id = ?
			`, siteID), updatedBlock, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateBlockQuantity(input string) (string, error) {
	var blocks []map[string]interface{}

	if err := json.Unmarshal([]byte(input), &blocks); err != nil {
		return "", err
	}

	for i, block := range blocks {
		for _, chain := range block["chains"].([]interface{}) {
			updateChainQuantity(i, chain.(map[string]interface{}))
		}
	}

	result, err := json.Marshal(blocks)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func updateChainQuantity(blockIndex int, chain map[string]interface{}) {
	if chain["type"] == "quantity" {
		chain["isGoodsQuantity"] = blockIndex == 0
	} else if chain["type"] == "price" {
		for _, sub := range chain["prerequisite"].([]interface{}) {
			updateChainQuantity(blockIndex, sub.(map[string]interface{}))
		}
	} else if chain["type"] == "choice" {
		for _, block := range chain["blocks"].([]interface{}) {
			for _, sub := range block.(map[string]interface{})["chains"].([]interface{}) {
				updateChainQuantity(blockIndex, sub.(map[string]interface{}))
			}
		}
	}
}

func migrateGoods(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, blocks
		FROM
			%s_goods
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	goodsBlocks := make(map[int]string)
	for rows.Next() {
		var id int
		var blocks string

		if err := rows.Scan(&id, &blocks); err != nil {
			return err
		}

		goodsBlocks[id] = blocks
	}

	for id, b := range goodsBlocks {
		updatedBlock, err := updateBlock(b)
		if err != nil {
			return err
		}
		if updatedBlock != "" {
			if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
				UPDATE
					%s_goods
				SET
					blocks = ?
				WHERE
					id = ?
			`, siteID), updatedBlock, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func migrateTemplate(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, blocks
		FROM
			%s_template
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	goodsBlocks := make(map[int]string)
	for rows.Next() {
		var id int
		var blocks string

		if err := rows.Scan(&id, &blocks); err != nil {
			return err
		}

		goodsBlocks[id] = blocks
	}

	for id, b := range goodsBlocks {
		updatedBlock, err := updateBlock(b)
		if err != nil {
			return err
		}
		if updatedBlock != "" {
			if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
				UPDATE
					%s_template
				SET
					blocks = ?
				WHERE
					id = ?
			`, siteID), updatedBlock, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateBlock(input string) (string, error) {
	var blocks []map[string]interface{}

	if err := json.Unmarshal([]byte(input), &blocks); err != nil {
		return "", err
	}

	for _, block := range blocks {
		if block["ID"].(string) == "retail_shop_order" {
			block["ID"] = "retail_deliver"
		}
		chains := block["chains"]
		if chains != nil {
			for _, chain := range chains.([]interface{}) {
				if chain.(map[string]interface{})["type"] == "retailShopOrder" || chain.(map[string]interface{})["type"] == "retailDeliver" {
					chain.(map[string]interface{})["type"] = "retailDeliver"
					options := make([]string, 0)
					for _, option := range chain.(map[string]interface{})["options"].([]interface{}) {
						if option.(string) == "PICK_UP" {
							options = append(options, "PICKUP")
						} else if option.(string) == "DELIVER" {
							options = append(options, "DELIVERY")
						} else {
							options = append(options, option.(string))
						}
					}
					optionParam := make(map[string]interface{})
					for option, param := range chain.(map[string]interface{})["optionParam"].(map[string]interface{}) {
						if option == "PICK_UP" {
							optionParam["PICKUP"] = param
						} else if option == "DELIVER" {
							optionParam["DELIVERY"] = param
						} else {
							optionParam[option] = param
						}
					}
					chain.(map[string]interface{})["options"] = options
					chain.(map[string]interface{})["optionParam"] = optionParam
				}
			}
		}
	}

	result, err := json.Marshal(blocks)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func migrateOrder(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, receipt
		FROM
			%s_order
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	orderRecipts := make(map[int]string)
	for rows.Next() {
		var id int
		var receipt string

		if err := rows.Scan(&id, &receipt); err != nil {
			return err
		}

		orderRecipts[id] = receipt
	}

	for id, b := range orderRecipts {
		// updated, err := updateChainReceipt(b)
		updated, err := updateChainReceiptSource(b)
		if err != nil {
			return err
		}
		if updated != "" {
			if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
				UPDATE
					%s_order
				SET
					receipt = ?
				WHERE
					id = ?
			`, siteID), updated, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateChainReceiptSource(input string) (string, error) {
	var receipts []map[string]interface{}

	if err := json.Unmarshal([]byte(input), &receipts); err != nil {
		return "", err
	}

	for _, receipt := range receipts {
		receipt["source"] = receipt["sourceName"]
		delete(receipt, "sourceName")
	}

	result, err := json.Marshal(receipts)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func updateChainReceipt(input string) (string, error) {
	var receipts []map[string]interface{}

	if err := json.Unmarshal([]byte(input), &receipts); err != nil {
		return "", err
	}

	for _, receipt := range receipts {
		if receipt["source"].(string) == "food" {
			receipt["source"] = "item"
		}
		for _, r := range receipt["receipts"].([]interface{}) {
			if r.(map[string]interface{})["type"] == "retailShopOrder" || r.(map[string]interface{})["type"] == "retailDeliver" {
				r.(map[string]interface{})["type"] = "retailDeliver"
				optionParam := make(map[string]interface{})
				for option, param := range r.(map[string]interface{})["option"].(map[string]interface{}) {
					if option == "PICK_UP" {
						optionParam["PICKUP"] = param
					} else if option == "DELIVER" {
						optionParam["DELIVERY"] = param
					} else {
						optionParam[option] = param
					}
				}
				r.(map[string]interface{})["option"] = optionParam
				delete(r.(map[string]interface{}), "shopID")
			}
		}
	}

	result, err := json.Marshal(receipts)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func migratePurchase(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, receipt
		FROM
			%s_purchase
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	orderRecipts := make(map[int]string)
	for rows.Next() {
		var id int
		var receipt string

		if err := rows.Scan(&id, &receipt); err != nil {
			return err
		}

		orderRecipts[id] = receipt
	}

	for id, b := range orderRecipts {
		updated, err := updateReceipt(b)
		if err != nil {
			return err
		}
		if updated != "" {
			if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
				UPDATE
					%s_purchase
				SET
					receipt = ?
				WHERE
					id = ?
			`, siteID), updated, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateReceipt(input string) (string, error) {
	var receipts []interface{}

	if err := json.Unmarshal([]byte(input), &receipts); err != nil {
		return "", err
	}

	for _, r := range receipts {
		if r.(map[string]interface{})["type"] == "retailShopOrder" || r.(map[string]interface{})["type"] == "retailDeliver" {
			r.(map[string]interface{})["type"] = "retailDeliver"
			optionParam := make(map[string]interface{})
			for option, param := range r.(map[string]interface{})["option"].(map[string]interface{}) {
				if option == "PICK_UP" {
					optionParam["PICKUP"] = param
				} else if option == "DELIVER" {
					optionParam["DELIVERY"] = param
				} else {
					optionParam[option] = param
				}
			}
			r.(map[string]interface{})["option"] = optionParam
			delete(r.(map[string]interface{}), "shopID")
		}
	}

	result, err := json.Marshal(receipts)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func migrateUserPassword(siteID string) error {
	users, _, err := user.GetUsers(siteID, "", "", "", 0, -1, "")
	if err != nil {
		return err
	}

	return datasource.Txn(func(txn *sql.Tx) {
		for _, u := range users {
			if u.IsPasswordSet {
				u.Password, err = engine.EncryptPassword(encrypt.Base64Encrypt(u.Password))
				if err := u.Update(siteID, txn); err != nil {
					panic(err)
				}
			}
		}
	})

}

func migrateMonitorCode(siteID string) error {
	rows, err := datasource.GetConn().Query(fmt.Sprintf(`
		SELECT
			id, exp, minutely_generation, hourly_generation, daily_generation
		FROM
			%s_monitorcode
	`, siteID))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, exp, minutelyG, hourlyG, dailyG int

		if err := rows.Scan(&id, &exp, &minutelyG, &hourlyG, &dailyG); err != nil {
			return err
		}

		processorList := make([]interface{}, 0)

		if exp != 0 {
			var offset float64
			if exp > 0 {
				offset, _ = strconv.ParseFloat("1"+strings.Repeat("0", exp), 64)

			} else {
				offset, _ = strconv.ParseFloat("0."+strings.Repeat("0", -1*exp-1)+"1", 64)
			}
			processorList = append(processorList, map[string]interface{}{
				"rule":   "offset",
				"method": "multiply",
				"offset": offset,
				"fields": []string{data.RTD},
			})
		}

		processorList = append(processorList, map[string]interface{}{
			"rule": "flag",
		})

		if minutelyG > 0 {
			processorList = append(processorList, map[string]interface{}{
				"rule":     "generate",
				"dataType": data.MINUTELY,
				"interval": minutelyG,
			})
		}

		if hourlyG > 0 {
			processorList = append(processorList, map[string]interface{}{
				"rule":     "generate",
				"dataType": data.HOURLY,
				"interval": hourlyG,
			})
		}

		if dailyG > 0 {
			processorList = append(processorList, map[string]interface{}{
				"rule":     "generate",
				"dataType": data.DAILY,
				"interval": dailyG,
			})
		}

		processors, _ := json.Marshal(processorList)

		if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
			UPDATE
				%s_monitorcode
			SET
				processors = ?
			WHERE
				id = ?
		`, siteID), string(processors), id); err != nil {
			return err
		}
	}

	return nil
}

func migrateSubscription(siteID, subscriberType string) error {

	subscriptionList, err := push.GetSubscriptionList(siteID, subscriberType, -1, "", "", map[string][]any{"entityID": {-1, 100}})

	if err != nil {
		return err
	}

	log.Println("test migrate get subsription: ", len(subscriptionList))
	for _, sub := range subscriptionList {
		print, _ := json.Marshal(sub)
		log.Println(string(print))
	}

	// subscriptionList, err := push.GetSubscriptionList(siteID, subscriberType, -1, "", "")
	// if err != nil {
	// 	return err
	// }

	// for _, sub := range subscriptionList {

	// 	if sub.SubscriberType == "user" {
	// 		entityID, err := strconv.Atoi(sub.TypeExt)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		sub.Ext["entityID"] = entityID
	// 	}

	// 	switch sub.Push {
	// 	case push.PUSH_ALI_SMS:
	// 		if sub.SubscriberType != "user" && sub.PushExt != "" {
	// 			sub.Ext["mobile"] = sub.PushExt
	// 		}
	// 	case push.PUSH_WXMINIAPP_SUBSCRIPTION:
	// 	case push.PUSH_WXOPEN_TEMPLATE:
	// 		if sub.PushExt != "" {
	// 			sub.Ext["wxUserInfo"] = sub.PushExt
	// 		}
	// 	case speaker.PUSH_SPEAKER:
	// 		deviceID, err := strconv.Atoi(sub.TypeExt)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		sub.Ext["deviceID"] = deviceID
	// 		parts := strings.Split(sub.PushExt, "::")
	// 		if len(parts) != 2 {
	// 			return errors.New("音频格式错误")
	// 		}
	// 		sub.Ext["resourceURI"] = parts[0]
	// 		sub.Ext["repeat"], _ = strconv.Atoi(parts[1])
	// 	case mission.PUSH_MISSION:

	// 	}

	// 	if err := sub.Update(siteID); err != nil {
	// 		return err
	// 	}
	// }

	// log.Println("migrate subscription: ", len(subscriptionList))

	return nil
}

func migrateTestSubscription(siteID string, pushType string, dataFlag string) error {
	switch pushType {
	case "station_status":
		return ipcclient.PushOffline(siteID, 261)
	case "hourly":
		hourly := new(data.HourlyData)
		hourly.StationID = 261
		hourly.MonitorID = 84
		hourly.Avg = 100
		hourly.Flag = dataFlag
		hourly.DataTime = util.Time(time.Now())
		return ipcclient.PushData(siteID, hourly)
	case "minutely":
		minutely := new(data.MinutelyData)
		minutely.StationID = 261
		minutely.MonitorID = 84
		minutely.Avg = 100
		minutely.Flag = dataFlag
		minutely.DataTime = util.Time(time.Now())
		return ipcclient.PushData(siteID, minutely)
	}

	return nil
}

func migrateTestUpload(siteID string, dataType string, stationID int, monitorID []int, value float64, dataTime time.Time, code []string) error {

	list := make([]data.IData, 0)

	for i, mid := range monitorID {
		var d data.IData

		switch dataType {
		case data.REAL_TIME:
			d = new(data.RealTimeData)
		case data.MINUTELY:
			d = new(data.MinutelyData)
		case data.HOURLY:
			d = new(data.HourlyData)
		case data.DAILY:
			d = new(data.DailyData)
		}

		if rtd, ok := d.(data.IRealTime); ok {
			rtd.SetRtd(value)
		} else if interval, ok := d.(data.IInterval); ok {
			interval.SetAvg(value)
			interval.SetCou(value)
			interval.SetMax(value)
			interval.SetMin(value)
		}

		d.SetStationID(stationID)
		d.SetMonitorID(mid)
		d.SetDataTime(util.Time(dataTime))
		d.SetCode(code[i])

		list = append(list, d)
	}

	uper := new(dataprocess.Uploader)
	up := new(operation.Upload)

	monitor.LoadMonitor(siteID)
	monitor.LoadMonitorCode(siteID)
	monitor.LoadFlagLimit(siteID)

	if err := uper.UploadBatchData(siteID, up, list...); err != nil {
		return err
	}

	if err := uper.UploadUnuploaded(siteID, up); err != nil {
		return err
	}

	return nil
}
