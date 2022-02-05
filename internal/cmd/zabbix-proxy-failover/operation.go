package zabbix_proxy_failover

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	iconfig "github.com/bangunindo/zabbix-proxy-failover/internal/config"
	"github.com/bangunindo/zabbix-proxy-failover/pkg/api"
	logger "github.com/sirupsen/logrus"
	"sort"
	"strconv"
	"strings"
	"time"

	wa "github.com/bangunindo/zabbix-proxy-failover/pkg/walker-alias"
)

type HostListOp struct {
	list         []*api.Host
	index        map[int]int
	indexByProxy map[int][]int
}

func (h *HostListOp) Len() int {
	return len(h.list)
}

func (h *HostListOp) List() []*api.Host {
	return h.list
}

func (h *HostListOp) Append(host *api.Host) {
	if h.index == nil {
		h.index = make(map[int]int)
	}
	if h.indexByProxy == nil {
		h.indexByProxy = make(map[int][]int)
	}
	h.list = append(h.list, host)
	idx := len(h.list) - 1
	h.index[host.HostID] = idx
	h.indexByProxy[host.ProxyID] = append(h.indexByProxy[host.ProxyID], idx)
}

func (h *HostListOp) RebuildIndex() {
	h.index = make(map[int]int)
	h.indexByProxy = make(map[int][]int)
	for i, host := range h.list {
		h.index[host.HostID] = i
		h.indexByProxy[host.ProxyID] = append(h.indexByProxy[host.ProxyID], i)
	}
}

func (h *HostListOp) SwapHost(h2 *HostListOp, i, j int) {
	h.list[i], h2.list[j] = h2.list[j], h.list[i]
}

func (h *HostListOp) HostsInProxy(proxyID int) *HostListOp {
	hostList := new(HostListOp)
	for idx := range h.indexByProxy[proxyID] {
		hostList.Append(h.list[idx])
	}
	return hostList
}

func (h *HostListOp) ByHostIDIdx(hostID int) (int, error) {
	idx, ok := h.index[hostID]
	if !ok {
		return 0, errors.New("can't find hostid")
	} else {
		return idx, nil
	}
}

