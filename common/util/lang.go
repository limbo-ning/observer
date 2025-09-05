package util

import (
	"bytes"
	"encoding/json"
	"errors"
)

var e_lang_unsupport = errors.New("unsupported language")

const default_lang = "default"

const (
	en    = "en"
	ar    = "ar"
	de    = "de"
	el    = "el"
	es    = "es"
	it    = "it"
	ja    = "ja"
	fr    = "fr"
	ko    = "ko"
	ru    = "ru"
	th    = "th"
	zh_cn = "zh_cn"
	zh_hk = "zh_hk"
	zh_tw = "zh_tw"

// en 英文
// en_US 英文 (美国)
// ar 阿拉伯文
// ar_AE 阿拉伯文 (阿拉伯联合酋长国)
// ar_BH 阿拉伯文 (巴林)
// ar_DZ 阿拉伯文 (阿尔及利亚)
// ar_EG 阿拉伯文 (埃及)
// ar_IQ 阿拉伯文 (伊拉克)
// ar_JO 阿拉伯文 (约旦)
// ar_KW 阿拉伯文 (科威特)
// ar_LB 阿拉伯文 (黎巴嫩)
// ar_LY 阿拉伯文 (利比亚)
// ar_MA 阿拉伯文 (摩洛哥)
// ar_OM 阿拉伯文 (阿曼)
// ar_QA 阿拉伯文 (卡塔尔)
// ar_SA 阿拉伯文 (沙特阿拉伯)
// ar_SD 阿拉伯文 (苏丹)
// ar_SY 阿拉伯文 (叙利亚)
// ar_TN 阿拉伯文 (突尼斯)
// ar_YE 阿拉伯文 (也门)
// be 白俄罗斯文
// be_BY 白俄罗斯文 (白俄罗斯)
// bg 保加利亚文
// bg_BG 保加利亚文 (保加利亚)
// ca 加泰罗尼亚文
// ca_ES 加泰罗尼亚文 (西班牙)
// ca_ES_EURO 加泰罗尼亚文 (西班牙,Euro)
// cs 捷克文
// cs_CZ 捷克文 (捷克共和国)
// da 丹麦文
// da_DK 丹麦文 (丹麦)
// de 德文
// de_AT 德文 (奥地利)
// de_AT_EURO 德文 (奥地利,Euro)
// de_CH 德文 (瑞士)
// de_DE 德文 (德国)
// de_DE_EURO 德文 (德国,Euro)
// de_LU 德文 (卢森堡)
// de_LU_EURO 德文 (卢森堡,Euro)
// el 希腊文
// el_GR 希腊文 (希腊)
// en_AU 英文 (澳大利亚)
// en_CA 英文 (加拿大)
// en_GB 英文 (英国)
// en_IE 英文 (爱尔兰)
// en_IE_EURO 英文 (爱尔兰,Euro)
// en_NZ 英文 (新西兰)
// en_ZA 英文 (南非)
// es 西班牙文
// es_BO 西班牙文 (玻利维亚)
// es_AR 西班牙文 (阿根廷)
// es_CL 西班牙文 (智利)
// es_CO 西班牙文 (哥伦比亚)
// es_CR 西班牙文 (哥斯达黎加)
// es_DO 西班牙文 (多米尼加共和国)
// es_EC 西班牙文 (厄瓜多尔)
// es_ES 西班牙文 (西班牙)
// es_ES_EURO 西班牙文 (西班牙,Euro)
// es_GT 西班牙文 (危地马拉)
// es_HN 西班牙文 (洪都拉斯)
// es_MX 西班牙文 (墨西哥)
// es_NI 西班牙文 (尼加拉瓜)
// et 爱沙尼亚文
// es_PA 西班牙文 (巴拿马)
// es_PE 西班牙文 (秘鲁)
// es_PR 西班牙文 (波多黎哥)
// es_PY 西班牙文 (巴拉圭)
// es_SV 西班牙文 (萨尔瓦多)
// es_UY 西班牙文 (乌拉圭)
// es_VE 西班牙文 (委内瑞拉)
// et_EE 爱沙尼亚文 (爱沙尼亚)
// fi 芬兰文
// fi_FI 芬兰文 (芬兰)
// fi_FI_EURO 芬兰文 (芬兰,Euro)
// fr 法文
// fr_BE 法文 (比利时)
// fr_BE_EURO 法文 (比利时,Euro)
// fr_CA 法文 (加拿大)
// fr_CH 法文 (瑞士)
// fr_FR 法文 (法国)
// fr_FR_EURO 法文 (法国,Euro)
// fr_LU 法文 (卢森堡)
// fr_LU_EURO 法文 (卢森堡,Euro)
// hr 克罗地亚文
// hr_HR 克罗地亚文 (克罗地亚)
// hu 匈牙利文
// hu_HU 匈牙利文 (匈牙利)
// is 冰岛文
// is_IS 冰岛文 (冰岛)
// it 意大利文
// it_CH 意大利文 (瑞士)
// it_IT 意大利文 (意大利)
// it_IT_EURO 意大利文 (意大利,Euro)
// iw 希伯来文
// iw_IL 希伯来文 (以色列)
// ja日文
// ja_JP日文 (日本)
// ko 朝鲜文
// ko_KR 朝鲜文 (南朝鲜)
// lt 立陶宛文
// lt_LT 立陶宛文 (立陶宛)
// lv 拉托维亚文(列托)
// lv_LV 拉托维亚文(列托) (拉脱维亚)
// mk 马其顿文
// mk_MK 马其顿文 (马其顿王国)
// nl 荷兰文
// nl_BE 荷兰文 (比利时)
// nl_BE_EURO 荷兰文 (比利时,Euro)
// nl_NL 荷兰文 (荷兰)
// nl_NL_EURO 荷兰文 (荷兰,Euro)
// no 挪威文
// no_NO 挪威文 (挪威)
// no_NO_NY 挪威文 (挪威,Nynorsk)
// pl 波兰文
// pl_PL 波兰文 (波兰)
// pt 葡萄牙文
// pt_BR 葡萄牙文 (巴西)
// pt_PT 葡萄牙文 (葡萄牙)
// pt_PT_EURO 葡萄牙文 (葡萄牙,Euro)
// ro 罗马尼亚文
// ro_RO 罗马尼亚文 (罗马尼亚)
// ru 俄文
// ru_RU 俄文 (俄罗斯)
// sh 塞波尼斯-克罗地亚文
// sh_YU 塞波尼斯-克罗地亚文 (南斯拉夫)
// sk 斯洛伐克文
// sk_SK 斯洛伐克文 (斯洛伐克)
// sl 斯洛文尼亚文
// sl_SI 斯洛文尼亚文 (斯洛文尼亚)
// sq 阿尔巴尼亚文
// sq_AL 阿尔巴尼亚文 (阿尔巴尼亚)
// sr 塞尔维亚文
// sr_YU 塞尔维亚文 (南斯拉夫)
// sv 瑞典文
// sv_SE 瑞典文 (瑞典)
// th 泰文
// th_TH 泰文 (泰国)
// tr 土耳其文
// tr_TR 土耳其文 (土耳其)
// uk 乌克兰文
// uk_UA 乌克兰文 (乌克兰)
// zh 中文
// zh_CN 中文 (中国)
// zh_HK 中文 (香港)
// zh_TW 中文 (台湾)
)

