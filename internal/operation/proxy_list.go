package operation

import (
	"github.com/bangunindo/zabbix-proxy-failover/pkg/api"
	"sort"
	"strconv"
	"strings"
)

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