func (h *HostListOp) MD5() string {
	hasher := md5.New()
	hasher.Write([]byte(" "))
	orderedHost := h.list[:]
	sort.Slice(orderedHost, func(i, j int) bool {
		return orderedHost[i].HostID < orderedHost[j].HostID
	})
	for _, host := range orderedHost {
		hasher.Write([]byte(strconv.Itoa(host.HostID)))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

type ProxyOp struct {
	ConfProxy   iconfig.Proxy
	ApiProxy    api.Proxy
	Hosts       HostListOp
	PlannedHost HostListOp
	IndexedTag  map[string]map[string]bool
	isDown      bool
	lockHost    bool
}

func (p *ProxyOp) MarkAs(isDown bool) {
	p.isDown = isDown
}

func (p *ProxyOp) IsDown() bool {
	return p.isDown
}

func (p *ProxyOp) LockHost() bool {
	return p.lockHost
}

func (p *ProxyOp) Equals(proxy *ProxyOp) bool {
	return p.ApiProxy.ProxyID == proxy.ApiProxy.ProxyID
}

type ProxyListOp struct {
	list          []*ProxyOp
	bucketKey     string
	indexByID     map[int]int
	indexByStatus map[bool][]int
	indexByTag    map[[2]string][]int
	indexByHostID map[int]int
	totalWeight   int
}

func (p *ProxyListOp) BucketKey() string {
	if p.bucketKey == "" {
		var proxyIDs []string
		sortedProxy := p.list[:]
		sort.Slice(sortedProxy, func(i, j int) bool {
			return sortedProxy[i].ApiProxy.ProxyID < sortedProxy[j].ApiProxy.ProxyID
		})
		for _, proxy := range sortedProxy {
			proxyIDs = append(proxyIDs, strconv.Itoa(proxy.ApiProxy.ProxyID))
		}
		p.bucketKey = strings.Join(proxyIDs, ",")
	}
	return p.bucketKey
}

func (p *ProxyListOp) List() []*ProxyOp {
	return p.list
}

func (p *ProxyListOp) Len() int {
	return len(p.list)
}

func (p *ProxyListOp) TotalWeight() int {
	return p.totalWeight
}

func (p *ProxyListOp) MarkAs(proxy *ProxyOp, isDown bool) {
	proxy.MarkAs(isDown)
	p.indexByStatus = make(map[bool][]int)
	for i, proxy := range p.list {
		p.indexByStatus[proxy.isDown] = append(p.indexByStatus[proxy.isDown], i)
	}
}

func (p *ProxyListOp) Append(proxy *ProxyOp) {
	if p.indexByID == nil {
		p.indexByID = make(map[int]int)
	}
	if p.indexByStatus == nil {
		p.indexByStatus = make(map[bool][]int)
	}
	if p.indexByTag == nil {
		p.indexByTag = make(map[[2]string][]int)
	}
	if p.indexByHostID == nil {
		p.indexByHostID = make(map[int]int)
	}
	p.list = append(p.list, proxy)
	lastIdx := len(p.list) - 1
	p.indexByID[proxy.ConfProxy.ProxyID] = lastIdx
	p.indexByStatus[proxy.isDown] = append(p.indexByStatus[proxy.isDown], lastIdx)
	for _, tag := range proxy.ConfProxy.Tag {
		key := [2]string{tag.Name, tag.Value}
		p.indexByTag[key] = append(p.indexByTag[key], lastIdx)
	}
	for _, host := range proxy.Hosts.List() {
		p.indexByHostID[host.HostID] = lastIdx
	}
	p.totalWeight += proxy.ConfProxy.Weight
}

func (p *ProxyListOp) SearchByID(proxyID int) *ProxyOp {
	if idx, ok := p.indexByID[proxyID]; ok {
		return p.list[idx]
	} else {
		return nil
	}
}

func hostProxyTagMatch(host *api.Host, proxy *ProxyOp) bool {
	if len(proxy.IndexedTag) > 0 && len(host.Tag) == 0 {
		return false
	}
	match := true
	for _, hTag := range host.Tag {
		if pTag, ok := proxy.IndexedTag[hTag.Tag]; ok && !pTag[hTag.Value] {
			match = false
		}
	}
	return match
}

func (p *ProxyListOp) ProxyMatchByTag(host *api.Host, excludeDown bool) ProxyListOp {
	var proxyList ProxyListOp
	for _, proxy := range p.list {
		if excludeDown && proxy.IsDown() {
			continue
		}
		if hostProxyTagMatch(host, proxy) {
			proxyList.Append(proxy)
		}
	}
	return proxyList
}

type Operation struct {
	allProxies ProxyListOp
	allHosts   HostListOp
}

func (o *Operation) LoadProxy(ctx context.Context, z api.ZabbixAPI, proxies []api.Proxy) error {
	log := logger.WithField("module", "operation.load_proxy")
	proxyIDList := map[int]bool{}
	apiProxyByID := map[int]api.Proxy{}
	log.Debug("populating proxies")
	for _, proxy := range proxies {
		apiProxyByID[proxy.ProxyID] = proxy
		proxyIDList[proxy.ProxyID] = true
	}
	confProxyByID := map[int]iconfig.Proxy{}
	for _, proxy := range config.Proxy {
		confProxyByID[proxy.ProxyID] = proxy
		proxyIDList[proxy.ProxyID] = true
	}
	for proxyID := range proxyIDList {
		logProxy := log.WithField("proxy_id", proxyID)
		logProxy.Debug("joining proxy")
		if _, ok := confProxyByID[proxyID]; !ok {
			return errors.New(fmt.Sprintf("proxy_id %d not found in conf", proxyID))
		}
		if _, ok := apiProxyByID[proxyID]; !ok {
			return errors.New(fmt.Sprintf("proxy_id %d not found in api", proxyID))
		}
		proxy := ProxyOp{
			ConfProxy: confProxyByID[proxyID],
			ApiProxy:  apiProxyByID[proxyID],
		}
		ctxTimeout, cancel := ctxRequestTimeout(ctx)
		logProxy.Debug("populating host information")
		if hosts, err := proxy.ApiProxy.GetHosts(ctxTimeout, z); err != nil {
			cancel()
			return err
		} else {
			cancel()
			for _, h := range hosts {
				var host api.Host
				host = h
				proxy.Hosts.Append(&host)
				o.allHosts.Append(&host)
			}
		}
		proxyTag := map[string]map[string]bool{}
		for _, pTag := range proxy.ConfProxy.Tag {
			if _, ok := proxyTag[pTag.Name]; !ok {
				proxyTag[pTag.Name] = map[string]bool{}
			}
			proxyTag[pTag.Name][pTag.Value] = true
		}
		proxy.IndexedTag = proxyTag
		o.allProxies.Append(&proxy)
	}
	return nil
}

func (o *Operation) EvaluateStatus(ctx context.Context, z api.ZabbixAPI) error {
	now := time.Now()
	log := logger.WithField("module", "operation.evaluate_status")
	for _, proxy := range o.allProxies.List() {
		logProxy := log.WithField("proxy_id", proxy.ApiProxy.ProxyID)
		logProxy.Debug("evaluating")
		switch config.Failover.Method {
		case iconfig.TRIGGER:
			logProxy.Debug("evaluating trigger")
			ctxTimeout, cancel := ctxRequestTimeout(ctx)
			if trigger, err := z.GetTriggerCtx(ctxTimeout, api.ReqTrigger{
				TriggerIDs: []int{*proxy.ConfProxy.TriggerID},
				Output:     []string{"triggerid", "value", "lastchange"},
			}); err != nil {
				cancel()
				return err
			} else if len(trigger) == 0 {
				cancel()
				return errors.New("no trigger returned")
			} else if trigger[0].Value == nil {
				cancel()
				return errors.New("trigger value not found")
			} else {
				cancel()
				if *trigger[0].Value == api.PROBLEM {
					o.allProxies.MarkAs(proxy, true)
				}
				if config.Failover.LockHostAfter != nil &&
					now.Sub(trigger[0].LastChange.Time) >= *config.Failover.LockHostAfter {
					proxy.lockHost = true
				}
			}
		case iconfig.LASTSEEN:
			logProxy.Debug("evaluating last_seen")
			if now.Sub(proxy.ApiProxy.LastAccess.Time) >= *config.Failover.Duration {
				o.allProxies.MarkAs(proxy, true)
			}
		}
	}
	return nil
}

type HostRouteStats struct {
	ProxyDown       int
	NoEligibleProxy int
	DoNothing       int
	Balanced        int
	TagChange       int
}

type HostRoute struct {
	proxyBucket       map[string]map[int]*HostListOp
	proxyBucketRandom map[string]*wa.WalkerAlias
	stats             HostRouteStats
	seed              int64
}

func (h *HostRoute) Route(host *api.Host, proxyList ProxyListOp) error {
	if h.seed == 0 {
		h.seed = time.Now().Unix()
	}
	if h.proxyBucket == nil {
		h.proxyBucket = make(map[string]map[int]*HostListOp)
	}
	if h.proxyBucketRandom == nil {
		h.proxyBucketRandom = make(map[string]*wa.WalkerAlias)
	}
	prevProxy := proxyList.SearchByID(host.ProxyID)
	eligibleProxy := proxyList.ProxyMatchByTag(host, true)
	if eligibleProxy.Len() == 0 || eligibleProxy.TotalWeight() == 0 {
		h.stats.NoEligibleProxy++
		prevProxy.PlannedHost.Append(host)
	} else if eligibleProxy.Len() == 1 {
		destProxy := eligibleProxy.List()[0]
		if destProxy.Equals(prevProxy) {
			h.stats.DoNothing++
		} else if prevProxy.IsDown() {
			h.stats.ProxyDown++
		} else {
			h.stats.TagChange++
		}
		destProxy.PlannedHost.Append(host)
	} else if (!config.Failover.AlwaysBalance || prevProxy.LockHost()) && !prevProxy.IsDown() {
		h.stats.DoNothing++
		prevProxy.PlannedHost.Append(host)
	} else {
		bucketKey := eligibleProxy.BucketKey()
		bucket, ok := h.proxyBucket[bucketKey]
		if !ok {
			bucket = make(map[int]*HostListOp)
			for _, proxy := range eligibleProxy.List() {
				bucket[proxy.ConfProxy.ProxyID] = new(HostListOp)
			}
			h.proxyBucket[bucketKey] = bucket
		}
		var random *wa.WalkerAlias
		random, ok = h.proxyBucketRandom[bucketKey]
		if !ok {
			probMap := map[int]float64{}
			for _, proxy := range eligibleProxy.List() {
				probMap[proxy.ConfProxy.ProxyID] = float64(proxy.ConfProxy.Weight)
			}
			var err error
			random, err = wa.NewWalkerAlias(probMap, h.seed)
			if err != nil {
				return err
			}
			h.proxyBucketRandom[bucketKey] = random
		}
		bucket[random.Random()].Append(host)
	}
	return nil
}

// PreferSticky will shuffle the host bucket, so it will prefer to the previous proxy
func (h *HostRoute) PreferSticky(proxyList ProxyListOp) error {
	log := logger.WithField("module", "host_route.prefer_sticky")
	for bucketName, bucket := range h.proxyBucket {
		logBucket := log.WithField("bucket", bucketName)
		logBucket.Debug("start analyzing bucket")
		for proxyID1, hostList1 := range bucket {
			for proxyID2, hostList2 := range bucket {
				if proxyID1 == proxyID2 ||
					hostList1.Len() == 0 ||
					hostList2.Len() == 0 {
					continue
				}
				proxyID1HostsInID2 := hostList2.HostsInProxy(proxyID1)
				proxyID2HostsInID1 := hostList1.HostsInProxy(proxyID2)
				if proxyID1HostsInID2.Len() > 0 && proxyID2HostsInID1.Len() > 0 {
					logBucket.Debug("swapping hosts between proxy ", proxyID1, " and ", proxyID2)
					var maxSwap int
					if proxyID1HostsInID2.Len() > proxyID2HostsInID1.Len() {
						maxSwap = proxyID2HostsInID1.Len()
					} else {
						maxSwap = proxyID1HostsInID2.Len()
					}
					for i := 0; i < maxSwap; i++ {
						h1 := proxyID2HostsInID1.List()[i]
						h2 := proxyID1HostsInID2.List()[i]
						h1Idx, err1 := hostList1.ByHostIDIdx(h1.HostID)
						h2Idx, err2 := hostList2.ByHostIDIdx(h2.HostID)
						if err1 != nil {
							return err1
						}
						if err2 != nil {
							return err2
						}
						hostList1.SwapHost(hostList2, h1Idx, h2Idx)
					}
					hostList1.RebuildIndex()
					hostList2.RebuildIndex()
				} else {
					logBucket.Debug("proxy ", proxyID1, " and ", proxyID2, " don't have matches")
				}
			}
		}
		for proxyID, hostList := range bucket {
			proxy := proxyList.SearchByID(proxyID)
			for _, host := range hostList.List() {
				if host.ProxyID == proxy.ApiProxy.ProxyID {
					h.stats.DoNothing++
				} else {
					h.stats.Balanced++
				}
				proxy.PlannedHost.Append(host)
			}
		}
	}
	return nil
}

func (o *Operation) HostPlanning() error {
	log := logger.WithField("module", "operation.host_plan")
	log.Debug("starting host planning")
	var hostRoute HostRoute
	for _, host := range o.allHosts.List() {
		if err := hostRoute.Route(host, o.allProxies); err != nil {
			return err
		}
	}
	log.WithFields(logger.Fields{
		"ProxyDown":       hostRoute.stats.ProxyDown,
		"NoEligibleProxy": hostRoute.stats.NoEligibleProxy,
		"DoNothing":       hostRoute.stats.DoNothing,
		"Balanced":        hostRoute.stats.Balanced,
		"TagChange":       hostRoute.stats.TagChange,
	}).Debug("intermediate statistics")
	log.Debug("starting prefer sticky")
	err := hostRoute.PreferSticky(o.allProxies)
	log.WithFields(logger.Fields{
		"ProxyDown":       hostRoute.stats.ProxyDown,
		"NoEligibleProxy": hostRoute.stats.NoEligibleProxy,
		"DoNothing":       hostRoute.stats.DoNothing,
		"Balanced":        hostRoute.stats.Balanced,
		"TagChange":       hostRoute.stats.TagChange,
	}).Debug("final statistics")
	return err
}

func (o *Operation) Drain(ctx context.Context, z api.ZabbixAPI) error {
	log := logger.WithField("module", "operation.drain")
	log.Debug("draining proxy to other proxy")
	for _, proxy := range o.allProxies.List() {
		logProxy := log.WithField("proxy_id", proxy.ApiProxy.ProxyID)
		logProxy.Debug("checking if there's any to drain")
		if proxy.Hosts.MD5() != proxy.PlannedHost.MD5() {
			logProxy.Debug("going to update hosts")
			var plannedHost []api.Host
			for _, host := range proxy.PlannedHost.List() {
				plannedHost = append(plannedHost, *host)
			}
			ctxTimeout, cancel := ctxRequestTimeout(ctx)
			if err := proxy.ApiProxy.UpdateHosts(ctxTimeout, z, plannedHost); err != nil {
				logProxy.WithError(err).Warnln("failed updating hosts")
				cancel()
				return err
			}
			cancel()
			logProxy.WithFields(logger.Fields{
				"previous_total_hosts": proxy.Hosts.Len(),
				"total_hosts":          proxy.PlannedHost.Len(),
			}).
				Info("updated hosts")
		} else {
			logProxy.Debug("no hosts update")
		}
	}
	return nil
}
