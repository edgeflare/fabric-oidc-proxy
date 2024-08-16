```shell
openssl s_client -connect ca-org1.fabnet.edgeflare.dev:443 -showcerts </dev/null 2>/dev/null | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > ${PWD}/fabric/tls/ca.crt
```

```shell
export OIDC_ISSUER=https://iam-b45a263d486c.asia-southeast1.edgeflare.dev
export OIDC_CLIENT_ID=$(ls __oidc/)
export OIDC_CLIENT_SECRET=$(cat __oidc/$(ls __oidc/))
export FABRIC_CA_URL=https://ca-org1.fabnet.edgeflare.dev
export FABRIC_CA_ADMIN=admin
export FABRIC_CA_ADMIN_SECRET=adminpw
export FABRIC_CA_TLS_TRUSTED_CERTS=$PWD/fabric/tls/ca.crt
export FABRIC_GW_PEER_SERVER_NAME_OVERRIDE=peer0-org1.fabnet.edgeflare.dev
export FABRIC_GW_PEER_ENDPOINT=peer0-org1.fabnet.edgeflare.dev:443
```
