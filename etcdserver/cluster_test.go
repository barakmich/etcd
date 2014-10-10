package etcdserver

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/coreos/etcd/pkg/types"
)

func urls(strs []string) types.URLs {
	out, err := types.NewURLs(strs)
	if err != nil {
		fmt.Println("Error!", err)
	}
	return out
}

func TestClusterAddSlice(t *testing.T) {
	tests := []struct {
		mems []Member

		want *Cluster
	}{
		{
			[]Member{},

			&Cluster{},
		},
		{
			[]Member{
				{ID: 1, PeerURLs: urls([]string{"http://foo:1", "http://bar:1"})},
				{ID: 2, PeerURLs: urls([]string{"http://baz:1"})},
			},

			&Cluster{
				1: &Member{
					ID:       1,
					PeerURLs: urls([]string{"http://foo:1", "http://bar:1"}),
				},
				2: &Member{
					ID:       2,
					PeerURLs: urls([]string{"http://baz:1"}),
				},
			},
		},
	}
	for i, tt := range tests {
		c := &Cluster{}
		if err := c.AddSlice(tt.mems); err != nil {
			t.Errorf("#%d: err=%#v, want nil", i, err)
			continue
		}
		if !reflect.DeepEqual(c, tt.want) {
			t.Errorf("#%d: c=%#v, want %#v", i, c, tt.want)
		}
	}
}

func TestClusterAddSliceBad(t *testing.T) {
	c := Cluster{
		1: &Member{ID: 1},
	}
	if err := c.AddSlice([]Member{{ID: 1}}); err == nil {
		t.Error("want err, but got nil")
	}
}

func TestClusterPick(t *testing.T) {
	cs := Cluster{
		1: &Member{ID: 1, PeerURLs: urls([]string{"http://abc:1", "http://def:1", "http://ghi:1", "http://jkl:1", "http://mno:1", "http://pqr:1", "http://stu:1"})},
		2: &Member{ID: 2, PeerURLs: urls([]string{"http://xyz:1"})},
		3: &Member{ID: 3, PeerURLs: types.URLs{}},
	}
	ids := map[string]bool{
		"http://abc:1": true,
		"http://def:1": true,
		"http://ghi:1": true,
		"http://jkl:1": true,
		"http://mno:1": true,
		"http://pqr:1": true,
		"http://stu:1": true,
	}
	for i := 0; i < 1000; i++ {
		a := cs.Pick(1)
		if !ids[a.String()] {
			t.Errorf("returned ID %q not in expected range!", a)
			break
		}
	}
	if b := cs.Pick(2); b.String() != "http://xyz:1" {
		t.Errorf("id=%q, want %q", b, "http://xyz:1")
	}
	if c := cs.Pick(3); c != nil {
		t.Errorf("id=%q, want %v", c, nil)
	}
	if d := cs.Pick(4); d != nil {
		t.Errorf("id=%q, want %v", d, nil)
	}
}

func TestClusterFind(t *testing.T) {
	tests := []struct {
		id    uint64
		name  string
		mems  []Member
		match bool
	}{
		{
			1,
			"node1",
			[]Member{{Name: "node1", ID: 1}},
			true,
		},
		{
			2,
			"foobar",
			[]Member{},
			false,
		},
		{
			2,
			"node2",
			[]Member{{Name: "node1", ID: 1}, {Name: "node2", ID: 2}},
			true,
		},
		{
			3,
			"node3",
			[]Member{{Name: "node1", ID: 1}, {Name: "node2", ID: 2}},
			false,
		},
	}

	for i, tt := range tests {
		c := Cluster{}
		c.AddSlice(tt.mems)

		m := c.FindID(tt.id)
		if m == nil && !tt.match {
			continue
		}
		if m == nil && tt.match {
			t.Errorf("#%d: expected match got empty", i)
		}
		if m.ID != tt.id && tt.match {
			t.Errorf("#%d: got = %v, want %v", i, m.Name, tt.id)
		}
	}
}

func TestClusterSet(t *testing.T) {
	tests := []struct {
		f    string
		mems []Member
	}{
		{
			"mem1=http://10.0.0.1:2379,mem1=http://128.193.4.20:2379,mem2=http://10.0.0.2:2379,default=http://127.0.0.1:2379",
			[]Member{
				{ID: 3736794188555456841, Name: "mem1", PeerURLs: urls([]string{"http://10.0.0.1:2379", "http://128.193.4.20:2379"})},
				{ID: 5674507346857578431, Name: "mem2", PeerURLs: urls([]string{"http://10.0.0.2:2379"})},
				{ID: 2676999861503984872, Name: "default", PeerURLs: urls([]string{"http://127.0.0.1:2379"})},
			},
		},
	}
	for i, tt := range tests {
		c := Cluster{}
		if err := c.AddSlice(tt.mems); err != nil {
			t.Error(err)
		}

		g := Cluster{}
		g.Set(tt.f)

		if g.String() != c.String() {
			t.Errorf("#%d: set = %v, want %v", i, g, c)
		}
	}
}

