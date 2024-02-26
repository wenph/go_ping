package ping

import (
	"github.com/levigross/grequests"
	"go_ping/utils"
	"net"
	"net/http"
	"time"
)

// HttpPing http ping原子函数，每个goroutines执行的
func HttpPing(domain string, timeout int, srcIp string) bool {
	// 创建grequests的RequestOptions
	if timeout < 1 {
		timeout = 1
	}
	requestTimeout := time.Duration(timeout) * time.Second
	ro := &grequests.RequestOptions{
		RequestTimeout: requestTimeout,
	}
	// 指定源IP
	if srcIp != "" {
		localAddr := net.ParseIP(srcIp)
		// 自定义DialContext函数，允许我们指定源IP地址
		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: localAddr,
			},
			//Timeout:   30 * time.Second,
			//KeepAlive: 30 * time.Second,
		}
		// 创建自定义的Transport，使用上面的Dialer
		transport := &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: dialer.DialContext,
			//TLSHandshakeTimeout: 10 * time.Second,
		}
		// 创建自定义的HTTP客户端
		client := &http.Client{
			Transport: transport,
		}
		// 创建grequests的RequestOptions，配置自定义客户端
		ro = &grequests.RequestOptions{
			HTTPClient:     client,
			RequestTimeout: requestTimeout,
		}
	}
	result := true
	// 使用grequests发出请求，同时使用自定义源IP
	resp, err := grequests.Get(domain, ro)
	if err != nil {
		utils.Log.Traceln(err)
		result = false
	}
	if resp != nil {
		err1 := resp.Close()
		if err1 != nil {
			utils.Log.Traceln(err1)
			result = false
		}
	}
	return result
}
