package speaker

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"

	"obsessiontech/common/util"
	"obsessiontech/environment/event"
)

const EVENT_SPEAKER_SPEAK = "speaker_speak"

func init() {
	event.Register(EVENT_SPEAKER_SPEAK, func() event.IEvent {
		return new(SpeakEvent)
	})
}

type SpeakEvent struct{}

func (e *SpeakEvent) ValidateEvent(siteID string, eventInstance *event.Event) error {

	deviceID, err := strconv.Atoi(eventInstance.MainRelateID)
	if err != nil {
		return err
	}

	if deviceID <= 0 {
		return errors.New("需要设备ID")
	}

	resourceURI, exists := eventInstance.SubRelateID["resourceURI"]
	if !exists || resourceURI == "" {
		resourceURL, exists := eventInstance.SubRelateID["resourceURL"]
		if !exists || resourceURL == "" {
			return errors.New("需要扬声音频资源")
		}
	}

	return nil
}

func (e *SpeakEvent) ExecuteEvent(siteID string, txn *sql.Tx, eventInstance *event.Event) error {

	deviceID, _ := strconv.Atoi(eventInstance.MainRelateID)

	repeat := 0
	if repeatSet, exists := eventInstance.Ext["repeat"]; exists {
		if repeatSetFlt, isFloat := repeatSet.(float64); isFloat {
			repeat = int(repeatSetFlt)
		} else if repeatSetInt, isInt := repeatSet.(int); isInt {
			repeat = int(repeatSetInt)
		} else if repeatSetStr, isStr := repeatSet.(string); isStr {
			repeat, _ = strconv.Atoi(repeatSetStr)
		}
	}

	if repeat == 0 {
		repeat = 1
	}

	if err := BroadcastSound(siteID, eventInstance.ID, deviceID, eventInstance.SubRelateID["resourceURL"], eventInstance.SubRelateID["resourceURI"], repeat); err != nil {
		return err
	}

	eventInstance.Feedback(event.IN_PROGRESS, map[string]interface{}{
		"at": util.Time(time.Now()),
	})

	if err := eventInstance.UpdateStatusWithTxn(siteID, txn); err != nil {
		log.Println("error update event after speak broadcast: ", err)
	}

	return nil
}
