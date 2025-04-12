package ki

import (
	"log"

	"github.com/river-now/river/kit/port"
)

const (
	default_refresh_server_port = 10_000
)

func MustGetAppPort() int {
	isDev := GetIsDev()
	portHasBeenSet := getPortHasBeenSet()
	defaultPort := getPort()

	if !isDev || portHasBeenSet {
		return defaultPort
	}

	port, err := port.GetFreePort(defaultPort)
	if err != nil {
		log.Panicf("error: failed to get free port: %v", err)
	}

	setPort(port)
	setPortHasBeenSet()

	return port
}
