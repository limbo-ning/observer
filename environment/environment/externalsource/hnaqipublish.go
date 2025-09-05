package externalsource

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"obsessiontech/common/datasource"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/dataprocess"
	"obsessiontech/environment/environment/entity"
	"obsessiontech/environment/environment/monitor"
	"obsessiontech/environment/event"
	"obsessiontech/environment/site"
	"strconv"
	"strings"
	"time"
)

const (
	MODULE_HNAQIPUBLISH = "environment_hnaqipublish"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"
)

type HNAQIPublishModule struct {
	Host   string `json:"host"`
	Cookie string `json:"cookie"`
}

func init() {
	event.Register("hnaqipublish_sync", func() event.IEvent { return new(HNAQIPublishSync) })
}

func GetHNAQIPublishModule(siteID string, flags ...bool) (*HNAQIPublishModule, error) {
	var m *HNAQIPublishModule

	_, sm, err := site.GetSiteModule(siteID, MODULE_HNAQIPUBLISH, flags...)
	if err != nil {
		return nil, err
	}

	paramByte, err := json.Marshal(sm.Param)
	if err != nil {
		log.Println("error marshal environment hnaqipublish module param: ", err)
		return nil, err
	}

	if err := json.Unmarshal(paramByte, &m); err != nil {
		log.Println("error unmarshal environment hnaqipublish module: ", err)
		return nil, err
	}

	return m, nil
}

func (m *HNAQIPublishModule) Save(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		sm, err := site.GetSiteModuleWithTxn(siteID, txn, MODULE_HNAQIPUBLISH, true)
		if err != nil {
			panic(err)
		}

		paramByte, _ := json.Marshal(&m)
		json.Unmarshal(paramByte, &sm.Param)

		if err := sm.Save(siteID, txn); err != nil {
			panic(err)
		}
	})
}