func GetSupportedLang() map[string]string {
	result := make(map[string]string)

	result[en] = en
	result[ar] = ar
	result[de] = de
	result[el] = el
	result[es] = es
	result[it] = it
	result[ja] = ja
	result[fr] = fr
	result[ko] = ko
	result[ru] = ru
	result[th] = th
	result[zh_cn] = zh_cn
	result[zh_hk] = zh_hk
	result[zh_tw] = zh_tw

	return result
}

func IsValidLang(lang string) error {
	switch lang {
	case en:
	case ar:
	case de:
	case fr:
	case ja:
	case es:
	case el:
	case it:
	case ko:
	case ru:
	case th:
	case zh_cn:
	case zh_hk:
	case zh_tw:
	default:
		return e_lang_unsupport
	}
	return nil
}

type Lang struct {
	Selected []string
	langs    map[string]string
}

func (l *Lang) Init() {
	l.langs = make(map[string]string)
}

func (l Lang) MarshalJSON() ([]byte, error) {

	if len(l.langs) == 1 {
		if _, exists := l.langs[default_lang]; exists {
			return json.Marshal(l.langs[default_lang])
		}
	}

	if l.Selected == nil || len(l.Selected) == 0 {
		return json.Marshal(l.langs)
	}

	if len(l.Selected) == 1 {
		lang, exists := l.langs[l.Selected[0]]
		if exists {
			return json.Marshal(lang)
		}

		lang, exists = l.langs[default_lang]
		if exists {
			return json.Marshal(lang)
		}

		return json.Marshal("-")
	} else {
		result := make(map[string]string)

		for _, selected := range l.Selected {
			lang, exists := l.langs[selected]
			if exists {
				result[selected] = lang
			}
		}

		return json.Marshal(result)
	}

}
func (i *Lang) UnmarshalJSON(data []byte) error {

	if bytes.HasPrefix(data, []byte("\"")) && bytes.HasSuffix(data, []byte("\"")) {
		var defaultStr string
		if err := json.Unmarshal(data, &defaultStr); err != nil {
			return err
		}
		i.langs = make(map[string]string)
		i.langs[default_lang] = defaultStr
		return nil
	}

	return json.Unmarshal(data, &i.langs)
}
