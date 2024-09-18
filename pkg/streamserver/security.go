package streamserver

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

var geoipDb *geoip2.Reader
var geoipWhitelist map[string]bool
var geoIPCidrWhitelist []*net.IPNet

func configureGeoIp() error {

	var err error

	if Config.Security.GeoIP.Database == "" {
		return nil
	}

	geoipDb, err = geoip2.Open(Config.Security.GeoIP.Database)
	if err != nil {
		geoipDb = nil
		return err
	}

	geoipWhitelist = make(map[string]bool)
	for _, country := range Config.Security.GeoIP.Whitelist {
		geoipWhitelist[country] = true
	}

	geoIPCidrWhitelist = make([]*net.IPNet, 0)

	for _, cidr := range Config.Security.GeoIP.InternalNetworks {
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

	if geoipDb == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ip := ""
		if r.Header.Get("X-Real-IP") != "" {
			ip = r.Header.Get("X-Real-IP")
		} else if r.Header.Get("X-Forwarded-For") != "" {
			ips := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
			ip = ips[0]
		} else {
			var err error
			ip, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if ip == "" {
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
			log.Printf("Access Denied: %s, Country: %s\n", ip, countryCode)
			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
