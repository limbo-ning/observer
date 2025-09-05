package subscription

import (
	"fmt"
	"log"

	"obsessiontech/common/util"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/push"
	"obsessiontech/environment/role"
	"obsessiontech/environment/user"
)

func checkAuth(siteID string, entityID, uid int) bool {

	actionAuth, err := role.GetAuthorityActions(siteID, entity.MODULE_ENTITY, "", "", uid, entity.ACTION_ADMIN_VIEW, entity.ACTION_ENTITY_VIEW)
	if err != nil {
		log.Println("error get auth actions to push: ", err)
		return false
	}

	if err := entity.CheckAuth(siteID, actionAuth, entityID, entity.ACTION_ENTITY_VIEW); err != nil {
		return false
	}

	return true
}

func GetSubscriptionsToPush(siteID string, entityID, stationID int, subscriptionType string) ([]*push.Subscription, error) {
	result := make([]*push.Subscription, 0)

	upush, err := getUserAndRoleSubscriptionsToPush(siteID, entityID, subscriptionType)
	if err != nil {
		log.Println("error get user subscriptions: ", err)
	} else {
		result = append(result, upush...)
	}

	epush, err := getEntitySubscriptionsToPush(siteID, entityID, subscriptionType)
	if err != nil {
		log.Println("error get entity subscriptions: ", err)
	} else {
		result = append(result, epush...)
	}

	spush, err := getStationSubscriptionsToPush(siteID, stationID, subscriptionType)
	if err != nil {
		log.Println("error get station subscriptions: ", err)
	} else {
		result = append(result, spush...)
	}

	log.Println("total subscription to push: ", len(result))

	return result, nil
}

func getUserAndRoleSubscriptionsToPush(siteID string, entityID int, subscriptionType string) ([]*push.Subscription, error) {

	uidPushUnique := make(map[string]*push.Subscription)

	roleSubscriptionList, err := push.GetSubscriptionList(siteID, "role", -1, subscriptionType, "", map[string][]any{"entityID": {-1, entityID}})
	if err != nil {
		return nil, err
	}

	roleSubList := make(map[int][]*push.Subscription)
	for _, s := range roleSubscriptionList {
		if _, exists := roleSubList[s.SubscriberID]; !exists {
			roleSubList[s.SubscriberID] = make([]*push.Subscription, 0)
		}
		roleSubList[s.SubscriberID] = append(roleSubList[s.SubscriberID], s)
	}

	for roleID, subList := range roleSubList {
		userList, _, err := user.GetRoleUsers(siteID, "", "", "", user.USER_ACTIVE, 0, -1, "", roleID)
		if err != nil {
			return nil, err
		}

		for _, u := range userList {
			for _, s := range subList {

				ns := new(push.Subscription)
				if err := util.Clone(s, ns); err != nil {
					return nil, err
				}

				ns.SubscriberType = "user"
				ns.SubscriberID = u.UserID
				uidPushUnique[fmt.Sprintf("%d#%s", u.UserID, s.Push)] = ns
			}
		}
	}

	subscriptionList, err := push.GetSubscriptionList(siteID, "user", -1, subscriptionType, "", map[string][]any{"entityID": {-1, entityID}})
	if err != nil {
		return nil, err
	}

	for _, s := range subscriptionList {
		uidPushUnique[fmt.Sprintf("%d#%s", s.SubscriberID, s.Push)] = s
	}

	result := make([]*push.Subscription, 0)
	for _, s := range uidPushUnique {
		if checkAuth(siteID, entityID, s.SubscriberID) {
			result = append(result, s)
		}
	}

	return result, nil
}

func getEntitySubscriptionsToPush(siteID string, entityID int, subscriptionType string) ([]*push.Subscription, error) {
	return push.GetSubscriptionList(siteID, "entity", entityID, subscriptionType, "", nil)
}

func getStationSubscriptionsToPush(siteID string, stationID int, subscriptionType string) ([]*push.Subscription, error) {
	return push.GetSubscriptionList(siteID, "station", stationID, subscriptionType, "", nil)
}
