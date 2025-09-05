package speaker

import (
	"errors"
	"fmt"
	"log"

	"obsessiontech/environment/authority"
	"obsessiontech/environment/event"
	"obsessiontech/environment/push"
)

const (
	PUSH_SPEAKER = "speaker"
)

func init() {
	push.Register(PUSH_SPEAKER, new(SpeakerPusher))
}

var e_invalid_speaker_push_config = errors.New("未实现扬声推送接口")

type ISpeakerPush interface {
	GetDeviceIDs(string) ([]int, error)
	GetResourceURI(string) (string, error)
	GetResourceURL(string) (string, error)
	GetRepeat(string) (int, error)
}

type SpeakerPusher struct{}

func (p *SpeakerPusher) Validate(siteID string, ipush push.IPush) error {

	i, ok := ipush.(ISpeakerPush)
	if !ok {
		log.Println("error validate not implement speakerPushInterface")
		return e_invalid_speaker_push_config
	}

	deviceIDs, err := i.GetDeviceIDs(siteID)
	if err != nil {
		return err
	} else if len(deviceIDs) == 0 {
		return errors.New("需要扬声设备")
	}
	uri, err := i.GetResourceURI(siteID)
	if err != nil {
		return err
	}
	if uri == "" {
		url, err := i.GetResourceURL(siteID)
		if err != nil {
			return err
		}
		if url == "" {
			return errors.New("需要指定音频资源")
		}
	}
	if repeat, err := i.GetRepeat(siteID); err != nil {
		return err
	} else if repeat <= 0 {
		return errors.New("需要指定播放次数")
	}

	return nil
}
func (p *SpeakerPusher) Push(siteID string, ipush push.IPush) error {
	i, ok := ipush.(ISpeakerPush)
	if !ok {
		log.Println("error push not implement speakerPushInterface")
		return e_invalid_speaker_push_config
	}

	deviceIDs, err := i.GetDeviceIDs(siteID)
	if err != nil {
		return err
	}

	for _, deviceID := range deviceIDs {

		speakerEvent := new(event.Event)
		speakerEvent.Type = EVENT_SPEAKER_SPEAK
		speakerEvent.MainRelateID = fmt.Sprintf("%d", deviceID)

		speakerEvent.SubRelateID = make(map[string]string)
		if uri, err := i.GetResourceURI(siteID); err != nil {
			return err
		} else if uri != "" {
			speakerEvent.SubRelateID["resourceURI"] = uri
		} else {
			if url, err := i.GetResourceURL(siteID); err != nil {
				return err
			} else if url != "" {
				speakerEvent.SubRelateID["resourceURL"] = url
			} else {
				return errors.New("需要指定音频资源")
			}
		}
		repeat, err := i.GetRepeat(siteID)
		if err != nil {
			return err
		}
		if repeat <= 0 {
			return errors.New("需要指定播放次数")
		}
		speakerEvent.Ext = make(map[string]interface{})
		speakerEvent.Ext["repeat"] = repeat

		if err := speakerEvent.Add(siteID, authority.ActionAuthSet{{Action: event.ACTION_ADMIN_EDIT}}); err != nil {
			return err
		}
	}

	return nil
}
