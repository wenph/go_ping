// Package task 本包提供任务相关的表格展示
package task

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"go_ping/utils"
	"os"
	"sort"
	"strconv"
	"time"
)

// ShowTableLoop 指定打流发包个数时，另一个goroutine需要用此来刷新表格展示
func ShowTableLoop(ctx context.Context, fr *FailRate, taskNum int, paramInput ParamInput) {
	for {
		select {
		case <-ctx.Done():
			//fmt.Println("Worker received the cancel signal and is exiting")
			showTable(fr, taskNum, paramInput)
			return // 停止执行此 goroutine
		default:
			// 正常工作的代码
			showTable(fr, taskNum, paramInput)
			// 等待一秒
			time.Sleep(time.Second)
		}
	}
}

func showTable(fr *FailRate, taskNum int, paramInput ParamInput) {
	// 字体颜色渲染
	red := color.New(color.FgRed).SprintFunc()
	// 清除终端
	print("\033[H\033[2J") // 可能不适用于所有终端
	// 一个包含string和int的二维表格
	var data [][]interface{}
	formattedTime := time.Now().Format("2006-01-02 15:04:05")
	// 加锁，读取探测结果
	fr.mutex.Lock()
	percentFirstLine := 0.0
	totalNumFirstLine := fr.FailNumber + fr.SuccessNumber
	if totalNumFirstLine != 0 {
		percentFirstLine = float64(fr.FailNumber) * 100 / float64(totalNumFirstLine)
	}
	totalLine := []string{"汇总", formattedTime, "所有实例", paramInput.PingType, strconv.Itoa(fr.FailNumber), strconv.Itoa(totalNumFirstLine), strconv.Itoa(taskNum), fmt.Sprintf("%.2f%%", percentFirstLine)}
	// 解锁
	fr.mutex.Unlock()
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "时间", "目标实例", "发包类型", "已失败数", "已发包数", "计划发包数", "失败占比"})
	table.SetFooter(totalLine)
	table.SetFooterAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetFooterColor(tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{tablewriter.FgRedColor}, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{})
	// data排序，假设我们按照每行的第4个元素（索引为3的整数）进行排序
	sort.Slice(data, func(i, j int) bool {
		// 断言值为int类型
		valueI, _ := data[i][3].(float64)
		valueJ, _ := data[j][3].(float64)
		return valueI < valueJ
	})
	for i, v := range data {
		// 转换为 string 切片
		stringSlice := []string{strconv.Itoa(i + 1), formattedTime, fmt.Sprintf("%s", v[0]), paramInput.PingType, red(fmt.Sprintf("%s", v[1])), fmt.Sprintf("%s", v[2]), strconv.Itoa(paramInput.Number), fmt.Sprintf("%.2f%%", v[3])}
		table.Append(stringSlice)
	}
	// 渲染表格
	table.Render()
}

// ShowTableForever 持续打流是永远输出表格内容
func ShowTableForever(fr *FailRate, tb *ForeverTable, instanceName string, pingType string) {
	// 颜色渲染字体
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	// 清除终端
	print("\033[H\033[2J") // 可能不适用于所有终端
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"批次", "时间", "目标实例", "发包类型", "成功数", "失败数", "目标总数", "失败占比", "变化IP数", "变化IP"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	// 统计数据
	fr.Statistic()
	// 写数据
	tb.AppendLine(fr.SuccessNumber, fr.FailNumber, fr.FromSuccessToFail, fr.FromFailToSuccess)
	// 打印table
	tb.mutex.Lock() // 加锁读数据
	for i, v := range tb.ForeverTableList {
		// 转换为 string 切片
		stringSlice := []string{v.Id, v.TimeString, instanceName, pingType, green(v.SuccessNumber), red(v.FailNumber), v.TotalNumber, v.FailPercent, v.ChangeIpNumber, v.ChangeIpSet}
		table.Append(stringSlice)
		// 日志记录最后一行
		if i+1 == len(tb.ForeverTableList) {
			utils.Log.Infoln(fmt.Sprintf("批次 %s，时间 %s，目标实例 %s，发包类型 %s，成功数 %s，失败数 %s，目标总数 %s，失败占比 %s，变化IP数 %s，变化IP %s", v.Id, v.TimeString, instanceName, pingType, v.SuccessNumber, v.FailNumber, v.TotalNumber, v.FailPercent, v.ChangeIpNumber, v.ChangeIpSet))
		}
	}
	tb.mutex.Unlock() // 解锁
	// 渲染表格
	table.Render()
	// 清空数据
	fr.Clean()
}
