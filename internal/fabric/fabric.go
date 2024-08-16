package fabric

import (
	"github.com/edgeflare/fabric-oidc-proxy/internal/config"
	"go.uber.org/zap"
)

var (
	cfg config.Config
)

// Init initializes client config to interact with the Fabric network
func Init(conf *config.Config, lgr *zap.Logger) error {
	cfg = *conf
	return nil
}
