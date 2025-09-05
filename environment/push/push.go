package push

import (
	"errors"
	"fmt"
	"log"
)

var E_invalid_subsriber = errors.New("推送不支持该订阅者类型")
var E_subscriber_should_not_push_but_valid = errors.New("该订阅不应直接推送")

var registry = make(map[string]IPusher)
var subscriptionRegistry = make(map[string]func(*Subscription) IPush)

func Register(pushType string, pusher IPusher) {
	if _, exists := registry[pushType]; exists {
		panic(fmt.Errorf("duplicate push type: %s", pushType))
	}
	registry[pushType] = pusher
}

func RegisterSubsciption(subType string, fac func(*Subscription) IPush) {
	if _, exists := registry[subType]; exists {
		panic(fmt.Errorf("duplicate subscription type: %s", subType))
	}
	subscriptionRegistry[subType] = fac
}

type IPusher interface {
	Validate(string, IPush) error
	Push(string, IPush) error
}

type IPush interface {
	ShouldPush(string) error
	GetSubscriptionType() string
	GetPushType() string
}

var e_invalid_push = errors.New("推送渠道无效")

func getPusher(ipush IPush) (IPusher, error) {
	pusher, exists := registry[ipush.GetPushType()]
	if !exists {
		log.Printf("error %s: %s", e_invalid_push.Error(), ipush.GetPushType())
		return nil, e_invalid_push
	}
	return pusher, nil
}

var e_subscription_not_support = errors.New("不支持的推送订阅")

func getPush(subs *Subscription) (IPush, error) {
	fac, exists := subscriptionRegistry[subs.Type]
	if !exists {
		log.Printf("error %s: %s", e_subscription_not_support.Error(), subs.Type)
		return nil, e_subscription_not_support
	}
	return fac(subs), nil
}

func Validate(siteID string, ipush IPush) error {
	pusher, err := getPusher(ipush)
	if err != nil {
		return err
	}
	err = pusher.Validate(siteID, ipush)
	if err != E_subscriber_should_not_push_but_valid {
		return err
	}

	return nil
}

func Push(siteID string, ipush IPush) error {
	pusher, err := getPusher(ipush)
	if err != nil {
		return err
	}
	if err := ipush.ShouldPush(siteID); err != nil {
		return err
	}
	return pusher.Push(siteID, ipush)
}
