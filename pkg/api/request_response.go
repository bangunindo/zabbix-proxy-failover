package api

import (
	"context"
	"encoding/json"
	"strconv"
)

type RequestError interface {
	Code() int
	Message() string
	Data() string
}

type requestError struct {
	RespCode    int    `json:"code"`
	RespMessage string `json:"message"`
	RespData    string `json:"data"`
}

func (r requestError) Error() string {
	return "apierror " + r.RespMessage + r.RespData
}

func (r requestError) Code() int {
	return r.RespCode
}

func (r requestError) Message() string {
	return r.RespMessage
}

func (r requestError) Data() string {
	return r.RespData
}

type commonRequest struct {
	JSONRpcVersion string      `json:"jsonrpc"`
	Method         string      `json:"method"`
	ID             int         `json:"id"`
	Auth           string      `json:"auth,omitempty"`
	Params         interface{} `json:"params"`
}

type commonResponse struct {
	JSONRpcVersion string          `json:"jsonrpc"`
	ID             int             `json:"id"`
	Result         json.RawMessage `json:"result"`
	Error          *requestError   `json:"error"`
}

type Login struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type TagOperator int

const (
	CONTAINS  TagOperator = 0
	EQUALS                = 1
	NOTLIKE               = 2
	NOTEQUAL              = 3
	EXISTS                = 4
	NOTEXISTS             = 5
)

var operatorToString = map[TagOperator]string{
	CONTAINS:  "contains",
	EQUALS:    "equals",
	NOTLIKE:   "not_like",
	NOTEQUAL:  "not_equal",
	EXISTS:    "exists",
	NOTEXISTS: "notexists",
}

func (t TagOperator) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(strconv.Itoa(int(t)))
	return data, err
}

func (t *TagOperator) UnmarshalJSON(data []byte) error {
	var valStr string
	if err := json.Unmarshal(data, &valStr); err != nil {
		return err
	} else {
		if val, err := strconv.Atoi(valStr); err != nil {
			return err
		} else {
			*t = TagOperator(val)
			return nil
		}
	}
}

func (t TagOperator) String() string {
	return operatorToString[t]
}

type TriggerValue int

const (
	OK      TriggerValue = 0
	PROBLEM              = 1
)

var triggerToString = map[TriggerValue]string{
	OK:      "ok",
	PROBLEM: "problem",
}

func (t TriggerValue) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(strconv.Itoa(int(t)))
	return data, err
}

func (t *TriggerValue) UnmarshalJSON(data []byte) error {
	var valStr string
	if err := json.Unmarshal(data, &valStr); err != nil {
		return err
	} else {
		if val, err := strconv.Atoi(valStr); err != nil {
			return err
		} else {
			*t = TriggerValue(val)
			return nil
		}
	}
}

func (t TriggerValue) String() string {
	return triggerToString[t]
}

type Tag struct {
	Tag      string       `json:"tag"`
	Value    string       `json:"value"`
	Operator *TagOperator `json:"operator,omitempty"`
}

type Host struct {
	HostID  int    `json:"hostid,string"`
	Host    string `json:"host"`
	ProxyID int    `json:"proxy_hostid,string"`
	Tag     []Tag  `json:"tags"`
}

type Proxy struct {
	ProxyID     int            `json:"proxyid,string"`
	Host        string         `json:"host,omitempty"`
	Status      ProxyStatus    `json:"status,string,omitempty"`
	LastAccess  ZabbixUnixTime `json:"lastaccess,omitempty"`
	Description string         `json:"description,omitempty"`
	Hosts       []int          `json:"hosts,omitempty"`
}

type Trigger struct {
	TriggerID  int            `json:"triggerid,string"`
	Value      *TriggerValue  `json:"value"`
	LastChange ZabbixUnixTime `json:"lastchange"`
}

func (p *Proxy) UpdateHosts(ctx context.Context, z ZabbixAPI, hostList []Host) error {
	var hostListReq []int
	for _, host := range hostList {
		hostListReq = append(hostListReq, host.HostID)
	}
	_, err := z.UpdateProxyCtx(ctx, Proxy{
		ProxyID: p.ProxyID,
		Hosts:   hostListReq,
	})
	return err
}

func (p *Proxy) GetHosts(ctx context.Context, z ZabbixAPI) ([]Host, error) {
	return z.GetHostCtx(ctx, ReqHost{
		ProxyIDs:   []int{p.ProxyID},
		SelectTags: []string{"tag", "value"},
		Output:     []string{"hostid", "host", "proxy_hostid"},
	})
}

type ReqProxy struct {
	ProxyIDs []int `json:"proxyids,omitempty"`
}

type ReqHost struct {
	ProxyIDs   []int    `json:"proxyids,omitempty"`
	SelectTags []string `json:"selectTags,omitempty"`
	Tags       []Tag    `json:"tags,omitempty"`
	Output     []string `json:"output,omitempty"`
}

type ReqTrigger struct {
	TriggerIDs []int    `json:"triggerids,omitempty"`
	Output     []string `json:"output,omitempty"`
}
