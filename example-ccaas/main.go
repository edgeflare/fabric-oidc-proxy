package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/edgeflare/fabric-oidc-proxy/example-ccaas/asset_cc"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type serverConfig struct {
	CCID    string
	Address string
}

func main() {
	// Define flags with default values from environment variables
	ccid := flag.String("ccid", getEnvOrDefault("CHAINCODE_ID", "assetcc"), "Chaincode ID")
	address := flag.String("address", getEnvOrDefault("CHAINCODE_SERVER_ADDRESS", "0.0.0.0:7052"), "CC server address")
	tlsDisabled := flag.String("tls-disabled", getEnvOrDefault("CHAINCODE_TLS_DISABLED", "true"), "TLS disabled")
	tlsKey := flag.String("tls-key", getEnvOrDefault("CHAINCODE_TLS_KEY", "/fabric/chaincode/tls/server.key"), "TLS key")
	tlsCert := flag.String("tls-cert", getEnvOrDefault("CHAINCODE_TLS_CERT", "/fabric/chaincode/tls/server.crt"), "TLScrt")
	clientCACert := flag.String("tls-client-cacert", getEnvOrDefault("CHAINCODE_TLS_CLIENT_CACERT", ""), "Client CA cert")

	// Parse the flags
	flag.Parse()

	config := serverConfig{
		CCID:    *ccid,
		Address: *address,
	}

	log.Printf("Starting chaincode server with ID: %s and address: %s", config.CCID, config.Address)

	chaincode, err := contractapi.NewChaincode(&asset_cc.AssetContract{})
	if err != nil {
		log.Panicf("Error creating %s chaincode: %s", os.Getenv("CHAINCODE_NAME"), err)
	}

	server := &shim.ChaincodeServer{
		CCID:     config.CCID,
		Address:  config.Address,
		CC:       chaincode,
		TLSProps: getTLSProperties(*tlsDisabled, *tlsKey, *tlsCert, *clientCACert),
	}

	log.Println("Chaincode server configured. Attempting to start...")

	go func() {
		if err := server.Start(); err != nil {
			log.Panicf("Error starting %s chaincode: %s", os.Getenv("CHAINCODE_NAME"), err)
		}
		log.Println("Chaincode server started successfully.")
	}()

	// Wait for the server to start successfully
	select {}
}

func getTLSProperties(tlsDisabledStr, key, cert, clientCACert string) shim.TLSProperties {
	// Convert tlsDisabledStr to boolean
	tlsDisabled := getBoolOrDefault(tlsDisabledStr, false)
	var keyBytes, certBytes, clientCACertBytes []byte
	var err error

	if !tlsDisabled {
		log.Println("TLS is enabled. Reading TLS key and certificate files.")
		keyBytes, err = os.ReadFile(key)
		if err != nil {
			log.Panicf("Error while reading %s. %s", key, err)
		}
		certBytes, err = os.ReadFile(cert)
		if err != nil {
			log.Panicf("Error while reading %s. %s", cert, err)
		}
	}

	if clientCACert != "" {
		log.Println("Client CA certificate is provided. Reading file.")
		clientCACertBytes, err = os.ReadFile(clientCACert)
		if err != nil {
			log.Panicf("Error while reading %s. %s", clientCACert, err)
		}
	}

	return shim.TLSProperties{
		Disabled:      tlsDisabled,
		Key:           keyBytes,
		Cert:          certBytes,
		ClientCACerts: clientCACertBytes,
	}
}

func getEnvOrDefault(env, defaultVal string) string {
	value, ok := os.LookupEnv(env)
	if !ok {
		value = defaultVal
		log.Printf("Environment variable %s not set. Using default value: %s", env, defaultVal)
	} else {
		log.Printf("Environment variable %s set. Using value: %s", env, value)
	}
	return value
}

// Returns default value if the string cannot be parsed
func getBoolOrDefault(value string, defaultVal bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Error parsing boolean value: %s. Using default: %v", value, defaultVal)
		return defaultVal
	}
	return parsed
}
