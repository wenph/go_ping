package utils

import (
	"fmt"
	"github.com/malfunkt/iprange"
	"net"
	"os"
)

func getLocalIpList() []string {
	localIpList := []string{}
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// 检查IP地址断言是否为net.IPNet类型
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIpList = append(localIpList, ipnet.IP.String())
			}
		}
	}
	return localIpList
}

func GenIpListBySegment(cidr string) []string {
	// 解析CIDR获取IPNet结构体
	if !ValidCIDR(cidr) {
		fmt.Println(fmt.Sprintf("网络地址格式错误：%s", cidr))
		os.Exit(0)
	}
	list, err := iprange.ParseList(cidr)
	if err != nil {
		fmt.Println(fmt.Sprintf("网络地址翻译出错：%s", cidr))
		os.Exit(0)
	}
	ranges := list.Expand()
	// 获取网络起始地址
	var ips []string
	for _, ip := range ranges {
		ips = append(ips, ip.String())
	}
	return ips
}

func GenIpListByDomain(domain string) []string {
	var ipList []string
	// 查找域名对应的IP地址
	ips, err := net.LookupIP(domain)
	if err != nil {
		Log.Errorln(fmt.Sprintf("无法获取IP: %v", err))
		return ipList
	}
	for _, ip := range ips {
		ipList = append(ipList, ip.String())
	}
	return ipList
}
