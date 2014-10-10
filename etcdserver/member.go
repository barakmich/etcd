package etcdserver

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/pkg/flags"
	"github.com/coreos/etcd/pkg/types"
)

const machineKVPrefix = "/_etcd/machines/"

type Member struct {
	ID uint64
	// Name is purely advisory. May be empty.
	Name string

	PeerURLs   types.URLs
	ClientURLs []string
}

// NewMember creates a Member without an ID and generates one based on the
// name, peer URLs. This is used for bootstrapping.
func newMember(name string, peerURLs types.URLs, now *time.Time) *Member {
	m := &Member{Name: name, PeerURLs: peerURLs}

	var b []byte
	peerURLstrings := m.PeerURLs.StringSlice()
	sort.Strings(peerURLstrings)
	for _, p := range peerURLstrings {
		b = append(b, []byte(p)...)
	}

	if now != nil {
		b = append(b, []byte(fmt.Sprintf("%d", now.Unix()))...)
	}

	hash := sha1.Sum(b)
	m.ID = binary.BigEndian.Uint64(hash[:8])
	return m
}

func NewMember(name string, peers []string) *Member {
	return newMember(name, types.URLs(*flags.NewURLsValue(strings.Join(peers, ","))), nil)
}

func NewMemberFromURLs(name string, peers types.URLs) *Member {
	return newMember(name, peers, nil)
}

func (m Member) storeKey() string {
	return path.Join(machineKVPrefix, strconv.FormatUint(m.ID, 16))
}
