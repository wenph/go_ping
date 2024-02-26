package ping

import (
	"go_ping/utils"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"os"
	"strings"
	"time"
)

func GenSendHandle(srcIp string) (*icmp.PacketConn, error) {
	// 使用特权模式监听ICMP数据包（需要管理员权限）
	var c *icmp.PacketConn
	var err error
	if srcIp != "" {
		// 创建监听用的地址对象
		laddr := &net.IPAddr{IP: net.ParseIP(srcIp)}
		// 创建ICMP监听
		c, err = icmp.ListenPacket("ip4:icmp", laddr.String())
	} else {
		c, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	}
	if err != nil {
		utils.Log.Errorln("创建ICMP监听", err)
		os.Exit(1)
	}
	return c, err
}

// GenSendHandleV6 支持ipv6监听
func GenSendHandleV6(srcIp string) (*icmp.PacketConn, error) {
	// 使用特权模式监听ICMP数据包（需要管理员权限）
	var c *icmp.PacketConn
	var err error
	if srcIp != "" {
		// 创建监听用的地址对象
		laddr := &net.IPAddr{IP: net.ParseIP(srcIp)}
		// 创建ICMP监听
		c, err = icmp.ListenPacket("ip6:ipv6-icmp", laddr.String())
	} else {
		c, err = icmp.ListenPacket("ip6:ipv6-icmp", "::") // 使用 "::" 作为本地地址，表示任意IPv6地址
	}
	if err != nil {
		utils.Log.Errorln("创建ICMP监听", err)
		os.Exit(1)
	}
	return c, err
}

// IcmpPingSend icmp ping发送函数，每个goroutines执行的
func IcmpPingSend(dstIpOrDomain string, handle *icmp.PacketConn, handleV6 *icmp.PacketConn, icmpId int, icmpSeq int, icmpSendPkgInterval int) bool {
	// 目标地址
	netType := "ip4"
	if strings.Contains(dstIpOrDomain, ":") {
		netType = "ip6"
	}
	dst, err := net.ResolveIPAddr(netType, dstIpOrDomain)
	if err != nil {
		utils.Log.Errorln("目标地址出错", err)
		return false
	}

	// 创建一个ICMP消息
	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, // ICMP回显请求
		Code: 0,
		Body: &icmp.Echo{
			ID:   icmpId,  // 使用进程ID作为标识符，0 到 65535（2^16 - 1）
			Seq:  icmpSeq, // 序列号，0 到 65535（2^16 - 1）
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	// ipv6
	if strings.Contains(dstIpOrDomain, ":") {
		message = icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest, // Echo请求类型
			Code: 0,
			Body: &icmp.Echo{
				ID:   icmpId,  // 使用进程ID作为标识符，0 到 65535（2^16 - 1）
				Seq:  icmpSeq, // 序列号，0 到 65535（2^16 - 1）
				Data: []byte("HELLO-R-U-THERE"),
			},
		}
	}

	// 将ICMP消息编码为字节
	binaryMessage, err := message.Marshal(nil)
	if err != nil {
		utils.Log.Errorln("将ICMP消息编码为字节出错", err)
		return false
	}

	// 发送消息
	if strings.Contains(dstIpOrDomain, ":") {
		if _, err = handleV6.WriteTo(binaryMessage, dst); err != nil {
			utils.Log.Errorln("发送消息出错", err)
			return false
		}
	} else {
		if _, err = handle.WriteTo(binaryMessage, dst); err != nil {
			utils.Log.Errorln("发送消息出错", err)
			return false
		}
	}
	utils.Log.Traceln("success send icmp to", dst, time.Now().String())
	time.Sleep(time.Duration(icmpSendPkgInterval) * time.Millisecond)
	return true
}
