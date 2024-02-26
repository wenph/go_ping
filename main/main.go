package main

import (
	"context"
	"fmt"
	flag "github.com/spf13/pflag"
	"go_ping/ping"
	"go_ping/show"
	"go_ping/task"
	"go_ping/utils"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// 定义命令行参数对应的变量
var (
	version      = flag.BoolP("version", "V", false, "show version")
	dstTarget    = flag.StringP("dst.target", "d", "", "打流目的目标，可以填写目标IP/域名/网段\n和文件互斥，使用文件就无需使用此参数")
	dstPort      = flag.IntP("dst.port", "p", utils.DefaultPortNumber, "打流目的端口，取值[1~65535)")
	dstFile      = flag.StringP("dst.file", "f", "", "指定存放目的信息的文件路径，文件内容每行的格式：\n如果是tcp打流(IP PORT)：1.1.1.1 80 或者 1.1.1.0/24 80\n如果是icmp打流：1.1.1.1 或者 1.1.1.0/24\n如果是http打流(域名不能以http开头)：1.1.1.1 80 或者 1.1.1.0/24 80 或者 taobao.com 80")
	dstFileLoose = flag.BoolP("dst.file.loose", "L", false, "文件格式校验模式，此参数可打开宽松模式，默认严格模式\n严格模式：TCP和HTTP打流 文件内必须包含端口信息，ICMP不能包含端口信息\n宽松模式：系统会根据-p参数自动加上或去掉端口信息")
	srcIp        = flag.StringP("src.ip", "s", "", "指定源IP")
	pingType     = flag.StringP("ping.type", "t", "tcp", "打流类型，取值[tcp,icmp,http]\nicmp打流需要使用root权限")
	timeout      = flag.IntP("ping.timeout", "m", 1, "设置超时时间，单位秒，取值[1~10]")
	concurrency  = flag.IntP("ping.concurrency", "c", 11, "设置总并发数，取值[1~100)")
	number       = flag.IntP("ping.number", "n", 100, "指定每个IP或域名的打流次数，取值[0~100000]，0表示持续打流 即不指定打流次数")
	showMode     = flag.StringP("show.mode", "o", "table", "指定展示模式，取值：\ntable：表格输出，持续打流模式只能表格输出\nwaterfall：瀑布展示，即一行一行日志输出\njson：json格式，适用于对接系统")
	domainA      = flag.BoolP("domain.a", "a", false, "打流域名下解析的A记录，打流结合-d和-p使用")
	logLevel     = flag.StringP("log.level", "l", "info", "设置日志级别，debug/info/warn/error，日志输出到/tmp/go_ping.log")
)

var wg sync.WaitGroup

func main() {
	//创建监听退出chan
	ctx, cancel := context.WithCancel(context.Background())
	//初始化信号
	chSignal := make(chan os.Signal)
	//监听指定信号 ctrl+c kill
	signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	go func() {
		for s := range chSignal {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				// 发送取消信号
				cancel()
				os.Exit(0)
			default:
			}
		}
	}()
	// 设置标准化参数名称的函数
	flag.CommandLine.SetNormalizeFunc(show.WordSepNormalizeFunc)
	// 把用户传递的命令行参数解析为对应变量的值
	flag.Parse()
	// 审计
	params := map[string]string{}
	flag.Visit(func(f *flag.Flag) {
		key := fmt.Sprintf("%s", f.Name)
		value := fmt.Sprintf("%s", f.Value)
		params[key] = value
	})
	// 收集参数，待后面使用
	paramInput := task.ParamInput{
		DstTarget:    *dstTarget,
		DstPort:      *dstPort,
		DstFile:      *dstFile,
		DstFileLoose: *dstFileLoose,
		SrcIp:        *srcIp,
		PingType:     *pingType,
		Timeout:      *timeout,
		Concurrency:  *concurrency,
		Number:       *number,
		ShowMode:     *showMode,
		LogLevel:     *logLevel,
		DomainA:      *domainA,
	}
	// 校验参数
	utils.ValidateParams(params)
	// 设置日志级别
	utils.SetLogLevel(*logLevel)
	// 软件版本
	show.ShowOrUpgradeVersion(*version)
	// 统计时间
	startTime := time.Now()
	// 数据统计
	fr := task.NewFailRate()
	// 文件行数统计
	fl := task.FileTaskItemNumber{}
	if *showMode == utils.ShowModeJson {
		fmt.Println("目前还不支持JSON格式输出，敬请期待...")
		os.Exit(0)
	}
	// icmp要以root权限发包
	if *pingType == utils.PingTypeICMP && runtime.GOOS != "windows" && os.Getuid() != 0 {
		fmt.Println("请以root(sudo)权限运行！")
		os.Exit(0)
	}
	if *pingType == utils.PingTypeTCP || *pingType == utils.PingTypeHTTP {
		task.TaskSchedule(paramInput, &wg, fr, ctx, &fl)
	} else if *pingType == utils.PingTypeICMP {
		// 构建conn
		handle, _ := ping.GenSendHandle(paramInput.SrcIp)
		handleV6, _ := ping.GenSendHandleV6(paramInput.SrcIp)
		defer handle.Close()
		defer handleV6.Close()
		task.TaskScheduleICMP(paramInput, &wg, fr, ctx, handle, handleV6, &fl)
	}
	// 如果用户指定发包数
	if *number != 0 {
		taskNum := fl.TaskNumber
		if *showMode == utils.ShowModeTable {
			go task.ShowTableLoop(ctx, fr, taskNum, paramInput)
		}
		wg.Wait()               // 等待所有登记的goroutine都结束
		time.Sleep(time.Second) // 等待表格再刷最后一遍，防止显示半个表格
		cancel()                // 关闭通道，向所有 goroutine 发送停止信号
	}
	// 等待其他goroutine清理现场
	time.Sleep(time.Second)
	fmt.Println("总共花费时间：", time.Since(startTime))
}
