// Package torblock contains a Traefik plugin for blocking requests from the Tor network
package torblock

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"text/template"
	"time"
)

var ipRegex = regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`)

// Config for the plugin configuration.
type Config struct {
	Enabled               bool
	AddressListURL        string
	UpdateIntervalSeconds int
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Enabled:               true,
		AddressListURL:        "https://check.torproject.org/exit-addresses",
		UpdateIntervalSeconds: 3600,
	}
}

// TorBlock plugin struct.
type TorBlock struct {
	next           http.Handler
	name           string
	template       *template.Template
	enabled        bool
	addressListURL string
	updateInterval time.Duration
	blockedIPs     *IPv4Set
	client         *http.Client
}

// New creates a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	_, err := url.ParseRequestURI(config.AddressListURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exit-addresses url")
	}
	if config.UpdateIntervalSeconds < 60 {
		return nil, fmt.Errorf("update interval cannot be lower than 60 seconds")
	}

	a := &TorBlock{
		next:           next,
		name:           name,
		template:       template.New("torblock").Delims("[[", "]]"),
		enabled:        config.Enabled,
		addressListURL: config.AddressListURL,
		updateInterval: time.Duration(config.UpdateIntervalSeconds) * time.Second,
		blockedIPs:     CreateIPv4Set(),
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
	a.UpdateBlockedIPs()
	go a.UpdateWorker()

	return a, nil
}

// ServeHTTP handles all requests flowing through the plugin.
func (a *TorBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check if enabled
	if !a.enabled {
		a.next.ServeHTTP(rw, req)
		return
	}

	// Extract IP from remote address
	remoteHost, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		log.Printf("torblock: failed to extract ip from remote address: %v", err)
		a.next.ServeHTTP(rw, req)
		return
	}

	// Parse the IP, and skip filtering if not a valid IPv4 address
	remoteIP, err := ParseIPv4(remoteHost)
	if err != nil {
		a.next.ServeHTTP(rw, req)
		return
	}

	// Check if the IP is blocked and cancel request if needed
	if a.blockedIPs.Contains(remoteIP) {
		log.Printf("torblock: request denied (%s)", remoteHost)
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	// Continue
	a.next.ServeHTTP(rw, req)
}

// UpdateWorker regularly updates the list of blocked IPs in a configurable update interval.
func (a *TorBlock) UpdateWorker() {
	for range time.Tick(a.updateInterval) {
		a.UpdateBlockedIPs()
	}
}

// UpdateBlockedIPs updates the list of blocked IPs via the addressListURL.
func (a *TorBlock) UpdateBlockedIPs() {
	res, err := a.client.Get(a.addressListURL)
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
	newSet := CreateIPv4Set()
	for _, ipStr := range foundIPStrs {
		ip, err := ParseIPv4(ipStr)
		if err == nil {
			newSet.Add(ip)
		}
	}
	a.blockedIPs = newSet
	log.Printf("torblock: updated blocked ip list (found %d ips)", len(foundIPStrs))
}