func InitLoginHNAQIPublish(siteID string) (int, string, []byte, error) {
	m, err := GetHNAQIPublishModule(siteID)
	if err != nil {
		return -1, "", nil, err
	}

	URL := fmt.Sprintf("%s/Login/CheckCode?ID=1", m.Host)
	client := &http.Client{}

	log.Println("request hnaqipublish init login:", URL)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		log.Println("error request hnaqipublish init login: ", err)
		return -1, "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Host", strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1))
	req.Header.Set("Referer", m.Host+"/Login/Index")
	req.Header.Set("User-Agent", "ozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request hnaqipublish init login: ", err)
		return -1, "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request hnaqipublish init login: ", err)
		return -1, "", nil, err
	}

	m.Cookie = ""
	for _, c := range resp.Cookies() {
		m.Cookie += fmt.Sprintf("%s=%s; ", c.Name, c.Value)
	}

	log.Println("cookie: ", m.Cookie)

	if err := m.Save(siteID); err != nil {
		return -1, "", nil, err
	}

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func LoginHNAQIPublish(siteID string, usr, pw, imgCode string) (int, string, []byte, error) {

	m, err := GetHNAQIPublishModule(siteID)
	if err != nil {
		return -1, "", nil, err
	}

	URL := fmt.Sprintf("%s/Login/CheckLogin", m.Host)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	postform := make(url.Values)
	postform.Set("usr", usr)
	postform.Set("password", pw)
	postform.Set("imgCode", imgCode)

	req, err := http.NewRequest("POST", URL, strings.NewReader(postform.Encode()))
	if err != nil {
		log.Println("error request hnaqipublish login: ", err)
		return -1, "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", m.Cookie)
	req.Header.Set("Host", strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1))
	req.Header.Set("Cache-Control", "nocache")
	req.Header.Set("Referer", m.Host+"/Login/Index")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len([]byte(postform.Encode()))))

	log.Println("request hnaqipublish login:", postform.Encode(), m.Cookie, strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1), m.Host+"/Login/Index", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36", fmt.Sprintf("%d", len([]byte(postform.Encode()))))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request hnaqipublish login: ", err)
		return -1, "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request hnaqipublish login: ", err)
		return -1, "", nil, err
	}

	for _, c := range resp.Cookies() {
		m.Cookie += fmt.Sprintf("%s=%s; ", c.Name, c.Value)
	}

	log.Println("cookie: ", m.Cookie)

	if err := m.Save(siteID); err != nil {
		return -1, "", nil, err
	}

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func GetHNAQIPublishStationTree(siteID, stationType string) (int, string, []byte, error) {

	m, err := GetHNAQIPublishModule(siteID)
	if err != nil {
		return -1, "", nil, err
	}

	URL := fmt.Sprintf("%s/TreeData/GetStationTreeByType?type=%s", m.Host, stationType)
	client := &http.Client{}

	log.Println("request hnaqipublish station tree:", URL)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		log.Println("error request hnaqipublish station tree: ", err)
		return -1, "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", m.Cookie)
	req.Header.Set("Host", strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1))
	req.Header.Set("Cache-Control", "nocache")
	req.Header.Set("Referer", m.Host+"/Index/Home")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request hnaqipublish station tree: ", err)
		return -1, "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request hnaqipublish station tree: ", err)
		return -1, "", nil, err
	}

	log.Println("get hn aqi publish station tree: ", string(body))

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func GetHNAQIPublishRptData(siteID string, dataType string, stations []string, beginTime, endTime time.Time) (int, string, []byte, error) {

	if len(stations) == 0 {
		return -1, "", nil, errors.New("需要站点")
	}

	var rptDataType string

	switch dataType {
	case data.HOURLY:
		rptDataType = "Hourly"
	case data.DAILY:
		rptDataType = "Daily"
	default:
		return -1, "", nil, errors.New("不正确的省站数据类型")
	}

	m, err := GetHNAQIPublishModule(siteID)
	if err != nil {
		return -1, "", nil, err
	}

	URL := fmt.Sprintf("%s/DataQuery/Get%sRpt", m.Host, rptDataType)
	client := &http.Client{}

	packedStations := make([]string, 0)
	for _, s := range stations {
		packedStations = append(packedStations, fmt.Sprintf("'%s'", s))
	}

	postform := make(url.Values)
	postform.Set("Stns", strings.Join(packedStations, ","))
	postform.Set("dataType", "0")
	postform.Set("beginTime", util.FormatDateTime(beginTime))
	postform.Set("endTime", util.FormatDateTime(endTime))
	postform.Set("choiceType", "isGK")
	postform.Set("sjy", "false")

	log.Println("request hnaqipublish rpt data:", postform.Encode())
	req, err := http.NewRequest("POST", URL, strings.NewReader(postform.Encode()))
	if err != nil {
		log.Println("error request hnaqipublish rpt data: ", err)
		return -1, "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", m.Cookie)
	req.Header.Set("Host", strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1))
	req.Header.Set("Cache-Control", "nocache")
	req.Header.Set("Referer", m.Host+"/Index/Home")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request hnaqipublish rpt data: ", err)
		return -1, "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request hnaqipublish rpt data: ", err)
		return -1, "", nil, err
	}

	log.Println("get hn aqi publish rpt data: ", resp.Header.Get("Content-Type"), string(body))

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func GetHNAQIPublishStatsData(siteID, dataType string, stations []string, beginTime, endTime time.Time) (int, string, []byte, error) {

	if len(stations) == 0 {
		return -1, "", nil, errors.New("需要站点")
	}

	m, err := GetHNAQIPublishModule(siteID)
	if err != nil {
		return -1, "", nil, err
	}

	URL := fmt.Sprintf("%s/ZNStatistics/Get%sRpt", m.Host, dataType)
	client := &http.Client{}

	packedStations := make([]string, 0)
	for _, s := range stations {
		packedStations = append(packedStations, fmt.Sprintf("'%s'", s))
	}

	postform := make(url.Values)
	postform.Set("Stns", strings.Join(packedStations, ","))
	postform.Set("dateType", "Sd")
	postform.Set("beginTime", util.FormatDate(beginTime))
	postform.Set("endTime", util.FormatDate(endTime))
	postform.Set("tcType", "afterTC")
	postform.Set("choiceType", "isGK")
	postform.Set("sjy", "false")
	postform.Set("isSCBNew", "2")

	log.Println("request hnaqipublish stats data:", postform.Encode())
	req, err := http.NewRequest("POST", URL, strings.NewReader(postform.Encode()))
	if err != nil {
		log.Println("error request hnaqipublish stats data: ", err)
		return -1, "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", m.Cookie)
	req.Header.Set("Host", strings.Replace(strings.Replace(m.Host, "http://", "", -1), "https://", "", -1))
	req.Header.Set("Cache-Control", "nocache")
	req.Header.Set("Referer", m.Host+"/ZNStatistics/CompreehensiveIndex")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error request hnaqipublish stats data: ", err)
		return -1, "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error request hnaqipublish stats data: ", err)
		return -1, "", nil, err
	}

	log.Println("get hn aqi publish rpt data: ", resp.Header.Get("Content-Type"), string(body))

	return resp.StatusCode, resp.Header.Get("Content-Type"), body, nil
}

