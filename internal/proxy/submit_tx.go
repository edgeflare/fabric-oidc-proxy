package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/edgeflare/fabric-oidc-proxy/internal/fabric"
	"github.com/edgeflare/pgo"
)

type TxRequest struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func submitTxHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := pgo.OIDCUser(r)
	if !ok || user.Active == false {
		http.Error(w, "no user found", http.StatusUnauthorized)
		return
	}

	var req TxRequest
	if err := pgo.BindOrRespondError(r, w, &req); err != nil {
		return
	}

	channeID, chaincodeID := r.PathValue("channel"), r.PathValue("chaincode")
	if channeID == "" || chaincodeID == "" {
		http.Error(w, "channel and chaincode name are required", http.StatusBadRequest)
		return
	}

	resultBytes, err := fabric.SubmitTransaction(r.Context(), channeID, chaincodeID, req.Name, req.Args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to submit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	var resultJson json.RawMessage
	if err := json.Unmarshal(resultBytes, &resultJson); err != nil {
		pgo.RespondText(w, http.StatusOK, string(resultBytes))
		return
	}

	pgo.RespondJSON(w, http.StatusOK, resultJson)
}
