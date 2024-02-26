// Package task 本包提供任务的预处理函数，在任务执行前，根据参数处理得到任务列表。
package task

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/bwmarrin/snowflake"
	mapset "github.com/deckarep/golang-set"
	"go_ping/utils"
	"os"
	"runtime"
	"strconv"
	"strings"
)

/*
GenTaskList 根据参数生成tcp总列表
目标 分为 IP 网段 域名
域名 分为正常域名 和 域名下的A记录
*/
func GenTaskList(paramInput ParamInput, port int) *[]TaskItem {
	totalTaskList := []TaskItem{}
	// ip 网段 域名
	if utils.ValidateIP(paramInput.DstTarget) { // ip
		taskItem := TaskItem{
			DstTarget: paramInput.DstTarget,
			DstPort:   port,
			SrcIp:     paramInput.SrcIp,
			Timeout:   paramInput.Timeout,
			PingType:  paramInput.PingType,
		}
		totalTaskList = append(totalTaskList, taskItem)
	} else if utils.ValidCIDR(paramInput.DstTarget) { // 网段
		ips := utils.GenIpListBySegment(paramInput.DstTarget)
		for k := 0; k < len(ips); k++ {
			taskItem := TaskItem{
				DstTarget: ips[k],
				DstPort:   port,
				SrcIp:     paramInput.SrcIp,
				Timeout:   paramInput.Timeout,
				PingType:  paramInput.PingType,
			}
			totalTaskList = append(totalTaskList, taskItem)
		}
	} else if utils.ValidateDomain(paramInput.DstTarget) { // 域名
		if paramInput.DomainA { // 域名下的A记录
			ips := utils.GenIpListByDomain(paramInput.DstTarget)
			for k := 0; k < len(ips); k++ {
				taskItem := TaskItem{
					DstTarget: ips[k],
					DstPort:   port,
					SrcIp:     paramInput.SrcIp,
					Timeout:   paramInput.Timeout,
					PingType:  paramInput.PingType,
				}
				totalTaskList = append(totalTaskList, taskItem)
			}
		} else {
			taskItem := TaskItem{
				DstTarget: paramInput.DstTarget,
				DstPort:   port,
				SrcIp:     paramInput.SrcIp,
				Timeout:   paramInput.Timeout,
				PingType:  paramInput.PingType,
			}
			totalTaskList = append(totalTaskList, taskItem)
		}
	}
	return &totalTaskList
}

// GenTaskListBySingleTarget 根据单个IP参数生成总列表
func GenTaskListBySingleTarget(paramInput ParamInput) *[]TaskItem {
	// 生成任务列表
	totalTaskList := GenTaskList(paramInput, paramInput.DstPort)
	// http打流需要自动加上http头
	if paramInput.PingType == utils.PingTypeHTTP {
		for i := 0; i < len(*totalTaskList); i++ {
			(*totalTaskList)[i].DstTarget = fmt.Sprintf("http://%s:%d", (*totalTaskList)[i].DstTarget, (*totalTaskList)[i].DstPort)
		}
	}
	// 返回指针
	return totalTaskList
}

