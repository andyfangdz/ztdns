package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

var ztToken = os.Getenv("ZT_TOKEN")
var ztNetwork = os.Getenv("ZT_NETWORK")

type MemberConfig struct {
	IpAssignments []string
}

type Member struct {
	Id     string
	Name   string
	Config MemberConfig
}

type MemberResponse []Member

type handler struct{}

var ztCache = cache.New(1*time.Minute, 2*time.Minute)

func getZTDomainAddresses() map[string]string {
	ret := make(map[string]string)
	var ztResponse MemberResponse
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://my.zerotier.com/api/network/%s/member", ztNetwork), nil)
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", ztToken))
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &ztResponse)
	for _, member := range ztResponse {
		ret[member.Name] = member.Config.IpAssignments[0]
	}
	return ret
}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name

		var addressBookInterface interface{}
		var found bool

		addressBookInterface, found = ztCache.Get("query")
		if !found {
			addressBookInterface = getZTDomainAddresses()
			ztCache.Set("query", addressBookInterface, cache.DefaultExpiration)
		}

		if addressBook, ok := addressBookInterface.(map[string]string); ok {
			if address, ok := addressBook[domain]; ok {
				msg.Answer = append(msg.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
					A:   net.ParseIP(address),
				})
			}
		}

	}
	w.WriteMsg(&msg)
}

func main() {
	srv := &dns.Server{Addr: ":" + strconv.Itoa(53), Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
