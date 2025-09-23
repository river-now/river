package ki

import (
	"flag"
)

func (c *Config) BuildWaveWithHook(hook func(isDev bool) error) {
	devModeFlag := flag.Bool("dev", false, "set dev mode")
	hookModeFlag := flag.Bool("hook", false, "set hook mode")
	noBinaryFlag := flag.Bool("no-binary", false, "skip go binary compilation")

	flag.Parse()

	isDev := *devModeFlag
	isHook := *hookModeFlag
	noBinary := *noBinaryFlag

	if isHook {
		if err := hook(isDev); err != nil {
			panic(err)
		}
		return
	}

	if isDev {
		c.MustStartDev()
		return
	}

	if err := c.BuildWave(BuildOptions{RecompileGoBinary: !noBinary}); err != nil {
		panic(err)
	}
}
