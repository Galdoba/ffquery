package cache

import (
	"fmt"

	"github.com/Galdoba/appcontext/xdg"
	"github.com/Galdoba/ffquery/internal/infrastructure/config"
)

func Dir() string {
	cacheDir := xdg.Location(xdg.ForCache(), xdg.WithProgramName(config.AppName))
	fmt.Println(cacheDir)
	return cacheDir
}
