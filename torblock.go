package torblock

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"text/template"
	"time"

	"github.com/jpxd/torblock/netaddr"
)

var ipRegex = regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`)

// Config the plugin configuration.
type Config struct {
	AddressListURL string
	UpdateInterval int32
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AddressListURL: "https://check.torproject.org/exit-addresses",
		UpdateInterval: 3600,
	}
}

// TorBlock plugin struct.
type TorBlock struct {
	next           http.Handler
	name           string
	template       *template.Template
	addressListURL string
	updateInterval time.Duration
	blockedIPs     *netaddr.IPSet
}

// New creates a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	_, err := url.ParseRequestURI(config.AddressListURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exit-addresses url")
	}
	if config.UpdateInterval < 60 {
		return nil, fmt.Errorf("update interval cannot be lower than 60 seconds")
	}

	a := &TorBlock{
		next:           next,
		name:           name,
		template:       template.New("torblock").Delims("[[", "]]"),
		addressListURL: config.AddressListURL,
		updateInterval: time.Duration(config.UpdateInterval) * time.Second,
		blockedIPs:     &netaddr.IPSet{},
	}
	a.UpdateBlockedIPs()
	go a.UpdateWorker()

	return a, nil
}

func (a *TorBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	remoteAddr, err := netaddr.ParseIPPort(req.RemoteAddr)
	if err != nil {
		log.Printf("torblock: bad request remote address")
		return
	}

	if a.blockedIPs.Contains(remoteAddr.IP()) {
		log.Printf("torblock: request denied (%s)", remoteAddr.IP())
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	a.next.ServeHTTP(rw, req)
}

func (a *TorBlock) UpdateWorker() {
	for range time.Tick(a.updateInterval) {
		a.UpdateBlockedIPs()
	}
}

func (a *TorBlock) UpdateBlockedIPs() {
	log.Printf("torblock: updating blocked ip list")

	res, err := http.DefaultClient.Get(a.addressListURL)
	if err != nil {
		log.Printf("torblock: failed to update address list: %s", err)
		return
	}
	if res.StatusCode != 200 {
		log.Printf("torblock: failed to update address list: status code is %d", res.StatusCode)
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("torblock: failed to read address list body: %s", err)
		return
	}
	bodyStr := string(body)

	foundIPStrs := ipRegex.FindAllString(bodyStr, -1)
	builder := netaddr.IPSetBuilder{}
	for _, ipStr := range foundIPStrs {
		ip, err := netaddr.ParseIP(ipStr)
		if err == nil {
			builder.Add(ip)
		}
	}
	blockedIPs, err := builder.IPSet()
	if err != nil {
		log.Printf("torblock: failed to build blocked ip set: %s", err)
		return
	}
	a.blockedIPs = blockedIPs
	log.Printf("torblock: updated blocked ip list (found %d ips)", len(foundIPStrs))
}
