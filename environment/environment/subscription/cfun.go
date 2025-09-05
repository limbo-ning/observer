package subscription

/*
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#ifndef GO_ENV_DATATYPE_H
#define GO_ENV_DATATYPE_H
typedef enum DATATYPE {
    REAL_TIME,
    MINUTELY,
    HOURLY,
    DAILY
} DATATYPE;
#endif

#ifndef GO_ENV_MONITOR_H
typedef struct MONITOR {
    int id;
    char* name;
    char* unit;
} MONITOR;
#endif

#ifndef GO_ENV_DATA_H
typedef struct DATA {
    DATATYPE data_type;
    time_t data_time;
    int station_id;
    int monitor_id;
    double value;
    char* flag;
} DATA;
#endif

#ifndef GO_ENV_ENTITY_H
typedef struct ENTITY {
    int id;
    char* name;
    char* address;
} ENTITY;
#endif

#ifndef GO_ENV_STATION_H
typedef struct STATION {
    int id;
    int entity_id;
    char* name;
} STATION;
#endif

#ifndef GO_ENV_PUSH_H
typedef enum SUBSTYPE {
	STATION_STATUS,
	DATA_DAILY,
	DATA_HOURLY,
	DATA_MINUTELY,
	DATA_REAL_TIME
} SUBSTYPE;

typedef char* (*GET_MONITOR_NAME) (char*, int);
typedef char* (*GET_MONITOR_FLAG_NAME) (char*, char*);
typedef double (*GET_RECENT_DATA) (char*, char*, int, int);
typedef char* (*GET_MONITOR_FLAG_LIMIT) (char*, int, char*);

typedef struct ENV_CALLBACK {
	GET_MONITOR_NAME get_monitor_name;
	GET_MONITOR_FLAG_NAME get_monitor_flag_name;
	GET_RECENT_DATA get_recent_data;
	GET_MONITOR_FLAG_LIMIT get_monitor_flag_limit;
} ENV_CALLBACK;
#endif

extern double getRecentDataValue(char *, char*, int, int);
extern char* getMonitorFlagLimit(char*, int, int, char*);
extern char* getMonitorName(char*, int);
extern char* getMonitorFlagName(char*, char*);

typedef int (*GET_SMS_PARAM) (char*, char ** [], char ** [], SUBSTYPE, int, ENTITY, STATION, time_t, DATA [], size_t, ENV_CALLBACK);
int GetSMSParamCall(void * f, char * siteid, char ** ptr_keys [], char ** ptr_values [], SUBSTYPE sub, int is_cease, ENTITY e, STATION s, time_t t, DATA datas [], size_t data_len, ENV_CALLBACK cb) {
	return ((GET_SMS_PARAM)f) (siteid, ptr_keys, ptr_values, sub, is_cease, e, s, t, datas,  data_len, cb);
}

typedef char* (*GET_WX_OPEN_TEMPLATE_FIRST)(char*, SUBSTYPE, int);
char* GetWxOpenTemplateFirstCall(void* f, char* siteid, SUBSTYPE sub, int is_cease) {
	return ((GET_WX_OPEN_TEMPLATE_FIRST)f)(siteid, sub, is_cease);
}

typedef char* (*GET_WX_OPEN_TEMPLATE_REMARK)(char*, SUBSTYPE, int);
char* GetWxOpenTemplateRemarkCall(void* f, char* siteid, SUBSTYPE sub, int is_cease) {
	return ((GET_WX_OPEN_TEMPLATE_REMARK)f)(siteid, sub, is_cease);
}

typedef int (*GET_WX_OPEN_TEMPLATE_KEYWORDS) (char*, char ** [], SUBSTYPE, int, ENTITY, STATION, time_t, DATA [], size_t, ENV_CALLBACK);
int GetWxOpenTemplateKeywordsCall(void * f, char * siteid, char ** ptr_keys [], SUBSTYPE sub, int is_cease, ENTITY e, STATION s, time_t t, DATA datas [], size_t data_len, ENV_CALLBACK cb) {
	return ((GET_WX_OPEN_TEMPLATE_KEYWORDS)f) (siteid, ptr_keys, sub, is_cease, e, s, t, datas,  data_len, cb);
}

*/
import "C"
import (
	"errors"
	"log"
	"obsessiontech/common/cgo/dl"
	"obsessiontech/environment/environment/data"
	"obsessiontech/environment/environment/entity"
	"time"
	"unsafe"
)

func parseStringListFromArrayPtr(array **C.char, cSize C.int) []string {
	result := make([]string, 0)

	size := int(cSize)

	if size > 0 {
		for i, s := range (*[1 << 28]*C.char)(unsafe.Pointer(array))[:size:size] {
			goStr := C.GoString(s)
			log.Printf("parsing string array: [%d] - [%s]", i, goStr)
			C.free(unsafe.Pointer(s))
			result = append(result, goStr)
		}
	}

	return result
}