func findStnName(stnlist []interface{}, stnid string) string {

	if stnid == "" {
		return ""
	}

	for _, stn := range stnlist {
		stn, ok := stn.(map[string]interface{})
		if !ok {
			continue
		}
		if stn["id"] == stnid {
			return stn["text"].(string)
		}

		if stn["children"] != nil {
			if childlist, ok := stn["children"].([]interface{}); ok {
				child := findStnName(childlist, stnid)
				if child != "" {
					return child
				}
			}
		}
	}

	return ""
}

func SyncHNAQIPublishRptData(siteID, dataType string, syncTime time.Time, traceBackCount int) error {
	y, m, d := syncTime.Date()
	var beginTime, endTime time.Time

	var fac func() data.IData

	switch dataType {
	case data.HOURLY:
		fac = func() data.IData { return new(data.HourlyData) }
		hour := syncTime.Hour()
		beginTime = time.Date(y, m, d, hour, 0, 0, 0, time.Local).Add(-1 * time.Hour * time.Duration(traceBackCount))
		endTime = time.Date(y, m, d, hour, 0, 0, 0, time.Local)
	case data.DAILY:
		beginTime = time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, -1*traceBackCount)
		endTime = time.Date(y, m, d, 23, 59, 59, 0, time.Local)
		fac = func() data.IData { return new(data.DailyData) }
	default:
		return errors.New("不正确的省站数据类型")
	}

	return syncData(siteID, func(ids []string) (int, string, []byte, error) {
		return GetHNAQIPublishRptData(siteID, dataType, ids, beginTime, endTime)
	}, fac)
}

func SyncHNAQIPublishStatsData(siteID, dataType string, syncTime time.Time, traceBackCount int) error {
	y, m, d := syncTime.Date()
	beginTime := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, -1*traceBackCount)

	tick := time.Time(beginTime)
	for i := 0; i <= traceBackCount; i++ {
		if err := syncData(siteID, func(ids []string) (int, string, []byte, error) {
			return GetHNAQIPublishStatsData(siteID, dataType, ids, tick, tick)
		}, func() data.IData {
			return new(data.DailyData)
		}); err != nil {
			return err
		}

		tick = tick.AddDate(0, 0, 1)
	}

	return nil
}

