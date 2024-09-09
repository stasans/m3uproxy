package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/oschwald/geoip2-golang"
)

type GeoIPConfig struct {
	Database         string   `json:"database"`
	Whitelist        []string `json:"whitelist,omitempty"`
	InternalNetworks []string `json:"internal_networks,omitempty"`
}

type SecurityConfig struct {
	GeoIP       GeoIPConfig `json:"geoip"`
	DisableCors bool        `json:"disable_cors"`
	CorsOrigins []string    `json:"cors_origins,omitempty"`
}

var securityConfig *SecurityConfig

var geoipDb *geoip2.Reader
var geoipWhitelist map[string]bool
var geoIPCidrWhitelist []*net.IPNet

func configureSecurity(data json.RawMessage) error {

	securityConfig = &SecurityConfig{}
	err := json.Unmarshal(data, securityConfig)
	if err != nil {
		return nil
	}

	return nil
}

func configureGeoIp() error {

	if securityConfig == nil {
		return errors.New("missing configuration")
	}

	var err error

	geoipDb, err = geoip2.Open(securityConfig.GeoIP.Database)
	if err != nil {
		geoipDb = nil
		return err
	}

	geoipAllowedCountries := make(map[string]bool)
	for _, country := range securityConfig.GeoIP.Whitelist {
		geoipAllowedCountries[country] = true
	}

	geoIPCidrWhitelist = make([]*net.IPNet, 0)

	for _, cidr := range securityConfig.GeoIP.InternalNetworks {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		geoIPCidrWhitelist = append(geoIPCidrWhitelist, ipnet)
	}

	return nil
}

func cleanGeoIp() {
	if geoipDb != nil {
		geoipDb.Close()
	}
}

func geoip(next http.Handler) http.Handler {

	if securityConfig == nil || geoipDb == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		parsedIP := net.ParseIP(ip)

		for _, ipnet := range geoIPCidrWhitelist {
			if ipnet.Contains(parsedIP) {
				next.ServeHTTP(w, r)
				return
			}
		}

		record, err := geoipDb.Country(parsedIP)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		countryCode := record.Country.IsoCode
		if _, ok := geoipWhitelist[countryCode]; !ok {
			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