func callLib(libpath, symbol string, process func(unsafe.Pointer) error) (err error) {
	var handle unsafe.Pointer
	var cancel func()

	handle, cancel, err = dl.OpenDL(libpath, symbol)
	if err != nil {
		return err
	}

	defer func() {
		cancel()
		if e := recover(); e != nil {
			log.Println("fatal error panic: ", libpath, e)
			err = errors.New("fatal error")
		}
	}()

	return process(handle)
}

func getSMSParam(siteID, libpath, subscriptionType string, isCease bool, e *entity.Entity, s *entity.Station, t time.Time, dataList []data.IData) (map[string]string, error) {

	result := make(map[string]string)

	if err := callLib(libpath, "get_sms_param", func(handle unsafe.Pointer) (err error) {

		defer func() {
			if e := recover(); e != nil {
				log.Println("fatal error sms_template_code panic: ", libpath, e)
				err = errors.New("fatal error")
			}
		}()

		cSiteID := C.CString(siteID)
		defer C.free(unsafe.Pointer(cSiteID))

		var keys **C.char
		var values **C.char

		cEntity := C.ENTITY{
			id:      C.int(e.ID),
			name:    C.CString(e.Name),
			address: C.CString(e.Address),
		}

		defer C.free(unsafe.Pointer(cEntity.name))
		defer C.free(unsafe.Pointer(cEntity.address))

		cStation := C.STATION{
			id:   C.int(s.ID),
			name: C.CString(s.Name),
		}

		defer C.free(unsafe.Pointer(cStation.name))

		var cSubsType C.SUBSTYPE
		switch subscriptionType {
		case STATION_STATUS:
			cSubsType = C.STATION_STATUS
		case DATA_DAILY:
			cSubsType = C.DATA_DAILY
		case DATA_HOURLY:
			cSubsType = C.DATA_HOURLY
		case DATA_MINUTELY:
			cSubsType = C.DATA_MINUTELY
		case DATA_REAL_TIME:
			cSubsType = C.DATA_REAL_TIME
		}

		var datasPointer *C.DATA
		if len(dataList) > 0 {
			datas := make([]C.DATA, len(dataList))
			for i, d := range dataList {
				cd := C.DATA{}
				switch d.GetDataType() {
				case data.REAL_TIME:
					cd.data_type = C.REAL_TIME
					cd.value = C.double(d.(data.IRealTime).GetRtd())
				case data.MINUTELY:
					cd.data_type = C.MINUTELY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				case data.HOURLY:
					cd.data_type = C.HOURLY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				case data.DAILY:
					cd.data_type = C.DAILY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				}
				cd.data_time = C.long(time.Time(d.GetDataTime()).Unix())
				cd.monitor_id = C.int(d.GetMonitorID())
				cd.station_id = C.int(d.GetStationID())
				cd.flag = C.CString(d.GetFlag())
				defer C.free(unsafe.Pointer(cd.flag))
				datas[i] = cd
			}
			datasPointer = (*C.DATA)(unsafe.Pointer(&datas[0]))
		}

		cb := C.ENV_CALLBACK{}
		cb.get_monitor_name = C.GET_MONITOR_NAME(C.getMonitorName)
		cb.get_monitor_flag_name = C.GET_MONITOR_FLAG_NAME(C.getMonitorFlagName)
		cb.get_monitor_flag_limit = C.GET_MONITOR_FLAG_LIMIT(C.getMonitorFlagLimit)
		cb.get_recent_data = C.GET_RECENT_DATA(C.getRecentDataValue)

		cIsCease := C.int(0)
		if isCease {
			cIsCease = C.int(1)
		}

		log.Println("get param call: ", t, t.Unix())

		size, err := C.GetSMSParamCall(handle, cSiteID, &keys, &values, cSubsType, cIsCease, cEntity, cStation, C.long(t.Unix()), datasPointer, C.ulong(len(dataList)), cb)
		if err != nil {
			log.Println("error  call cgo get sms param: ", err)
			// return err
		}

		keyList := parseStringListFromArrayPtr(keys, size)
		valueList := parseStringListFromArrayPtr(values, size)

		for i, k := range keyList {
			result[k] = valueList[i]
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func getWxOpenTemplateFirst(siteID, libpath, subscriptionType string, isCease bool) (string, error) {
	var result string
	if err := callLib(libpath, "get_wx_open_template_first", func(handle unsafe.Pointer) error {
		cSiteID := C.CString(siteID)
		defer C.free(unsafe.Pointer(cSiteID))

		var cSubsType C.SUBSTYPE
		switch subscriptionType {
		case STATION_STATUS:
			cSubsType = C.STATION_STATUS
		case DATA_DAILY:
			cSubsType = C.DATA_DAILY
		case DATA_HOURLY:
			cSubsType = C.DATA_HOURLY
		case DATA_MINUTELY:
			cSubsType = C.DATA_MINUTELY
		case DATA_REAL_TIME:
			cSubsType = C.DATA_REAL_TIME
		}

		cIsCease := C.int(0)
		if isCease {
			cIsCease = C.int(1)
		}

		ret, err := C.GetWxOpenTemplateFirstCall(handle, cSiteID, cSubsType, cIsCease)
		if err != nil {
			log.Println("error call cgo get template first: ", err)
			// return err
		}
		defer C.free(unsafe.Pointer(ret))

		result = C.GoString(ret)

		return nil
	}); err != nil {
		return "", err
	}

	return result, nil
}

func getWxOpenTemplateRemark(siteID, libpath, subscriptionType string, isCease bool) (string, error) {
	var result string
	if err := callLib(libpath, "get_wx_open_template_remark", func(handle unsafe.Pointer) error {
		cSiteID := C.CString(siteID)
		defer C.free(unsafe.Pointer(cSiteID))

		var cSubsType C.SUBSTYPE
		switch subscriptionType {
		case STATION_STATUS:
			cSubsType = C.STATION_STATUS
		case DATA_DAILY:
			cSubsType = C.DATA_DAILY
		case DATA_HOURLY:
			cSubsType = C.DATA_HOURLY
		case DATA_MINUTELY:
			cSubsType = C.DATA_MINUTELY
		case DATA_REAL_TIME:
			cSubsType = C.DATA_REAL_TIME
		}

		cIsCease := C.int(0)
		if isCease {
			cIsCease = C.int(1)
		}

		ret, err := C.GetWxOpenTemplateRemarkCall(handle, cSiteID, cSubsType, cIsCease)
		if err != nil {
			log.Println("error call cgo get template remark: ", err)
			// return err
		}
		defer C.free(unsafe.Pointer(ret))

		result = C.GoString(ret)

		return nil
	}); err != nil {
		return "", err
	}

	return result, nil
}

func getWxOpenTemplateKeywords(siteID, libpath, subscriptionType string, isCease bool, e *entity.Entity, s *entity.Station, t time.Time, dataList []data.IData) ([]string, error) {
	var result []string
	if err := callLib(libpath, "get_wx_open_template_keywords", func(handle unsafe.Pointer) error {
		cSiteID := C.CString(siteID)
		defer C.free(unsafe.Pointer(cSiteID))

		var keys **C.char

		cEntity := C.ENTITY{
			id:      C.int(e.ID),
			name:    C.CString(e.Name),
			address: C.CString(e.Address),
		}

		defer C.free(unsafe.Pointer(cEntity.name))
		defer C.free(unsafe.Pointer(cEntity.address))

		cStation := C.STATION{
			id:   C.int(s.ID),
			name: C.CString(s.Name),
		}

		defer C.free(unsafe.Pointer(cStation.name))

		var cSubsType C.SUBSTYPE
		switch subscriptionType {
		case STATION_STATUS:
			cSubsType = C.STATION_STATUS
		case DATA_DAILY:
			cSubsType = C.DATA_DAILY
		case DATA_HOURLY:
			cSubsType = C.DATA_HOURLY
		case DATA_MINUTELY:
			cSubsType = C.DATA_MINUTELY
		case DATA_REAL_TIME:
			cSubsType = C.DATA_REAL_TIME
		}

		cIsCease := C.int(0)
		if isCease {
			cIsCease = C.int(1)
		}

		var datasPointer *C.DATA
		if len(dataList) > 0 {
			datas := make([]C.DATA, len(dataList))
			for i, d := range dataList {
				cd := C.DATA{}
				switch d.GetDataType() {
				case data.REAL_TIME:
					cd.data_type = C.REAL_TIME
					cd.value = C.double(d.(data.IRealTime).GetRtd())
				case data.MINUTELY:
					cd.data_type = C.MINUTELY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				case data.HOURLY:
					cd.data_type = C.HOURLY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				case data.DAILY:
					cd.data_type = C.DAILY
					cd.value = C.double(d.(data.IInterval).GetAvg())
				}
				cd.data_time = C.long(time.Time(d.GetDataTime()).Unix())
				cd.monitor_id = C.int(d.GetMonitorID())
				cd.station_id = C.int(d.GetStationID())
				cd.flag = C.CString(d.GetFlag())
				defer C.free(unsafe.Pointer(cd.flag))
				datas[i] = cd
			}
			datasPointer = (*C.DATA)(unsafe.Pointer(&datas[0]))
		}

		cb := C.ENV_CALLBACK{}
		cb.get_monitor_name = C.GET_MONITOR_NAME(C.getMonitorName)
		cb.get_monitor_flag_name = C.GET_MONITOR_FLAG_NAME(C.getMonitorFlagName)
		cb.get_monitor_flag_limit = C.GET_MONITOR_FLAG_LIMIT(C.getMonitorFlagLimit)
		cb.get_recent_data = C.GET_RECENT_DATA(C.getRecentDataValue)

		size, err := C.GetWxOpenTemplateKeywordsCall(handle, cSiteID, &keys, cSubsType, cIsCease, cEntity, cStation, C.long(t.Unix()), datasPointer, C.ulong(len(dataList)), cb)
		if err != nil {
			log.Println("error call cgo get template keywords: ", err)
			// return err
		}

		result = parseStringListFromArrayPtr(keys, size)

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
