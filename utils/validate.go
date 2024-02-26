package utils

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"net"
	"os"
	"strconv"
	"strings"
)

func ValidateParams(params map[string]string) {
	for key, value := range params {
		switch key {
		case "ping.type":
			if !ContainsString(PingTypeList, value) {
				fmt.Println("打流类型格式错误")
				os.Exit(0)
			}
		case "src.ip":
			if !ValidateIP(value) {
				fmt.Println("源IP格式错误")
				os.Exit(0)
			}
		case "dst.target":
			// 如果是ip，就通过，如果不是ip，再继续校验
			if !ValidateIP(value) {
				// 如果是域名，就通过，如果不是域名，再继续校验
				if !ValidCIDR(value) {
					// 如果不是域名，就校验网段
					if !ValidateDomain(value) {
						fmt.Println("目的信息格式错误")
						os.Exit(0)
					} else {
						// 域名长度不能太长
						if len(value) > DomainMaxLen {
							fmt.Println("目的信息格式错误")
							os.Exit(0)
						}
					}
				}
			}
		case "dst.port":
			if !govalidator.IsNumeric(value) {
				fmt.Println("目的端口格式错误")
				os.Exit(0)
			}
			valueInt, _ := strconv.Atoi(value)
			if valueInt <= 0 || valueInt >= 65535 {
				fmt.Println("目的端口格式错误")
				os.Exit(0)
			}
		case "ping.timeout":
			if !govalidator.IsNumeric(value) {
				fmt.Println("超时时间格式错误")
				os.Exit(0)
			}
			valueInt, _ := strconv.Atoi(value)
			if valueInt <= 0 || valueInt > 10 {
				fmt.Println("超时时间格式错误")
				os.Exit(0)
			}
		case "ping.concurrency":
			if !govalidator.IsNumeric(value) {
				fmt.Println("总并发数格式错误")
				os.Exit(0)
			}
			valueInt, _ := strconv.Atoi(value)
			if valueInt <= 0 || valueInt >= 100 {
				fmt.Println("总并发数格式错误")
				os.Exit(0)
			}
		case "ping.number":
			if !govalidator.IsNumeric(value) {
				fmt.Println("每个IP发包数目格式错误")
				os.Exit(0)
			}
			valueInt, _ := strconv.Atoi(value)
			if valueInt < 0 || valueInt > 100000 {
				fmt.Println("每个IP发包数目格式错误")
				os.Exit(0)
			}
		case "dst.file":
			if !FileExists(value) {
				fmt.Println("文件路径不存在")
				os.Exit(0)
			} else {
				content, err := os.ReadFile(value)
				if err != nil {
					fmt.Println(err)
					os.Exit(0)
				}
				fileContent := string(content)
				split := strings.Split(fileContent, "\n")
				if len(split) == 0 {
					fmt.Println("文件内容为空")
					os.Exit(0)
				}
			}
		case "show.mode":
			if !ContainsString(ShowModeList, value) {
				fmt.Println("展示模式格式错误")
				os.Exit(0)
			}
		case "log.level":
			if value != ErrorLevel && value != WarnLevel && value != InfoLevel && value != DebugLevel {
				fmt.Println("日志级别格式错误")
				os.Exit(0)
			}
		}
	}
}

// ContainsString 检查切片中是否包含一个整数
func ContainsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func ValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func ValidateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return true
}

func ValidateDomain(domain string) bool {
	return govalidator.IsDNSName(domain)
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // 文件存在
	}
	if os.IsNotExist(err) {
		return false // 文件不存在
	}
	// 因为其他原因无法确定文件是否存在，可能是权限问题或其他错误
	return false
}

func GenIcmpIdAndSeq(num int) (int, int) {
	id := num / MaxIcmpNum
	seq := num % MaxIcmpNum
	id = MaxIcmpNum - id
	return id, seq
}
