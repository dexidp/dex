package ldap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

func BenchmarkGroupLookup(b *testing.B) {
	host := os.Getenv("DEX_LDAP_HOST")
	if host == "" {
		b.Skipf(`test environment variable "DEX_LDAP_HOST" not set, skipping`)
	}
	host = fmt.Sprintf("%s:%s", host, getenv("DEX_LDAP_PORT", "389"))

	for _, size := range []int{8, 128, 2048, 8192} {
		b.Run(fmt.Sprintf("groups=%d", size), func(b *testing.B) {
			baseDN, user := benchmarkGroupDirectory(b, host, size)
			for _, mode := range []struct {
				name     string
				matchers []UserMatcher
			}{
				{
					name:     "nonrecursive",
					matchers: []UserMatcher{{UserAttr: "DN", GroupAttr: "member"}},
				},
				{
					name:     "recursive",
					matchers: []UserMatcher{{UserAttr: "DN", GroupAttr: "member", RecursionGroupAttr: "member"}},
				},
				{
					name: "overlapping_matchers",
					matchers: []UserMatcher{
						{UserAttr: "DN", GroupAttr: "member"},
						{UserAttr: "DN", GroupAttr: "member", RecursionGroupAttr: "member"},
					},
				},
			} {
				b.Run(mode.name, func(b *testing.B) {
					c := &Config{Host: host, InsecureNoSSL: true}
					c.BindDN = "cn=admin,dc=example,dc=org"
					c.BindPW = "admin"
					c.UserSearch.BaseDN = baseDN
					c.UserSearch.Username = UsernameAttributes{"cn"}
					c.GroupSearch.BaseDN = baseDN
					c.GroupSearch.NameAttr = "cn"
					c.GroupSearch.UserMatchers = mode.matchers
					conn, err := c.openConnector(slog.New(slog.DiscardHandler))
					if err != nil {
						b.Fatal(err)
					}
					groups, err := conn.groups(context.Background(), user)
					if err != nil || len(groups) != size {
						b.Fatalf("groups: got %d names, want %d; error: %v", len(groups), size, err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						if _, err := conn.groups(context.Background(), user); err != nil {
							b.Fatal(err)
						}
					}
					b.ReportMetric(float64(size), "groups/op")
				})
			}
		})
	}
}

// benchmarkGroupDirectory creates flat groups so recursive lookups have no
// parents. The overlapping matcher case returns the same groups twice but
// makes no parent queries, isolating duplicate lookup from per-group LDAP I/O.
func benchmarkGroupDirectory(b *testing.B, host string, size int) (string, ldap.Entry) {
	b.Helper()
	conn, err := ldap.DialURL("ldap://" + host)
	if err != nil {
		b.Fatal(err)
	}
	if err := conn.Bind("cn=admin,dc=example,dc=org", "admin"); err != nil {
		conn.Close()
		b.Fatal(err)
	}
	created := make([]string, 0, 1+size)
	b.Cleanup(func() {
		for i := len(created) - 1; i >= 0; i-- {
			if err := conn.Del(ldap.NewDelRequest(created[i], nil)); err != nil {
				b.Errorf("remove benchmark entry: %v", err)
			}
		}
		conn.Close()
	})
	ou := fmt.Sprintf("dex-group-benchmark-%d-%d", size, time.Now().UnixNano())
	baseDN := "ou=" + ou + ",dc=example,dc=org"
	add := ldap.NewAddRequest(baseDN, nil)
	add.Attribute("objectClass", []string{"organizationalUnit"})
	add.Attribute("ou", []string{ou})
	if err := conn.Add(add); err != nil {
		b.Fatal(err)
	}
	created = append(created, baseDN)
	user := ldap.Entry{DN: "cn=user," + baseDN}
	for i := range size {
		name := fmt.Sprintf("group%05d", i)
		dn := "cn=" + name + "," + baseDN
		add := ldap.NewAddRequest(dn, nil)
		add.Attribute("objectClass", []string{"groupOfNames"})
		add.Attribute("cn", []string{name})
		add.Attribute("member", []string{user.DN})
		if err := conn.Add(add); err != nil {
			b.Fatal(err)
		}
		created = append(created, dn)
	}
	return baseDN, user
}
