package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/edgeflare/fabric-oidc-proxy/internal/fabric"
	"github.com/edgeflare/pgo/pkg/httputil"
)

type TxRequest struct {
	Func string   `json:"func"`
	Args []string `json:"args"`
}

func submitTxHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := httputil.OIDCUser(r)
	if !ok || claims == nil {
		http.Error(w, "no user found", http.StatusUnauthorized)
		return
	}

	var req TxRequest
	if err := httputil.BindOrError(r, w, &req); err != nil {
		return
	}

	channeID, chaincodeID := r.PathValue("channel"), r.PathValue("chaincode")
	if channeID == "" || chaincodeID == "" {
		http.Error(w, "channel and chaincode name are required", http.StatusBadRequest)
		return
	}

	resultBytes, err := fabric.SubmitTransaction(r.Context(), channeID, chaincodeID, req.Func, req.Args...)
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprintf("failed to submit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	var resultJson json.RawMessage
	if err := json.Unmarshal(resultBytes, &resultJson); err != nil {
		httputil.Text(w, http.StatusOK, string(resultBytes))
		return
	}

	httputil.JSON(w, http.StatusOK, resultJson)
}
