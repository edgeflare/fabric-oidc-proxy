package proxy

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/edgeflare/fabric-oidc-proxy/internal/fabric"
	"github.com/edgeflare/fabric-oidc-proxy/internal/util"
	"github.com/edgeflare/pgo"
	"github.com/hyperledger/fabric-ca/api"
)

// enrollUserHandler is a http.Handler that registers and enrolls a user with the Fabric CA.
func enrollUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := pgo.OIDCUser(r)
	if !ok || user.Active == false {
		http.Error(w, "no user found", http.StatusUnauthorized)
		return
	}

	fabricClaim, err := util.Jq(user.Claims, cfg.Fabric.CA.OIDCClaimKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Marshal the fabric claim to JSON
	fabricClaimBytes, err := json.Marshal(fabricClaim)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON into a RegistrationRequest
	var regReq api.RegistrationRequest
	err = json.Unmarshal(fabricClaimBytes, &regReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	regReq.Name = user.Subject

	userDir := filepath.Join(cfg.Fabric.CA.ClientHome, "users", regReq.Name)

	var keyCert *fabric.MSPKeyCert
	_, err = fabric.GetMSPKeyfile(userDir)
	if err != nil {
		if _, err := fabric.RegisterAndEnrollUser(regReq); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	keyCert, err = fabric.LoadMSPKeyCert(userDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	keyCert.Cert = base64.StdEncoding.EncodeToString([]byte(keyCert.Cert))
	keyCert.Key = base64.StdEncoding.EncodeToString([]byte(keyCert.Key))

	pgo.RespondJSON(w, http.StatusOK, keyCert)
}
