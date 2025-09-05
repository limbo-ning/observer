package message

import (
	"obsessiontech/wechat/util"
)

func PushMiniAppContactMessage(msg IContactMessage) error {
	return SendContactMessage(msg, util.GetMiniAppAccessToken())

}
