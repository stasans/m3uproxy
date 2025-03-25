package streamserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/gorilla/mux"
)

type streamLicense struct {
	keyType string
	keyId   string
	key     string
}

type streamLicenseType struct {
	keys map[string]streamLicense
}

type streamLicenseManager struct {
	licenses map[string]streamLicenseType
}

type emeLicenseRequest struct {
	KeyIds []string `json:"kids"`
}

type emeLicenseKey struct {
	Type string `json:"kty"`
	Key  string `json:"k"`
	Id   string `json:"kid"`
}

type emeLicenseResponse struct {
	Keys []emeLicenseKey `json:"keys"`
}

func newStreamLicenseManager() *streamLicenseManager {
	return &streamLicenseManager{
		licenses: make(map[string]streamLicenseType),
	}
}

func (m *streamLicenseManager) addLicense(keyType, keyId, key string) {

	if _, ok := m.licenses[keyType]; !ok {
		m.licenses[keyType] = streamLicenseType{
			keys: make(map[string]streamLicense),
		}
	}

	m.licenses[keyType].keys[keyId] = streamLicense{
		keyType: keyType,
		keyId:   keyId,
		key:     key,
	}
}

func (m *streamLicenseManager) getLicense(keyType string, keyId string) (streamLicense, bool) {

	if _, ok := m.licenses[keyType]; !ok {
		return streamLicense{}, false
	}

	license, ok := m.licenses[keyType].keys[keyId]
	return license, ok
}

// process a  License Release Format based on EME JSON-base spec
func licenseManagerRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	authParts := strings.SplitN(authHeader, " ", 2)
	token := authParts[1]

	ok := auth.VerifyToken(token)
	if !ok {
		http.Error(w, "Forbidden", http.StatusUnauthorized)
		log.Printf("Unauthorized access to stream stream %s: Token expired, missing, or invalid.\n", r.URL.Path)
		return
	}

	vars := mux.Vars(r)
	licenseType := vars["type"]

	// Check if the license manager is initialized
	if licenseManger == nil {
		http.Error(w, "No licenses found", http.StatusNotFound)
		return
	}

	switch licenseType {
	case "clearkey":

		var licenses emeLicenseResponse
		for _, license := range licenseManger.licenses["clearkey"].keys {

			licenses.Keys = append(licenses.Keys, emeLicenseKey{
				Type: "oct",
				Key:  license.key,
				Id:   license.keyId,
			})
		}

		// Return the license
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(licenses)
	case "widevine":
	case "playready":
	default:
		http.Error(w, "Invalid license type", http.StatusBadRequest)
		return
	}

}

func registerLicenseRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/drm/{type:.*}", basicAuth(licenseManagerRequest))
	return r
}
