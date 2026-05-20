package config

const (
	AppName         = "ffquery"
	Version         = "0.0.0"
	Dayly    Period = "dayly"
	Weekly   Period = "weekly"
	Monthly  Period = "monthly"
	Yearly   Period = "yearly"
	NoPeriod Period = ""
)

type Config struct {
	Persistence Persistence `json:"persistence"`
	Logger      Logger      `json:"logger"`
}

type Persistence struct {
	JSON JsonStore `json:"json"`
}

type JsonStore struct {
	WriteScans bool   `json:"write_scans"`
	Path       string `json:"path"`
	Prefix     string `json:"prefix"`
	Indent     string `json:"indent"`
}

type Logger struct {
	Path             string  `json:"path"`
	FileSizeRotation bool    `json:"file_size_rotation"`
	SizeLimitMB      float64 `json:"size_limit_mb"`
	PeriodicRotation bool    `json:"periodic_rotation"`
	Period           Period  `json:"period"`
}

type Period string
