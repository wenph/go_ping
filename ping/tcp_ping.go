package ping

import (
	"fmt"
	"go_ping/utils"
	"net"
	"strings"
	"time"
)

// TcpPing tcp ping原子函数，每个goroutines执行的
func TcpPing(dstIpOrDomain string, dstPort int, timeout int, srcIp string) bool {
	// 目标地址
	dstAddress := net.JoinHostPort(dstIpOrDomain, fmt.Sprintf("%d", dstPort))
	// 指定超时时间
	if timeout <= 0 {
		timeout = 1
	}
	duration := time.Duration(timeout) * time.Second
	d := net.Dialer{Timeout: duration}
	// 指定源IP
	if srcIp != "" {
		srcAddress := fmt.Sprintf("%s:0", srcIp)
		if strings.Contains(srcIp, ":") {
			srcAddress = fmt.Sprintf("[%s]:0", srcIp)
		}
		srcTCPAddress, err := net.ResolveTCPAddr("tcp", srcAddress)
		if err != nil {
			utils.Log.Errorln(err)
			return false
		}
		d = net.Dialer{
			LocalAddr: srcTCPAddress,
			Timeout:   duration,
		}
	}
	conn, err := d.Dial("tcp", dstAddress)
	result := true
	if err != nil {
		utils.Log.Traceln(err)
		result = false
	}
	if conn != nil {
		err1 := conn.Close()
		if err1 != nil {
			utils.Log.Traceln(err1)
			result = false
		}
	}
	return result
}