func TestClusterSetBad(t *testing.T) {
	tests := []string{
		// invalid URL
		"%^",
		// no URL defined for member
		"mem1=,mem2=http://128.193.4.20:2379,mem3=http://10.0.0.2:2379",
		"mem1,mem2=http://128.193.4.20:2379,mem3=http://10.0.0.2:2379",
		// TODO(philips): anyone know of a 64 bit sha1 hash collision
		// "06b2f82fd81b2c20=http://128.193.4.20:2379,02c60cb75083ceef=http://128.193.4.20:2379",
	}
	for i, tt := range tests {
		g := Cluster{}
		if err := g.Set(tt); err == nil {
			t.Errorf("#%d: set = %v, want err", i, tt)
		}
	}
}

func TestClusterIDs(t *testing.T) {
	cs := Cluster{}
	cs.AddSlice([]Member{
		{ID: 1},
		{ID: 4},
		{ID: 100},
	})
	w := []uint64{1, 4, 100}
	g := cs.IDs()
	if !reflect.DeepEqual(w, g) {
		t.Errorf("IDs=%+v, want %+v", g, w)
	}
}

func TestClusterAddBad(t *testing.T) {
	// Should not be possible to add the same ID multiple times
	mems := []Member{
		{ID: 1, Name: "mem1"},
		{ID: 1, Name: "mem2"},
	}
	c := &Cluster{}
	c.Add(Member{ID: 1, Name: "mem1"})
	for i, m := range mems {
		if err := c.Add(m); err == nil {
			t.Errorf("#%d: set = %v, want err", i, m)
		}
	}
}

func TestClusterPeerURLs(t *testing.T) {
	tests := []struct {
		mems  []Member
		wurls []string
	}{
		// single peer with a single address
		{
			mems: []Member{
				{ID: 1, PeerURLs: urls([]string{"http://192.0.2.1:8001"})},
			},
			wurls: []string{"http://192.0.2.1:8001"},
		},

		// several members explicitly unsorted
		{
			mems: []Member{
				{ID: 2, PeerURLs: urls([]string{"http://192.0.2.3:80", "http://192.0.2.4:80"})},
				{ID: 3, PeerURLs: urls([]string{"http://192.0.2.5:80", "http://192.0.2.6:80"})},
				{ID: 1, PeerURLs: urls([]string{"http://192.0.2.1:80", "http://192.0.2.2:80"})},
			},
			wurls: []string{"http://192.0.2.1:80", "http://192.0.2.2:80", "http://192.0.2.3:80", "http://192.0.2.4:80", "http://192.0.2.5:80", "http://192.0.2.6:80"},
		},

		// no members
		{
			mems:  []Member{},
			wurls: []string{},
		},

		// peer with no peer urls
		{
			mems: []Member{
				{ID: 3, PeerURLs: types.URLs{}},
			},
			wurls: []string{},
		},
	}

	for i, tt := range tests {
		c := Cluster{}
		if err := c.AddSlice(tt.mems); err != nil {
			t.Errorf("AddSlice error: %v", err)
			continue
		}
		urls := c.PeerURLs()
		if !reflect.DeepEqual(urls, tt.wurls) {
			t.Errorf("#%d: PeerURLs = %v, want %v", i, urls, tt.wurls)
		}
	}
}

func TestClusterClientURLs(t *testing.T) {
	tests := []struct {
		mems  []Member
		wurls []string
	}{
		// single peer with a single address
		{
			mems: []Member{
				{ID: 1, ClientURLs: []string{"http://192.0.2.1"}},
			},
			wurls: []string{"http://192.0.2.1"},
		},

		// single peer with a single address with a port
		{
			mems: []Member{
				{ID: 1, ClientURLs: []string{"http://192.0.2.1:8001"}},
			},
			wurls: []string{"http://192.0.2.1:8001"},
		},

		// several members explicitly unsorted
		{
			mems: []Member{
				{ID: 2, ClientURLs: []string{"http://192.0.2.3", "http://192.0.2.4"}},
				{ID: 3, ClientURLs: []string{"http://192.0.2.5", "http://192.0.2.6"}},
				{ID: 1, ClientURLs: []string{"http://192.0.2.1", "http://192.0.2.2"}},
			},
			wurls: []string{"http://192.0.2.1", "http://192.0.2.2", "http://192.0.2.3", "http://192.0.2.4", "http://192.0.2.5", "http://192.0.2.6"},
		},

		// no members
		{
			mems:  []Member{},
			wurls: []string{},
		},

		// peer with no client urls
		{
			mems: []Member{
				{ID: 3, ClientURLs: []string{}},
			},
			wurls: []string{},
		},
	}

	for i, tt := range tests {
		c := Cluster{}
		if err := c.AddSlice(tt.mems); err != nil {
			t.Errorf("AddSlice error: %v", err)
			continue
		}
		urls := c.ClientURLs()
		if !reflect.DeepEqual(urls, tt.wurls) {
			t.Errorf("#%d: ClientURLs = %v, want %v", i, urls, tt.wurls)
		}
	}
}
