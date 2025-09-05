package push

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
)

type Subscription struct {
	ID             int                    `json:"ID"`
	SubscriberType string                 `json:"subscriberType"`
	SubscriberID   int                    `json:"subscriberID"`
	Type           string                 `json:"type"`
	Push           string                 `json:"push"`
	PushInterval   *util.Interval         `json:"pushInterval"`
	Ext            map[string]interface{} `json:"ext"`
}

const subscriptionColumns = "subscription.id,subscription.subscriber_type,subscription.subscriber_id,subscription.type,subscription.push,subscription.push_interval,subscription.ext"

func subscriptionTable(siteID string) string {
	return siteID + "_subscription"
}

func (s *Subscription) scan(rows *sql.Rows) error {
	var interval, ext string
	if err := rows.Scan(&s.ID, &s.SubscriberType, &s.SubscriberID, &s.Type, &s.Push, &interval, &ext); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(interval), &s.PushInterval); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(ext), &s.Ext); err != nil {
		return err
	}
	return nil
}

func (s *Subscription) Add(siteID string, txn *sql.Tx) error {
	p, err := getPush(s)
	if err != nil {
		return err
	}
	if err := Validate(siteID, p); err != nil {
		return err
	}

	if s.PushInterval == nil {
		s.PushInterval = new(util.Interval)
		s.PushInterval.Init()
	}
	if s.Ext == nil {
		s.Ext = make(map[string]interface{})
	}

	interval, _ := json.Marshal(s.PushInterval)
	ext, _ := json.Marshal(s.Ext)

	SQL := fmt.Sprintf(`
		INSERT INTO %s
			(subscriber_type,subscriber_id,type,push,push_interval,ext)
		VALUES
			(?,?,?,?,?,?)
	`, subscriptionTable(siteID))

	var ret sql.Result

	if txn != nil {
		ret, err = txn.Exec(SQL, s.SubscriberType, s.SubscriberID, s.Type, s.Push, string(interval), string(ext))
	} else {
		ret, err = datasource.GetConn().Exec(SQL, s.SubscriberType, s.SubscriberID, s.Type, s.Push, string(interval), string(ext))
	}

	if err != nil {
		log.Println("error insert subscription: ", err)
		return err
	} else if id, err := ret.LastInsertId(); err != nil {
		log.Println("error insert subscription: ", err)
		return err
	} else {
		s.ID = int(id)
	}

	return nil
}

func (s *Subscription) Update(siteID string) error {
	p, err := getPush(s)
	if err != nil {
		return err
	}
	if err := Validate(siteID, p); err != nil {
		return err
	}

	if s.PushInterval == nil {
		s.PushInterval = new(util.Interval)
		s.PushInterval.Init()
	}
	if s.Ext == nil {
		s.Ext = make(map[string]interface{})
	}

	interval, _ := json.Marshal(s.PushInterval)
	ext, _ := json.Marshal(s.Ext)

	SQL := fmt.Sprintf(`
		UPDATE
			%s
		SET
			subscriber_type=?, subscriber_id=?, type=?,push=?,push_interval=?,ext=?
		WHERE
			id= ?
	`, subscriptionTable(siteID))

	_, err = datasource.GetConn().Exec(SQL, s.SubscriberType, s.SubscriberID, s.Type, s.Push, string(interval), string(ext), s.ID)
	if err != nil {
		log.Println("error update subscription: ", err)
		return err
	}

	return nil
}

func (s *Subscription) Delete(siteID string) error {

	if _, err := datasource.GetConn().Exec(fmt.Sprintf(`
		DELETE FROM
			%s
		WHERE
			id=?
	`, subscriptionTable(siteID)), s.ID); err != nil {
		log.Println("error delete subscription: ", err)
		return err
	}

	return nil
}

func GetSubscriptionsWithTxn(siteID string, txn *sql.Tx, forUpdate bool, subscriptionID ...int) ([]*Subscription, error) {
	result := make([]*Subscription, 0)
	if len(subscriptionID) == 0 {
		return result, nil
	}
	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)
	if len(subscriptionID) == 1 {
		whereStmts = append(whereStmts, "subscription.id=?")
		values = append(values, subscriptionID[0])
	} else {
		placeholder := make([]string, 0)
		for _, id := range subscriptionID {
			placeholder = append(placeholder, "?")
			values = append(values, id)
		}
		whereStmts = append(whereStmts, fmt.Sprintf("subscrpition.id IN (%s)", strings.Join(placeholder, ",")))
	}
	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s subscription
	`, subscriptionColumns, subscriptionTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	if forUpdate {
		SQL += "\nFOR UPDATE"
	}

	rows, err := txn.Query(SQL, values...)
	if err != nil {
		log.Println("error get subscription: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s Subscription
		if err := s.scan(rows); err != nil {
			log.Println("error scan subscription: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}

func GetSubscriptionList(siteID string, subscriberType string, subscriberID int, subscriptionType, push string, extQuery map[string][]any) ([]*Subscription, error) {

	whereStmts := make([]string, 0)
	values := make([]interface{}, 0)

	if subscriberType == "" {
		return nil, errors.New("需要参数subscriberType")
	}
	whereStmts = append(whereStmts, "subscription.subscriber_type = ?")
	values = append(values, subscriberType)

	if subscriberID > 0 {
		whereStmts = append(whereStmts, "subscription.subscriber_id = ?")
		values = append(values, subscriberID)
	}

	if subscriptionType != "" {
		whereStmts = append(whereStmts, "subscription.type = ?")
		values = append(values, subscriptionType)
	}

	if len(extQuery) > 0 {
		for key, contents := range extQuery {
			if len(contents) == 1 {
				whereStmts = append(whereStmts, fmt.Sprintf("JSON_EXTRACT(subscription.ext, '$.%s') = ?", key))
				values = append(values, contents[0])
			} else {
				placeholder := make([]string, 0)
				for _, v := range contents {
					placeholder = append(placeholder, "?")
					values = append(values, v)
				}
				whereStmts = append(whereStmts, fmt.Sprintf("JSON_EXTRACT(subscription.ext, '$.%s') IN (%s)", key, strings.Join(placeholder, ",")))
			}
		}
	}

	if push != "" {
		whereStmts = append(whereStmts, "subscription.push = ?")
		values = append(values, push)
	}

	SQL := fmt.Sprintf(`
		SELECT
			%s
		FROM
			%s subscription
	`, subscriptionColumns, subscriptionTable(siteID))

	if len(whereStmts) > 0 {
		SQL += "WHERE " + strings.Join(whereStmts, " AND ")
	}

	rows, err := datasource.GetConn().Query(SQL, values...)
	if err != nil {
		log.Println("error get subscription: ", SQL, values, err)
		return nil, err
	}
	defer rows.Close()

	result := make([]*Subscription, 0)

	for rows.Next() {
		var s Subscription
		if err := s.scan(rows); err != nil {
			log.Println("error scan subscription: ", err)
			return nil, err
		}
		result = append(result, &s)
	}

	return result, nil
}
