package wechat

const (
	MCH_AGENT             = "AGENT"
	MCH_OWN               = "OWN"
	MCH_STATUS_PROCESSING = "PROCESSING"
	MCH_STATUS_REJECTED   = "REJECTED"
	MCH_STATUS_ACTIVE     = "ACTIVE"
)

type WechatMerchant struct {
	MerchantID              string `json:"merchantID"`
	Type                    string `json:"type"`
	UniformSocialCreditCode string `json:"uniformSocialCreditCode"`
	ContactMobile           string `json:"contactMobile"`
	Status                  string `json:"status"`
}
