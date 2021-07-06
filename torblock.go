package torblock

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"
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
	blockedIPs     []net.IP
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
		blockedIPs:     make([]net.IP, 0),
	}
	a.UpdateBlockedIPs()
	go a.UpdateWorker()

	return a, nil
}

func (a *TorBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	requestIPs := a.GetRemoteIPs(req)

	for _, deniedIP := range a.blockedIPs {
		for _, requestIP := range requestIPs {
			if bytes.Compare(deniedIP, requestIP) == 0 {
				log.Printf("torblock: request denied (%s)", requestIP)
				rw.WriteHeader(http.StatusForbidden)
				return
			}
		}
	}

	a.next.ServeHTTP(rw, req)
}

// GetRemoteIP returns a list of IPs that are associated with this request.
func (a *TorBlock) GetRemoteIPs(req *http.Request) []net.IP {
	var ips []net.IP

	xff := req.Header.Get("X-Forwarded-For")
	xffs := strings.Split(xff, ",")
	for _, address := range xffs {
		trimmed := strings.TrimSpace(address)
		ip := net.ParseIP(trimmed)
		if ip != nil {
			ips = append(ips, ip)
		}
	}

	address, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		address = req.RemoteAddr
	}
	trimmed := strings.TrimSpace(address)
	ip := net.ParseIP(trimmed)
	if ip != nil {
		ips = append(ips, ip)
	}

	return ips
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
	foundIPs := make([]net.IP, 0, len(foundIPStrs))
	for _, ipStr := range foundIPStrs {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			foundIPs = append(foundIPs, ip)
		}
	}
	a.blockedIPs = foundIPs

	log.Printf("torblock: updated blocked ip list (found %d ips)", len(foundIPs))
}
