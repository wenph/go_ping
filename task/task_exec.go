// Package task 本包提供任务的批量执行，并把结果写到合适的结构体中。
package task

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"go_ping/ping"
	"go_ping/utils"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"sync"
	"time"
)

// TaskLoop 循环执行每个探测任务
func TaskLoop(taskList *RoutineTaskItem, wg *sync.WaitGroup, fr *FailRate, taskId string, paramInput ParamInput, ctx context.Context) {
	defer wg.Done() // goroutine结束就登记-1
	// 颜色渲染字体
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	routineId := taskList.RoutineId
	taskListLength := len(taskList.TaskItemList)
	taskIndex := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 如果执行完了，就退出
			if taskIndex >= taskListLength {
				return
			}
			item := taskList.TaskItemList[taskIndex]
			// tcp 打流
			switch item.PingType {
			case utils.PingTypeTCP:
				r := ping.TcpPing(item.DstTarget, item.DstPort, item.Timeout, item.SrcIp)
				colorOutPut := red("fail")
				if r {
					colorOutPut = green("success")
				}
				successNum, failNum := fr.Increment(fmt.Sprintf("%s|%d", item.DstTarget, item.DstPort), r)
				if paramInput.ShowMode == utils.ShowModeWaterfall && paramInput.Number != 0 {
					sprintf := fmt.Sprintf("第%02d-%06d批次\t%s\t%d\t%s\t失败率%.2f%%\t失败%d\t总共%d", routineId, item.Id, item.DstTarget, item.DstPort, colorOutPut, float64(failNum)*100/float64(failNum+successNum), failNum, failNum+successNum)
					fmt.Println(sprintf)
				}
			case utils.PingTypeHTTP:
				r := ping.HttpPing(item.DstTarget, item.Timeout, item.SrcIp)
				colorOutPut := red("fail")
				if r {
					colorOutPut = green("success")
				}
				successNum, failNum := fr.Increment(item.DstTarget, r)
				if paramInput.ShowMode == utils.ShowModeWaterfall && paramInput.Number != 0 {
					sprintf := fmt.Sprintf("第%02d-%06d批次\t%s\t%s\t失败率%.2f%%\t失败%d\t总共%d", routineId, item.Id, item.DstTarget, colorOutPut, float64(failNum)*100/float64(failNum+successNum), failNum, failNum+successNum)
					fmt.Println(sprintf)
				}
			}

			// 自增，循环知道这个goroutine执行完所有任务
			taskIndex++
		}
	}
}

// TaskLoopICMP 循环执行每个探测任务
func TaskLoopICMP(taskList *RoutineTaskItem, wg *sync.WaitGroup, ctx context.Context, handle *icmp.PacketConn, handleV6 *icmp.PacketConn, wgSend *sync.WaitGroup) {
	defer wg.Done()     // goroutine结束就登记-1
	defer wgSend.Done() // goroutine结束就登记-1
	//routineId := taskList.RoutineId
	taskListLength := len(taskList.TaskItemList)
	taskIndex := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 如果执行完了，就退出
			if taskIndex >= taskListLength {
				return
			}
			item := taskList.TaskItemList[taskIndex]
			// tcp 打流
			switch item.PingType {
			case utils.PingTypeICMP:
				ping.IcmpPingSend(item.DstTarget, handle, handleV6, item.IcmpId, item.IcmpSeq, item.IcmpSendInterval)
			}

			// 自增，循环知道这个goroutine执行完所有任务
			taskIndex++
		}
	}
}

// IcmpPingReceive icmp ping接收函数，1个goroutines执行的
func IcmpPingReceive(paramInput ParamInput, c *icmp.PacketConn, wg *sync.WaitGroup, fr *FailRate, ctx context.Context, done chan struct{}, idSeqIpMap *sync.Map, doneReceive *sync.WaitGroup) {
	defer wg.Done() // goroutine结束就登记-1
	defer doneReceive.Done()
	// 颜色渲染字体
	green := color.New(color.FgGreen).SprintFunc()
	// 设置接收超时
	timeout := paramInput.Timeout
	if paramInput.Timeout < 1 {
		timeout = 1
	}
	i := 0
	for {
		i++
		//now := time.Now()
		//fmt.Println(fmt.Sprintf("1---%d---time-%v", i, now))
		err := c.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
		if err != nil {
			utils.Log.Errorln("SetReadDeadline error: ", err)
		}
		//fmt.Println(fmt.Sprintf("2---%d---time-%v-%v", i, time.Now(), time.Since(now)))
		select {
		case <-ctx.Done():
			return
		case <-done:
			// 所有发包都已经完成
			//fmt.Println("rev all ip send", time.Now().String())
			return // 超时了，退出 goroutine
		default:
			// 准备接收回复
			//fmt.Println(fmt.Sprintf("3---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			reply := make([]byte, 1500)
			//fmt.Println(fmt.Sprintf("3.5---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			n, peer, err1 := c.ReadFrom(reply)
			//fmt.Println(fmt.Sprintf("4---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			//now = time.Now()
			if err1 != nil {
				if netErr, ok := err1.(*net.OpError); ok && netErr.Timeout() {
					utils.Log.Traceln("timeout error:", err1)
					continue
				}
				utils.Log.Errorln("ReadFrom error:", err1)
				continue
			}
			//fmt.Println(fmt.Sprintf("5---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			dstIp := peer.String()
			receivedMessage, err2 := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
			if err2 != nil {
				utils.Log.Errorln("ParseMessage error: ", err2)
				continue
			}
			//fmt.Println(fmt.Sprintf("6---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			// 检查回复类型
			if receivedMessage.Type == ipv4.ICMPTypeEchoReply {
				echo, ok := receivedMessage.Body.(*icmp.Echo)
				if ok {
					//fmt.Println(fmt.Sprintf("7---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					id := echo.ID
					seq := echo.Seq
					key := fmt.Sprintf("%d|%d", id, seq)
					value, exists := (*idSeqIpMap).Load(key)
					dstIpOrDomain, _ := value.(string)
					//fmt.Println(fmt.Sprintf("8---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					if exists {
						// 写往fr
						//fmt.Println(key)
						(*idSeqIpMap).Delete(key)
						//fmt.Println(fmt.Sprintf("9---%d---time-%v-%v", i, time.Now(), time.Since(now)))
						successNum, failNum := fr.Increment(dstIpOrDomain, true)
						// 打印调试日志
						//fmt.Println(fmt.Sprintf("success receive dstIp: %s, successNum: %d, failNum: %d, %s", dstIp, successNum, failNum, time.Now().String()))
						if paramInput.ShowMode == utils.ShowModeWaterfall && paramInput.Number != 0 {
							sprintf := fmt.Sprintf("%s\t%s\t失败率%.2f%%\t失败%d\t总共%d", dstIp, green("success"), float64(failNum)*100/float64(failNum+successNum), failNum, failNum+successNum)
							fmt.Println(sprintf)
						}
						//fmt.Println(fmt.Sprintf("10---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					} else {
						utils.Log.Traceln("other process ping, src ip:", dstIp, "key:", key)
					}
				} else {
					utils.Log.Traceln("Error asserting Echo Reply Body", dstIp)
				}
			} else {
				utils.Log.Traceln("Got non-echo reply type:", receivedMessage.Type)
			}
		}
	}
}

