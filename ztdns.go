package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
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

func getZTDomainAddresses() (map[string]string, error) {
	ret := make(map[string]string)
	var ztResponse MemberResponse
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://my.zerotier.com/api/network/%s/member", ztNetwork), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", ztToken))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(body, &ztResponse)
	for _, member := range ztResponse {
		if len(member.Config.IpAssignments) > 0 {
			ret[member.Name] = member.Config.IpAssignments[0]
		}
	}
	return ret, nil
}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		glog.Infof("Received request for: %s", domain)

		var addressBookInterface interface{}
		var found bool
		var err error

		addressBookInterface, found = ztCache.Get("query")
		if !found {
			addressBookInterface, err = getZTDomainAddresses()
			if err != nil {
				glog.Error(err)
				msg.SetRcode(r, dns.RcodeServerFailure)
				w.WriteMsg(&msg)
				return
			} else {
				ztCache.Set("query", addressBookInterface, cache.DefaultExpiration)
			}
		}

		if addressBook, ok := addressBookInterface.(map[string]string); ok {
			if address, ok := addressBook[domain]; ok {
				msg.Answer = append(msg.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
					A:   net.ParseIP(address),
				})
			} else {
				glog.Info("not found in zerotier registry, returning NXDOMAIN")
				msg.SetRcode(r, dns.RcodeNameError)
				w.WriteMsg(&msg)
				return
			}
		} else {
			glog.Error("cast error, returning ServFail")
			msg.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(&msg)
			return
		}
	default:
		glog.Info("non-A request, returning NotImp")
		msg.SetRcode(r, dns.RcodeNotImplemented)
		w.WriteMsg(&msg)
	}
	w.WriteMsg(&msg)
}

func main() {
	flag.Parse()
	srv := &dns.Server{Addr: ":" + strconv.Itoa(53), Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
