package streamserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/oschwald/geoip2-golang"

	"github.com/gorilla/mux"
)

type StreamServer struct {
	running            bool
	updateTimer        *time.Timer
	config             *ServerConfig
	api                *APIHandler
	epg                *EPGHandler
	channels           *PlaylistHandler
	stopStreamLoadChan chan bool
	restartChan        chan bool
	router             *mux.Router
	geoipDb            *geoip2.Reader
	geoipWhitelist     map[string]bool
	geoIPCidrWhitelist []*net.IPNet
}

// Initialize the server
func NewStreamServer(configPath string) *StreamServer {
	s := StreamServer{
		stopStreamLoadChan: make(chan bool),
		restartChan:        make(chan bool),
		config:             NewServerConfig(configPath),
		router:             mux.NewRouter(),
	}

	return &s
}

func (s *StreamServer) Run() {

	setupLogging(s.config.data.LogFile)

	s.api = NewAPIHandler(s.config, &s.restartChan)
	s.api.RegisterRoutes(s.router)

	s.channels = NewPlaylistHandler(s.config)
	s.channels.RegisterRoutes(s.router)

	s.epg = NewEPGHandler(s.config)
	s.epg.RegisterRoutes(s.router)

	registerPlayerRoutes(s.router)

	s.router.HandleFunc("/health", s.healthCheckRequest)

	for {
		log.Printf("Starting M3U Proxy Server\n")

		log.Printf("Starting stream server\n")
		log.Printf("Playlist: %s\n", s.config.data.Playlist)
		log.Printf("EPG: %s\n", s.config.data.Epg)

		err := auth.InitializeAuth(s.config.data.Auth)
		if err != nil {
			log.Printf("Failed to initialize authentication: %s\n", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.updateTimer = time.NewTimer(time.Duration(s.config.data.ScanTime) * time.Second)
		s.running = true
		go func() {
			s.channels.Load(ctx)
			for {
				select {
				case <-s.stopStreamLoadChan:
					log.Println("Stopping stream server")
					return
				case <-s.updateTimer.C:
					s.channels.Load(ctx)
					if s.running {
						s.updateTimer.Reset(time.Duration(s.config.data.ScanTime) * time.Second)
					}
				}
				if !s.running {
					break
				}
			}
		}()

		if s.configureSecurity() != nil {
			log.Println("GeoIP database not found, geo-location will not be available.")
		}

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", s.config.data.Port),
			Handler: s.secure(s.router),
		}

		// Channel to listen for termination signal (SIGINT, SIGTERM)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			log.Printf("Server listening on %s.\n", server.Addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Println("Server failed:", err)
			}
		}()

		quitServer := false
		select {
		case <-sigChan:
			log.Println("Signal received, shutting down server...")
			quitServer = true
		case <-s.restartChan:
			quitServer = false
		}

		s.updateTimer.Stop()
		s.running = false
		s.stopStreamLoadChan <- true
		log.Printf("Stream server stopped\n")

		s.cleanGeoIp()

		if err := server.Shutdown(ctx); err != nil {
			log.Println("Server forced to shutdown:", err)
		}

		if quitServer {
			log.Println("Server shutdown.")
			break
		}
	}
}

func (s *StreamServer) Restart() {
	go func() {
		log.Println("Server restart in 3 seconds")
		time.Sleep(3 * time.Second)
		s.restartChan <- true
	}()
}

func (s *StreamServer) healthCheckRequest(w http.ResponseWriter, r *http.Request) {
	if s.channels == nil {
		http.Error(w, "Streams not loaded", http.StatusServiceUnavailable)
		return
	}
	// health := map[string]interface{}{
	//     "status":        "ok",
	//     "activeStreams": activeStreams,
	//     "uptime":        time.Since(startTime).String(),
	// }

	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(health)

	w.WriteHeader(http.StatusOK)
}

func (s *StreamServer) configureSecurity() error {

	var err error

	if s.config.data.Security.GeoIP.Database == "" {
		return nil
	}

	s.geoipDb, err = geoip2.Open(s.config.data.Security.GeoIP.Database)
	if err != nil {
		s.geoipDb = nil
		return err
	}

	s.geoipWhitelist = make(map[string]bool)
	for _, country := range s.config.data.Security.GeoIP.Whitelist {
		s.geoipWhitelist[country] = true
	}

	s.geoIPCidrWhitelist = make([]*net.IPNet, 0)

	for _, cidr := range s.config.data.Security.GeoIP.InternalNetworks {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		s.geoIPCidrWhitelist = append(s.geoIPCidrWhitelist, ipnet)
	}

	return nil
}

func (s *StreamServer) cleanGeoIp() {
	if s.geoipDb != nil {
		s.geoipDb.Close()
	}
}

func (s *StreamServer) secure(next http.Handler) http.Handler {

	if len(s.config.data.Security.AllowedCORSDomains) > 0 {
		next = s.cors(next)
	}

	if s.geoipDb == nil {
		return next
	}

	log.Println("GeoIP enabled")

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

		for _, ipnet := range s.geoIPCidrWhitelist {
			if ipnet.Contains(parsedIP) {
				next.ServeHTTP(w, r)
				return
			}
		}

		record, err := s.geoipDb.Country(parsedIP)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		countryCode := record.Country.IsoCode
		if _, ok := s.geoipWhitelist[countryCode]; !ok {
			log.Printf("Access Denied: %s, Country: %s\n", ip, countryCode)
			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *StreamServer) cors(next http.Handler) http.Handler {
	log.Println("CORS enabled")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", strings.Join(s.config.data.Security.AllowedCORSDomains, ","))
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
