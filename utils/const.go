package utils

var (
	PingTypeTCP           = "tcp"
	PingTypeICMP          = "icmp"
	PingTypeHTTP          = "http"
	PingTypeList          = []string{PingTypeTCP, PingTypeICMP, PingTypeHTTP}
	ShowModeWaterfall     = "waterfall"
	ShowModeTable         = "table"
	ShowModeJson          = "json"
	ShowModeList          = []string{ShowModeWaterfall, ShowModeTable, ShowModeJson}
	MaxIcmpNum            = 65535
	IcmpSendIntervalMac   = 9       // 毫秒
	IcmpSendIntervalLinux = 1       // 毫秒
	ErrorLevel            = "error" // 日志级别
	WarnLevel             = "warn"
	InfoLevel             = "info"
	DebugLevel            = "debug"
	NoTaskError           = "没有可执行的任务，请检查参数！"
	DomainMaxLen          = 100
	DefaultPortNumber     = 80
)
