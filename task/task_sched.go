// Package task 本包提供任务的调度能力，通过预处理后的任务链表，根据参数不同的调度，有持续打流的、有指定打流发包个数的。。。
package task

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"go_ping/utils"
	"golang.org/x/net/icmp"
	"os"
	"sync"
	"time"
)

// TaskSchedule 任务调度入口，从这里区分文件 和 单个目标，for循环，建立多个goroutine。
func TaskSchedule(paramInput ParamInput, wg *sync.WaitGroup, fr *FailRate, ctx context.Context, fl *FileTaskItemNumber) {
	taskList := new([]TaskItem)
	if paramInput.DstFile != "" { // 读取文件
		taskList = GenTaskListByFile(paramInput)
	} else if paramInput.DstTarget != "" { // 单个目标
		taskList = GenTaskListBySingleTarget(paramInput)
	}
	if len(*taskList) == 0 {
		fmt.Println(utils.NoTaskError)
		os.Exit(0)
	}
	// 根据-n参数进行任务复制
	taskList = GenTotalTaskList(taskList, paramInput)
	fl.TaskNumber = len(*taskList)
	// 分配到多个goroutine中
	concurrencyTask := GenConcurrencyTaskList(taskList, paramInput.Concurrency)
	taskId := concurrencyTask.TaskId
	for _, list := range concurrencyTask.RoutineTaskList {
		if len(list.TaskItemList) == 0 {
			fmt.Println(utils.NoTaskError)
			os.Exit(0)
		}
		break
	}
	// 持续打流
	if paramInput.Number == 0 {
		instanceName := getInstanceName(paramInput)
		table := NewForeverTable()
		for {
			sTime := time.Now()
			for _, list := range concurrencyTask.RoutineTaskList {
				wg.Add(1)
				go TaskLoop(list, wg, fr, taskId, paramInput, ctx)
			}
			wg.Wait()
			//至少停顿1秒
			eTime := time.Now()
			duration := eTime.Sub(sTime)
			if duration < time.Second {
				time.Sleep(time.Second - duration)
			}
			// 画表
			ShowTableForever(fr, table, instanceName, paramInput.PingType)
		}
	} else { //指定打包次数
		for _, list := range concurrencyTask.RoutineTaskList {
			wg.Add(1)
			go TaskLoop(list, wg, fr, taskId, paramInput, ctx)
		}
	}
}

func TaskScheduleICMP(paramInput ParamInput, wg *sync.WaitGroup, fr *FailRate, ctx context.Context, handle *icmp.PacketConn, handleV6 *icmp.PacketConn, fl *FileTaskItemNumber) {
	taskList := new([]TaskItem)
	// 读取文件
	if paramInput.DstFile != "" { // 读取文件
		taskList = GenTaskListByFile(paramInput)
	} else if paramInput.DstTarget != "" { // 单个目标
		taskList = GenTaskListBySingleTarget(paramInput)
	}
	if len(*taskList) == 0 {
		fmt.Println(utils.NoTaskError)
		os.Exit(0)
	}
	// 根据-n参数进行任务复制
	taskList = GenTotalTaskList(taskList, paramInput)
	fl.TaskNumber = len(*taskList)
	// 分配到多个goroutine中
	concurrencyTask := GenConcurrencyTaskList(taskList, paramInput.Concurrency)
	taskId := concurrencyTask.TaskId
	for _, list := range concurrencyTask.RoutineTaskList {
		if len(list.TaskItemList) == 0 {
			fmt.Println(utils.NoTaskError)
			os.Exit(0)
		}
		break
	}
	// 持续打流
	if paramInput.Number == 0 {
		instanceName := getInstanceName(paramInput)
		table := NewForeverTable()
		for {
			sTime := time.Now()
			icmpSendReceivePkg(paramInput, wg, fr, ctx, handle, handleV6, taskList, concurrencyTask, taskId)
			wg.Wait()
			//至少停顿1秒
			eTime := time.Now()
			duration := eTime.Sub(sTime)
			if duration < time.Second {
				time.Sleep(time.Second - duration)
			}
			// 画表
			ShowTableForever(fr, table, instanceName, paramInput.PingType)
		}
	} else { //指定打包次数
		icmpSendReceivePkg(paramInput, wg, fr, ctx, handle, handleV6, taskList, concurrencyTask, taskId)
	}
}

