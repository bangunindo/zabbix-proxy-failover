package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ProxyStatus int

const (
	ACTIVE  ProxyStatus = 5
	PASSIVE             = 6
)

type ZabbixUnixTime struct {
	time.Time
}

func (ct *ZabbixUnixTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	unix, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	ct.Time = time.Unix(int64(unix), 0)
	return
}

func (ct ZabbixUnixTime) MarshalJSON() ([]byte, error) {
	if ct.Time.Unix() == 0 {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%d\"", ct.Time.Unix())), nil
}
