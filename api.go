package dwr910m

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DWR910M struct {
	Address string
}

type Message struct {
	Content     string `json:"smsContent"`
	Sender      string `json:"phoneNumber"`
	Date        string `json:"smsDate"`
	ID          int64  `json:"messageid"`
	ClassID     int64  `json:"classid"`
	SingleCount string `json:"singleCount"` // number
	SMSType     string `json:"smsType"`     // number
	TS          int64
}

type Messages struct {
	CurrentPage      int64     `json:"curPage"`
	FirstRecordIndex int64     `json:"startRowNum"`
	LastRecordIndex  int64     `json:"endRowNum"`
	RecordsPerPage   int64     `json:"recordsPerpage"`
	TotalPages       int64     `json:"totalPage"`
	TotalRecords     int64     `json:"totalRecords"`
	Messages         []Message `json:"data"`
}

type Status struct {
	Uptime              string
	ConnectedDevices    int64
	HardwareVersion     string
	IMEI                string
	IPAddress           string
	MacAddress          string
	NetworkType         string
	NetworkMask         string
	SignalStrength      string
	SignalStrengthLevel int64
	SoftwareVersion     string
	SSID                string
	WifiSecurityType    string
	SerialNumber        string
}

// create a new instance of the api
func New(address string) *DWR910M {
	return &DWR910M{Address: address}
}

// get the value of a property by its tag name
func getPropertyByTagName(text, name string) (string, error) {
	start := strings.Index(text, fmt.Sprintf("<%s>", name))
	if start < 0 {
		return "", errors.New("property not found")
	}
	end := strings.Index(text, fmt.Sprintf("</%s>", name))
	if end < 0 {
		return "", errors.New("property not found")
	}
	return strings.TrimSpace(text[start+len(name)+2 : end]), nil
}

// get the values from the dashboard page
func (dwr *DWR910M) GetStatus() (*Status, error) {
	// fetch the dashboard page
	res, e := http.Get(dwr.Address + "/jsonp_dashboard?")
	if e != nil {
		return nil, e
	}
	b, e := io.ReadAll(res.Body)
	if e != nil {
		return nil, e
	}
	res.Body.Close()
	body := string(b)

	status := Status{}

	uptime, e := getPropertyByTagName(body, "functionTimes")
	if e != nil {
		return nil, e
	}
	status.Uptime = uptime

	connectedDevices, e := getPropertyByTagName(body, "hotcount")
	if e != nil {
		return nil, e
	}
	connectedDevicesInt, e := strconv.ParseInt(connectedDevices, 10, 64)
	if e != nil {
		return nil, e
	}
	status.ConnectedDevices = connectedDevicesInt

	hardwareVersion, e := getPropertyByTagName(body, "hwVersion")
	if e != nil {
		return nil, e
	}
	status.HardwareVersion = hardwareVersion

	imei, e := getPropertyByTagName(body, "imei")
	if e != nil {
		return nil, e
	}
	status.IMEI = imei

	ipAddress, e := getPropertyByTagName(body, "ipAddress")
	if e != nil {
		return nil, e
	}
	status.IPAddress = ipAddress

	macAddress, e := getPropertyByTagName(body, "macAddress")
	if e != nil {
		return nil, e
	}
	status.MacAddress = macAddress

	networkType, e := getPropertyByTagName(body, "networkType")
	if e != nil {
		return nil, e
	}
	status.NetworkType = networkType

	networkMask, e := getPropertyByTagName(body, "networkmask")
	if e != nil {
		return nil, e
	}
	status.NetworkMask = networkMask

	signalStrength, e := getPropertyByTagName(body, "strengthDbm")
	if e != nil {
		return nil, e
	}
	status.SignalStrength = signalStrength

	signalStrengthLevel, e := getPropertyByTagName(body, "strengthLevel")
	if e != nil {
		return nil, e
	}
	signalStrengthLevelInt, e := strconv.ParseInt(signalStrengthLevel, 10, 64)
	if e != nil {
		return nil, e
	}
	status.SignalStrengthLevel = signalStrengthLevelInt

	softwareVersion, e := getPropertyByTagName(body, "swVersion")
	if e != nil {
		return nil, e
	}
	status.SoftwareVersion = softwareVersion

	ssid, e := getPropertyByTagName(body, "wifihotname")
	if e != nil {
		return nil, e
	}
	status.SSID = ssid

	wifiSecurityType, e := getPropertyByTagName(body, "wifisafetype")
	if e != nil {
		return nil, e
	}
	status.WifiSecurityType = wifiSecurityType

	serialNumber, e := getPropertyByTagName(body, "sn")
	if e != nil {
		return nil, e
	}
	status.SerialNumber = serialNumber

	return &status, nil
}

// get the sms messages
func (dwr *DWR910M) GetMessages(count int) ([]Message, error) {
	if count < 1 {
		return []Message{}, nil
	}

	// fetch the first page of messages
	res, e := http.Get(dwr.Address + "/PageList?pageIndex=1")
	if e != nil {
		return nil, e
	}

	var m []Message
	var messages Messages

	// parse the response
	e = json.NewDecoder(res.Body).Decode(&messages)
	if e != nil {
		return nil, e
	}

	// save messages to memory
	for _, message := range messages.Messages {
		m = append(m, message)
	}

	// check if the currnet amount of messages satisfies the count
	if len(m) >= count {
		return m[:count], nil
	}

	// fetch the rest of the messages
	for len(m) < count && messages.CurrentPage < messages.TotalPages {
		res, e = http.Get(dwr.Address + fmt.Sprintf("/PageList?pageIndex=%d", messages.CurrentPage+1))
		if e != nil {
			return nil, e
		}
		e = json.NewDecoder(res.Body).Decode(&messages)
		if e != nil {
			return nil, e
		}
		for _, message := range messages.Messages {
			// check if the count is reached
			if len(m) == count {
				break
			}
			m = append(m, message)
		}
	}

	// parse date into unix timestamp
	for i := range m {
		t, e := time.Parse("2006-01-02 15:04:05", m[i].Date)
		if e != nil {
			return nil, e
		}
		m[i].TS = t.UnixMilli()
	}

	return m, nil
}