// IcmpPingReceiveV6 icmp v6 ping接收函数，1个goroutines执行的
func IcmpPingReceiveV6(paramInput ParamInput, c *icmp.PacketConn, wg *sync.WaitGroup, fr *FailRate, ctx context.Context, done chan struct{}, idSeqIpMap *sync.Map, doneReceive *sync.WaitGroup) {
	defer wg.Done() // goroutine结束就登记-1
	defer doneReceive.Done()
	// 颜色渲染字体
	green := color.New(color.FgGreen).SprintFunc()
	// 设置接收超时
	timeout := paramInput.Timeout
	if paramInput.Timeout < 1 {
		timeout = 1
	}
	i := 0
	for {
		i++
		//now := time.Now()
		//fmt.Println(fmt.Sprintf("1---%d---time-%v", i, now))
		err := c.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
		if err != nil {
			utils.Log.Errorln("SetReadDeadline error: ", err)
		}
		//fmt.Println(fmt.Sprintf("2---%d---time-%v-%v", i, time.Now(), time.Since(now)))
		select {
		case <-ctx.Done():
			return
		case <-done:
			// 所有发包都已经完成
			//fmt.Println("rev all ip send", time.Now().String())
			return // 超时了，退出 goroutine
		default:
			// 准备接收回复
			//fmt.Println(fmt.Sprintf("3---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			reply := make([]byte, 1500)
			//fmt.Println(fmt.Sprintf("3.5---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			n, peer, err1 := c.ReadFrom(reply)
			//fmt.Println(fmt.Sprintf("4---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			//now = time.Now()
			if err1 != nil {
				if netErr, ok := err1.(*net.OpError); ok && netErr.Timeout() {
					utils.Log.Traceln("timeout error:", err1)
					continue
				}
				utils.Log.Errorln("ReadFrom error:", err1)
				continue
			}
			//fmt.Println(fmt.Sprintf("5---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			dstIp := peer.String()
			receivedMessage, err2 := icmp.ParseMessage(ipv6.ICMPTypeEchoReply.Protocol(), reply[:n])
			if err2 != nil {
				utils.Log.Errorln("ParseMessage error: ", err2)
				continue
			}
			//fmt.Println(fmt.Sprintf("6---%d---time-%v-%v", i, time.Now(), time.Since(now)))
			// 检查回复类型
			if receivedMessage.Type == ipv6.ICMPTypeEchoReply {
				echo, ok := receivedMessage.Body.(*icmp.Echo)
				if ok {
					//fmt.Println(fmt.Sprintf("7---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					id := echo.ID
					seq := echo.Seq
					key := fmt.Sprintf("%d|%d", id, seq)
					value, exists := (*idSeqIpMap).Load(key)
					dstIpOrDomain, _ := value.(string)
					//fmt.Println(fmt.Sprintf("8---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					if exists {
						// 写往fr
						//fmt.Println(key)
						(*idSeqIpMap).Delete(key)
						//fmt.Println(fmt.Sprintf("9---%d---time-%v-%v", i, time.Now(), time.Since(now)))
						successNum, failNum := fr.Increment(dstIpOrDomain, true)
						// 打印调试日志
						//fmt.Println(fmt.Sprintf("success receive dstIp: %s, successNum: %d, failNum: %d, %s", dstIp, successNum, failNum, time.Now().String()))
						if paramInput.ShowMode == utils.ShowModeWaterfall && paramInput.Number != 0 {
							sprintf := fmt.Sprintf("%s\t%s\t失败率%.2f%%\t失败%d\t总共%d", dstIp, green("success"), float64(failNum)*100/float64(failNum+successNum), failNum, failNum+successNum)
							fmt.Println(sprintf)
						}
						//fmt.Println(fmt.Sprintf("10---%d---time-%v-%v", i, time.Now(), time.Since(now)))
					} else {
						utils.Log.Traceln("other process ping, src ip:", dstIp, "key:", key)
					}
				} else {
					utils.Log.Traceln("Error asserting Echo Reply Body", dstIp)
				}
			} else {
				utils.Log.Traceln("Got non-echo reply type:", receivedMessage.Type)
			}
		}
	}
}