/*
GenTaskListByFile 通过文件生成总列表
每行 分为 0列、1lie、2列、多列
0列 ： 空行，直接略过
1列 ： 如果是严格模式，tcp/icmp/http的行为，如果宽松模式，tcp/icmp/http的行为
2列 ： 如果是严格模式，tcp/icmp/http的行为，如果宽松模式，tcp/icmp/http的行为
多列 ： 如果是严格模式 则报错，宽松模式则略过
*/
func GenTaskListByFile(paramInput ParamInput) *[]TaskItem {
	// 生成总任务列表
	var totalTaskList []TaskItem
	// set集合校验，防止重复
	uniqueKeySet := mapset.NewSet()
	content, err := os.ReadFile(paramInput.DstFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	// content 是一个 []byte，可以转换为 string
	fileContent := string(content)
	split := strings.Split(fileContent, "\n")
	totalLineNumber := len(split)
	for i := 0; i < totalLineNumber; i++ {
		taskList := new([]TaskItem)
		line := strings.TrimSpace(split[i])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		fieldsLength := len(fields)
		switch fieldsLength {
		case 1:
			if !paramInput.DstFileLoose {
				if paramInput.PingType == utils.PingTypeTCP || paramInput.PingType == utils.PingTypeHTTP {
					fmt.Println("文件格式不正确，行号：", i+1)
					os.Exit(0)
				}
				if !(utils.ValidateIP(fields[0]) || utils.ValidateDomain(fields[0]) || utils.ValidCIDR(fields[0])) {
					fmt.Println("文件格式不正确，行号：", i+1)
					os.Exit(0)
				}
				if paramInput.DomainA {
					if !utils.ValidateDomain(fields[0]) {
						fmt.Println("文件格式不正确，行号：", i+1)
						os.Exit(0)
					}
				}
			}
			if paramInput.DomainA {
				if !utils.ValidateDomain(fields[0]) {
					continue
				}
			}
			paramInput.DstTarget = fields[0]
			taskList = GenTaskList(paramInput, paramInput.DstPort)
		case 2:
			if !paramInput.DstFileLoose {
				if paramInput.PingType == utils.PingTypeICMP {
					fmt.Println("文件格式不正确，行号：", i+1)
					os.Exit(0)
				}
				if !(utils.ValidateIP(fields[0]) || utils.ValidateDomain(fields[0]) || utils.ValidCIDR(fields[0])) {
					fmt.Println("文件格式不正确，行号：", i+1)
					os.Exit(0)
				}
				if !govalidator.IsNumeric(fields[1]) {
					fmt.Println("文件格式不正确，行号：", i+1)
					os.Exit(0)
				}
				if paramInput.DomainA {
					if !utils.ValidateDomain(fields[0]) {
						fmt.Println("文件格式不正确，行号：", i+1)
						os.Exit(0)
					}
				}
			}
			if paramInput.DomainA {
				if !utils.ValidateDomain(fields[0]) {
					continue
				}
			}
			paramInput.DstTarget = fields[0]
			port, err1 := strconv.Atoi(fields[1])
			if err1 != nil {
				continue
			}
			taskList = GenTaskList(paramInput, port)
		default:
			if !paramInput.DstFileLoose {
				fmt.Println("文件格式不正确，行号：", i+1)
				os.Exit(0)
			}
		}
		// 汇总
		for j := 0; j < len(*taskList); j++ {
			if paramInput.PingType == utils.PingTypeHTTP {
				(*taskList)[j].DstTarget = fmt.Sprintf("http://%s:%d", (*taskList)[j].DstTarget, (*taskList)[j].DstPort)
			}
			keyId := (*taskList)[j].DstTarget
			if paramInput.PingType == utils.PingTypeTCP {
				keyId = fmt.Sprintf("%s|%d", (*taskList)[j].DstTarget, (*taskList)[j].DstPort)
			}
			if uniqueKeySet.Contains(keyId) {
				continue
			}
			totalTaskList = append(totalTaskList, (*taskList)[j])
			uniqueKeySet.Add(keyId)
		}
	}
	return &totalTaskList
}

// GenTotalTaskList 将任务复制成客户指定的打流次数
func GenTotalTaskList(taskList *[]TaskItem, paramInput ParamInput) *[]TaskItem {
	// icmp发包间隔
	icmpSendPkgInterval := utils.IcmpSendIntervalMac
	if runtime.GOOS == "linux" {
		icmpSendPkgInterval = utils.IcmpSendIntervalLinux
	}
	// 长度
	totalTaskLength := len(*taskList)
	// 生成总任务列表
	totalTaskList := []TaskItem{}
	// 持续打流，需要生成1个任务
	number := paramInput.Number
	if number == 0 {
		number = 1
	}
	k := 0
	for i := 0; i < number; i++ {
		for j := 0; j < totalTaskLength; j++ {
			taskItem := TaskItem{
				DstTarget: (*taskList)[j].DstTarget,
				DstPort:   (*taskList)[j].DstPort,
				SrcIp:     (*taskList)[j].SrcIp,
				Timeout:   (*taskList)[j].Timeout,
				PingType:  (*taskList)[j].PingType,
			}
			if (*taskList)[j].PingType == utils.PingTypeICMP {
				icmpId, icmpSeq := utils.GenIcmpIdAndSeq(k)
				taskItem.IcmpId = icmpId
				taskItem.IcmpSeq = icmpSeq
				taskItem.IcmpSendInterval = icmpSendPkgInterval
				k++
			}
			totalTaskList = append(totalTaskList, taskItem)
		}
	}
	// 返回指针
	return &totalTaskList
}

// GenConcurrencyTaskList 将总列表分配到Routine和Task里，蛇形分布
func GenConcurrencyTaskList(taskList *[]TaskItem, concurrency int) *TaskList {
	totalTaskLength := len(*taskList)
	newConcurrency := concurrency
	totalCol := 1
	if totalTaskLength < concurrency {
		newConcurrency = totalTaskLength
	} else {
		totalCol = totalTaskLength / concurrency
		if totalTaskLength%concurrency != 0 {
			totalCol++
		}
	}
	// 先初始化
	// 声明大任务列表，里面包含多个routine
	newTaskList := new(TaskList)
	var routineTaskItemList []*RoutineTaskItem
	for i := 0; i < newConcurrency; i++ {
		// 声明routine列表，里面包含多个最细粒度的任务
		routineTaskItem := new(RoutineTaskItem)
		routineTaskItem.RoutineId = i + 1
		var routineTaskList []*TaskItem
		// 把细粒度任务列表放入routine任务里
		routineTaskItem.TaskItemList = routineTaskList
		routineTaskItemList = append(routineTaskItemList, routineTaskItem)
	}
	// 把routine任务放入总的大任务列表里
	newTaskList.RoutineTaskList = routineTaskItemList
	// 生成唯一id
	node, _ := snowflake.NewNode(1) // 参数是节点编号
	id := node.Generate().String()
	newTaskList.TaskId = id
	// 蛇形分配
	k := 0
	for i := 0; i < totalCol; i++ {
		if i%2 == 0 {
			// 偶数列
			for j := 0; j < newConcurrency && k < totalTaskLength; j++ {
				item := (*taskList)[k]
				item.Id = i + 1
				newTaskList.RoutineTaskList[j].TaskItemList = append(newTaskList.RoutineTaskList[j].TaskItemList, &item)
				k++
			}
		} else {
			// 奇数列
			for j := newConcurrency - 1; j >= 0 && k < totalTaskLength; j-- {
				item := (*taskList)[k]
				item.Id = i + 1
				newTaskList.RoutineTaskList[j].TaskItemList = append(newTaskList.RoutineTaskList[j].TaskItemList, &item)
				k++
			}
		}
	}
	// 返回
	return newTaskList
}
