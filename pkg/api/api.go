package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-resty/resty/v2"
)

type ZabbixAPI interface {
	RequestCtx(ctx context.Context, method string, params interface{}, result interface{}, omitAuth bool) error
	LoginUserPassCtx(ctx context.Context, params Login) error
	GetHostCtx(ctx context.Context, params ReqHost) ([]Host, error)
	GetProxyCtx(ctx context.Context, params ReqProxy) ([]Proxy, error)
	UpdateProxyCtx(ctx context.Context, params Proxy) ([]string, error)
	GetTriggerCtx(ctx context.Context, params ReqTrigger) ([]Trigger, error)
	Ping(ctx context.Context) error
	Logout() error
}

type zabbixApi struct {
	url       string
	authToken string
	client    *resty.Client
}

func (z *zabbixApi) RequestCtx(ctx context.Context, method string, params interface{}, result interface{}, omitAuth bool) error {
	var zbxResp commonResponse
	var auth string
	if !omitAuth {
		auth = z.authToken
	}
	_, err := z.client.
		R().
		SetContext(ctx).
		SetBody(commonRequest{
			JSONRpcVersion: "2.0",
			Method:         method,
			ID:             1,
			Auth:           auth,
			Params:         params,
		}).
		SetResult(&zbxResp).
		Post(z.url)
	if err != nil {
		return err
	}
	if zbxResp.Error != nil {
		return zbxResp.Error
	} else {
		return json.Unmarshal(zbxResp.Result, result)
	}
}

func (z *zabbixApi) LoginUserPassCtx(ctx context.Context, params Login) error {
	var result string
	if err := z.RequestCtx(ctx, "user.login", params, &result, true); err != nil {
		return err
	} else {
		z.authToken = result
		return nil
	}
}

func (z *zabbixApi) Logout() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var result bool
	return z.RequestCtx(ctx, "user.logout", []string{}, &result, false)
}

func (z *zabbixApi) Ping(ctx context.Context) error {
	var result string
	return z.RequestCtx(ctx, "apiinfo.version", []string{}, &result, true)
}

func (z *zabbixApi) GetProxyCtx(ctx context.Context, params ReqProxy) ([]Proxy, error) {
	var result []Proxy
	err := z.RequestCtx(ctx, "proxy.get", params, &result, false)
	return result, err
}

func (z *zabbixApi) UpdateProxyCtx(ctx context.Context, params Proxy) ([]string, error) {
	var result []string
	err := z.RequestCtx(ctx, "proxy.update", params, &result, false)
	return result, err
}

func (z *zabbixApi) GetHostCtx(ctx context.Context, params ReqHost) ([]Host, error) {
	var result []Host
	err := z.RequestCtx(ctx, "host.get", params, &result, false)
	return result, err
}

func (z *zabbixApi) GetTriggerCtx(ctx context.Context, params ReqTrigger) ([]Trigger, error) {
	var result []Trigger
	err := z.RequestCtx(ctx, "trigger.get", params, &result, false)
	return result, err
}

func GetZabbixAPI(url string) ZabbixAPI {
	return &zabbixApi{
		url:    url,
		client: resty.New(),
	}
}
