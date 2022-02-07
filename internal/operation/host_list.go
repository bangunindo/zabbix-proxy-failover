package operation

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/bangunindo/zabbix-proxy-failover/pkg/api"
	"sort"
	"strconv"
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