// icmp一轮发包和收包程序
func icmpSendReceivePkg(paramInput ParamInput, wg *sync.WaitGroup, fr *FailRate, ctx context.Context, handle *icmp.PacketConn, handleV6 *icmp.PacketConn, taskList *[]TaskItem, concurrencyTask *TaskList, taskId string) {
	// 颜色渲染字体
	red := color.New(color.FgRed).SprintFunc()
	// 获取id|seq的集合，判断是本进程发出的icmp包
	icmpIdSeqIpMap := sync.Map{}
	for i := 0; i < len(*taskList); i++ {
		icmpIdSeqIpMap.Store(fmt.Sprintf("%d|%d", (*taskList)[i].IcmpId, (*taskList)[i].IcmpSeq), (*taskList)[i].DstTarget)
	}
	// 发包
	// 发包完成后通知收包
	var wgSend sync.WaitGroup
	var wgReceive sync.WaitGroup
	doneSend := make(chan struct{}) // 创建一个通道用于通知
	// 收包放前面
	wg.Add(2)
	wgReceive.Add(2)
	go IcmpPingReceive(paramInput, handle, wg, fr, ctx, doneSend, &icmpIdSeqIpMap, &wgReceive)
	go IcmpPingReceiveV6(paramInput, handleV6, wg, fr, ctx, doneSend, &icmpIdSeqIpMap, &wgReceive)
	// 发包
	for _, list := range concurrencyTask.RoutineTaskList {
		wg.Add(1)
		wgSend.Add(1)
		go TaskLoopICMP(list, wg, ctx, handle, handleV6, &wgSend)
	}
	// 启动一个 goroutine 来等待所有任务完成，然后发送通知
	go func() {
		wgSend.Wait()
		//fmt.Println("all ip has been send" + time.Now().String())
		time.Sleep(time.Duration(paramInput.Timeout) * time.Second)
		close(doneSend) // 关闭通道用于通知所有等待的 goroutine
	}()
	// 处理未收到的包
	wg.Add(1)
	go func() {
		defer wg.Done()
		wgReceive.Wait()
		icmpIdSeqIpMap.Range(func(key, value interface{}) bool {
			dstIp := value.(string)
			//fmt.Println(fmt.Sprintf("error ip and key %s %s", dstIp, key))
			successNum, failNum := fr.Increment(dstIp, false)
			if paramInput.ShowMode == utils.ShowModeWaterfall && paramInput.Number != 0 {
				sprintf := fmt.Sprintf("%s\t%s\t失败率%.2f%%\t失败%d\t总共%d", dstIp, red("fail"), float64(failNum)*100/float64(failNum+successNum), failNum, failNum+successNum)
				fmt.Println(sprintf)
			}
			// 如果回调返回true，则继续遍历，返回false则停止遍历
			return true
		})
	}()
}

func getInstanceName(paramInput ParamInput) string {
	//获取实例名称
	instanceName := ""
	if paramInput.DstFile != "" {
		instanceName = paramInput.DstFile
		return instanceName
	} else {
		if paramInput.PingType == utils.PingTypeTCP {
			instanceName = fmt.Sprintf("%s|%d", paramInput.DstTarget, paramInput.DstPort)
		} else if paramInput.PingType == utils.PingTypeICMP || paramInput.PingType == utils.PingTypeHTTP {
			instanceName = paramInput.DstTarget
		}
	}
	return instanceName
}
