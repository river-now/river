package envutil

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func GetStr(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func GetInt(key string, defaultValue int) int {
	strValue := GetStr(key, strconv.Itoa(defaultValue))
	value, err := strconv.Atoi(strValue)
	if err == nil {
		return value
	}
	return defaultValue
}

func GetBool(key string, defaultValue bool) bool {
	strValue := GetStr(key, strconv.FormatBool(defaultValue))
	value, err := strconv.ParseBool(strValue)
	if err == nil {
		return value
	}
	return defaultValue
}

type Env interface {
	GetIsDev() bool
	GetPort() int
}

const (
	ModeKey       = "MODE"
	ModeValueProd = "production"
	ModeValueDev  = "development"
	PortKey       = "PORT"
)

type Base struct {
	Mode   string
	IsDev  bool
	IsProd bool
	Port   int
}

func (e *Base) GetIsDev() bool { return e.IsDev }
func (e *Base) GetPort() int   { return e.Port }

type InitOptions struct {
	FallbackGetPortFunc func() int
	GetIsDevFunc        func() bool
}

func InitBase(options InitOptions) (Base, error) {
	base := Base{}

	err := godotenv.Load()
	if err != nil {
		err = fmt.Errorf("envutil: failed to load .env file: %w", err)
	}

	if options.GetIsDevFunc == nil {
		base.Mode = GetStr(ModeKey, ModeValueProd)
		base.IsDev = base.Mode == ModeValueDev
		base.IsProd = base.Mode == ModeValueProd
	} else {
		base.IsDev = options.GetIsDevFunc()
		base.IsProd = !base.IsDev
		base.Mode = ModeValueProd
		if base.IsDev {
			base.Mode = ModeValueDev
		}
	}

	if options.FallbackGetPortFunc == nil {
		base.Port = GetInt(PortKey, 8080)
	} else {
		base.Port = GetInt(PortKey, options.FallbackGetPortFunc())
	}

	return base, err
}

// SetDevMode sets the MODE environment variable to "development".
func SetDevMode() error {
	return os.Setenv(ModeKey, ModeValueDev)
}