func syncData(siteID string, fetchFunc func([]string) (int, string, []byte, error), fac func() data.IData) error {
	stations, err := entity.GetStations(siteID, nil, nil, "", "", "hnAQIPublishStn")
	if err != nil {
		return err
	}

	if len(stations) == 0 {
		return nil
	}

	status, contentType, res, err := GetHNAQIPublishStationTree(siteID, "isAll")
	if err != nil {
		return err
	}

	if status != 200 {
		return fmt.Errorf("HN AQI Publish status: %d", status)
	}

	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return fmt.Errorf("HN AQI Publish abnormal response: %s", contentType)
	}

	stnlist := make([]interface{}, 0)
	if err := json.Unmarshal(res, &stnlist); err != nil {
		return err
	}

	stns := make(map[string]*entity.Station)
	stnids := make(map[string]*entity.Station)
	ids := make([]string, 0)

	for _, s := range stations {
		stn, ok := s.Ext["hnAQIPublishStn"].(string)
		if !ok {
			continue
		}

		stnids[stn] = s

		name := findStnName(stnlist, stn)
		if name != "" {
			ids = append(ids, stn)
			stns[name] = s
		}
	}

	log.Println("hnaqi publish name mapping: ", len(ids), stns)

	if len(ids) == 0 {
		return nil
	}

	if err := monitor.LoadMonitorCode(siteID); err != nil {
		return err
	}
	if err := monitor.LoadFlagLimit(siteID); err != nil {
		return err
	}

	status, _, res, err = fetchFunc(ids)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("HN AQI Publish status: %d", status)
	}

	list := make([]map[string]string, 0)
	if err := json.Unmarshal(res, &list); err != nil {
		var datastring string
		if err := json.Unmarshal(res, &datastring); err != nil {
			return err
		} else {
			if err := json.Unmarshal([]byte(datastring), &list); err != nil {
				return err
			}
		}
	}

	log.Println("parsing sync: ", len(list))

	uper := new(dataprocess.Uploader)
	up := new(uploader)

	dataset := make([]data.IData, 0)

	for _, entry := range list {
		var station *entity.Station
		if s, exists := entry["SStation"]; exists {
			station = stnids[s]
		} else if s, exists = entry["StationName"]; exists {
			station = stns[s]
		} else if s, exists = entry["SStationName"]; exists {
			station = stns[s]
		} else {
			log.Println("no station entry")
			continue
		}
		if station == nil {
			log.Println("no station: ", station)
			continue
		}

		var dataTime time.Time
		var err error
		if s, exists := entry["QueryTime"]; exists {
			dataTime, err = util.ParseDateTime(s)
		} else if s, exists = entry["SDatetime"]; exists {
			dataTime, err = util.ParseDate(s)
		} else if s, exists = entry["SDateTime"]; exists {
			parts := strings.Split(s, "至")
			dataTime, err = util.ParseDate(parts[0])
		} else {
			log.Println("no time entry")
			continue
		}
		if err != nil {
			return err
		}

		stationDataSet := make(map[int]data.IData)

		for key, value := range entry {

			if value == "" {
				continue
			}

			mc := monitor.GetMonitorCodeByCode(siteID, station.ID, key)
			if mc == nil {
				log.Println("no monitor: ", key, value, station.ID)
				continue
			}
			d := fac()
			d.SetStationID(station.ID)
			d.SetMonitorID(mc.MonitorID)
			d.SetDataTime(util.Time(dataTime))

			var err error
			v, err := strconv.ParseFloat(strings.Trim(value, "\r\n "), 64)
			if err != nil {
				log.Println("error parse value: ", value, err)
				continue
			}
			if itv, ok := d.(data.IInterval); ok {
				itv.SetAvg(v)
			} else if rtd, ok := d.(data.IRealTime); ok {
				rtd.SetRtd(v)
			}

			d.SetCode(key)

			stationDataSet[mc.MonitorID] = d
			dataset = append(dataset, d)
		}

		primary := entry["PrimaryEP"]
		if primary != "" {
			log.Println("has PrimaryEP: ", primary)
			for _, p := range strings.Split(primary, ",") {
				p = strings.Trim(p, "\n\r ")
				if p != "" {
					pmc := monitor.GetMonitorCodeByCode(siteID, station.ID, p)
					if pmc != nil {
						pd, exists := stationDataSet[pmc.MonitorID]
						if exists {
							pd.SetFlagBit(data.SetFlagBit(pd.GetFlagBit(), monitor.FLAG_PRIMARY_POLLUTANT))
							log.Println("primary ep data flag bit: ", pd.GetFlagBit())
						} else {
							log.Println("no primary ep monitor in dataset: ", p, station.ID, pmc.MonitorID)
						}
					} else {
						log.Println("no primary ep monitor: ", p, station.ID)
					}
				}
			}
		}
	}

	log.Println("data to sync: ", len(dataset))

	if err := uper.UploadBatchData(siteID, up, dataset...); err != nil {
		return err
	}

	if err := uper.UploadUnuploaded(siteID, up); err != nil {
		return err
	}

	return nil
}

type HNAQIPublishSync struct{}

func (h *HNAQIPublishSync) ValidateEvent(siteID string, e *event.Event) error {

	switch e.MainRelateID {
	case "rptData":
		switch e.SubRelateID["dataType"] {
		case data.HOURLY:
		case data.DAILY:
		default:
			return errors.New("不支持的数据类型")
		}
	case "statsRptData":
		switch e.SubRelateID["dataType"] {
		case "MYAvg":
		default:
			return errors.New("不支持的统计类型")
		}
	default:
		return errors.New("不支持的同步项")
	}

	return nil
}

func (h *HNAQIPublishSync) ExecuteEvent(siteID string, txn *sql.Tx, e *event.Event) error {
	switch e.MainRelateID {
	case "rptData":
		traceBackCount, _ := strconv.Atoi(e.SubRelateID["traceBackCount"])

		if err := SyncHNAQIPublishRptData(siteID, e.SubRelateID["dataType"], time.Now(), traceBackCount); err != nil {
			return err
		}
	case "statsRptData":
		traceBackCount, _ := strconv.Atoi(e.SubRelateID["traceBackCount"])

		if err := SyncHNAQIPublishStatsData(siteID, e.SubRelateID["dataType"], time.Now(), traceBackCount); err != nil {
			return err
		}
	}

	return nil
}
