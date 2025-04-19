package ki

import (
	"fmt"
	"os"
	"strconv"
)

const (
	modeKey              = "wave_MODE"
	devModeVal           = "development"
	portKey              = "PORT"
	portHasBeenSetKey    = "wave_PORT_HAS_BEEN_SET"
	refreshServerPortKey = "wave_REFRESH_SERVER_PORT"
	trueStr              = "true"
)

func GetIsDev() bool {
	return os.Getenv(modeKey) == devModeVal
}

func setPort(port int) {
	os.Setenv(portKey, fmt.Sprintf("%d", port))
}

func getPort() int {
	port, err := strconv.Atoi(os.Getenv(portKey))
	if err != nil {
		return 0
	}
	return port
}

func setPortHasBeenSet() {
	os.Setenv(portHasBeenSetKey, trueStr)
}

func getPortHasBeenSet() bool {
	return os.Getenv(portHasBeenSetKey) == trueStr
}

func getRefreshServerPort() int {
	port, err := strconv.Atoi(os.Getenv(refreshServerPortKey))
	if err != nil {
		return 0
	}
	return port
}

func SetModeToDev() {
	os.Setenv(modeKey, devModeVal)
}

func set_refresh_server_port(port int) {
	os.Setenv(refreshServerPortKey, fmt.Sprintf("%d", port))
}
