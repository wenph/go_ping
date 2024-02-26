package show

import (
	flag "github.com/spf13/pflag"
	"strings"
)

func WordSepNormalizeFunc(f *flag.FlagSet, name string) flag.NormalizedName {
	// := 是声明并赋值，并且系统自动推断类型，不需要var关键字
	// []string定义字符串列表，列表里初始化为"-", "_"
	from := []string{"-", "_"}
	to := "."
	for _, sep := range from {
		name = strings.Replace(name, sep, to, -1)
	}
	return flag.NormalizedName(name)
}
