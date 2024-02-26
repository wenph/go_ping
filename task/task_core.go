// Package task 本包提供任务相关的结构体
// 提供任务参数结构体、任务结构体、结果结构体、展示结构体，以及操作结果的方法
package task

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"go_ping/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TaskList 任务列表，存储任务的结构体
type TaskList struct {
	TaskId          string             // 大任务id，雪花算法算出的一个唯一递增id
	RoutineTaskList []*RoutineTaskItem // 协程列表，指针类型，防止多次拷贝
}

// RoutineTaskItem 每个协程的结构体
type RoutineTaskItem struct {
	RoutineId    int         // 每个协程名字，每个协程里都是从1开始增长
	TaskItemList []*TaskItem // 每个协程里需要探测的任务列表，指针类型，防止多次拷贝
}

// TaskItem 一个最细粒度的任务，直接用来调度的
type TaskItem struct {
	Id               int       // 任务id
	DstTarget        string    // 探测目表
	DstPort          int       // 探测目标端口
	SrcIp            string    // 源ip，可为空，如果客户没有指定的话
	Timeout          int       // 超时时间，有默认值
	PingType         string    // tcp/udp/icmp/http
	IcmpId           int       // icmp需要以下字段,请求ID
	IcmpSeq          int       // 序列号
	IcmpSendInterval int       // icmp每个goroutine的发包间隔，单位毫秒
	SendTime         time.Time // 发送时间
	Completed        bool      // 是否已完成
}

// =============================================

// FailRate 存储任务结果的结构体
type FailRate struct {
	mutex             sync.Mutex               //锁住以下字段
	SuccessNumber     int                      // 成功数
	FailNumber        int                      // 失败数
	ResultMap         map[string]*FailRateItem // 放每个IP的统计
	LastResultMap     map[string]*FailRateItem // 上一次的统计，用于跟本次对比
	FromSuccessToFail mapset.Set               // 输出变化的IP
	FromFailToSuccess mapset.Set               // 输出变化的IP
}

type FailRateItem struct {
	SuccessNumber int // 成功数
	FailNumber    int // 失败数
}

// NewFailRate 初始化一个空FailRate
func NewFailRate() *FailRate {
	return &FailRate{
		ResultMap:         make(map[string]*FailRateItem),
		LastResultMap:     make(map[string]*FailRateItem),
		FromSuccessToFail: mapset.NewSet(),
		FromFailToSuccess: mapset.NewSet(),
	}
}

func (c *FailRate) Increment(key string, success bool) (successNum int, failNum int) {
	c.mutex.Lock()
	if success {
		c.SuccessNumber++
	} else {
		c.FailNumber++
	}
	s := c.SuccessNumber
	f := c.FailNumber
	// 检查key是否存在
	value, exists := c.ResultMap[key]
	if exists {
		if success {
			value.SuccessNumber++
		} else {
			value.FailNumber++
		}
	} else {
		if success {
			c.ResultMap[key] = &FailRateItem{
				SuccessNumber: 1,
				FailNumber:    0,
			}
		} else {
			c.ResultMap[key] = &FailRateItem{
				SuccessNumber: 0,
				FailNumber:    1,
			}
		}
	}
	c.mutex.Unlock()
	return s, f
}

func (c *FailRate) Statistic() {
	c.mutex.Lock()
	// 和上次比较失败的、成功的
	lastSuccessSet := mapset.NewSet()
	lastFailSet := mapset.NewSet()
	nowSuccessSet := mapset.NewSet()
	nowFailSet := mapset.NewSet()
	isInitLastResultMap := false
	if len(c.LastResultMap) == 0 {
		isInitLastResultMap = true
	}
	for key, failRateItem := range c.LastResultMap {
		if failRateItem.SuccessNumber > 0 {
			lastSuccessSet.Add(key)
		} else if failRateItem.FailNumber > 0 {
			lastFailSet.Add(key)
		}
	}
	for key, failRateItem := range c.ResultMap {
		if failRateItem.SuccessNumber > 0 {
			nowSuccessSet.Add(key)
		} else if failRateItem.FailNumber > 0 {
			nowFailSet.Add(key)
		}
	}
	if !isInitLastResultMap {
		diffSuccess := nowSuccessSet.Difference(lastSuccessSet)
		diffFail := nowFailSet.Difference(lastFailSet)
		if diffFail.Cardinality() != 0 {
			utils.Log.Warnln("from success to fail:", diffFail)
			c.FromSuccessToFail = diffFail
		} else {
			c.FromSuccessToFail = mapset.NewSet()
		}
		if diffSuccess.Cardinality() != 0 {
			utils.Log.Warnln("from fail to success:", diffSuccess)
			c.FromFailToSuccess = diffSuccess
		} else {
			c.FromFailToSuccess = mapset.NewSet()
		}
	}
	c.mutex.Unlock()
}

func (c *FailRate) Clean() {
	c.mutex.Lock()
	c.SuccessNumber = 0
	c.FailNumber = 0
	for _, failRateItem := range c.LastResultMap {
		// 清空老的
		failRateItem.SuccessNumber = 0
		failRateItem.FailNumber = 0
	}
	for key, failRateItem := range c.ResultMap {
		// 存到老的里面
		value, exists := c.LastResultMap[key]
		if exists {
			value.SuccessNumber = failRateItem.SuccessNumber
			value.FailNumber = failRateItem.FailNumber
		} else {
			c.LastResultMap[key] = &FailRateItem{
				SuccessNumber: failRateItem.SuccessNumber,
				FailNumber:    failRateItem.FailNumber,
			}
		}
		// 清零
		failRateItem.SuccessNumber = 0
		failRateItem.FailNumber = 0
	}
	c.mutex.Unlock()
}

// ==================================================

// ParamInput 存储用户命令行输入的参数
type ParamInput struct {
	DstTarget    string
	DstPort      int
	DstFile      string
	DstFileLoose bool
	SrcIp        string
	PingType     string
	Timeout      int
	Concurrency  int
	Number       int
	ShowMode     string
	LogLevel     string
	DomainA      bool
}

// ==================================================

// ForeverTable 持续打流的展示表
type ForeverTable struct {
	mutex            sync.Mutex //锁住以下字段
	ForeverTableList []ForeverTableLine
}

// ForeverTableLine 持续打流的展示表的行
type ForeverTableLine struct {
	Id             string
	TimeString     string
	SuccessNumber  string
	FailNumber     string
	TotalNumber    string
	FailPercent    string
	ChangeIpNumber string // 连通性变化的IP个数
	ChangeIpSet    string // 变化的IP
}

// NewForeverTable 初始化一个空ForeverTable
func NewForeverTable() *ForeverTable {
	return &ForeverTable{ForeverTableList: []ForeverTableLine{}}
}

func (c *ForeverTable) AppendLine(successNum int, failNum int, FromSuccessToFail mapset.Set, FromFailToSuccess mapset.Set) {
	showIpLen := 1
	ChangeIPSet := FromSuccessToFail.Union(FromFailToSuccess)
	slice := ChangeIPSet.ToSlice()
	// 创建一个字符串切片用于存储转换后的字符串元素
	stringSlice := make([]string, 0, len(slice))
	// 遍历接口切片，将元素转换为字符串，并添加到字符串切片中
	for _, elem := range slice {
		str, ok := elem.(string)
		if !ok {
			continue
		}
		stringSlice = append(stringSlice, str)
	}
	sort.Strings(stringSlice)
	changeIpList := []string{}
	hasMore := ""
	if len(stringSlice) > showIpLen {
		changeIpList = stringSlice[0:showIpLen]
		hasMore = ",..."
	} else {
		changeIpList = stringSlice
	}
	c.mutex.Lock()
	id := 1
	foreverTableListLength := len(c.ForeverTableList)
	if foreverTableListLength != 0 {
		newestId, _ := strconv.Atoi(c.ForeverTableList[foreverTableListLength-1].Id)
		id = newestId + 1
	}
	failPercent := 0.0
	totalNum := successNum + failNum
	if totalNum != 0 {
		failPercent = float64(failNum) * 100 / float64(totalNum)
	}
	foreverTableLine := ForeverTableLine{
		Id:             strconv.Itoa(id),
		TimeString:     time.Now().Format("2006-01-02 15:04:05"),
		SuccessNumber:  strconv.Itoa(successNum),
		FailNumber:     strconv.Itoa(failNum),
		TotalNumber:    strconv.Itoa(totalNum),
		FailPercent:    fmt.Sprintf("%.2f%%", failPercent),
		ChangeIpNumber: strconv.Itoa(ChangeIPSet.Cardinality()),
		ChangeIpSet:    strings.Join(changeIpList, ",") + hasMore,
	}
	n := 20
	newForeverTableList := c.ForeverTableList
	if foreverTableListLength >= n {
		newForeverTableList = c.ForeverTableList[foreverTableListLength-n+1 : foreverTableListLength]
	}
	// 在最前端插入新的Item实例到切片中
	newForeverTableList = append(newForeverTableList, foreverTableLine)
	c.ForeverTableList = newForeverTableList
	c.mutex.Unlock()
}

// ==================================================

// FileTaskItemNumber 文件里的行数
type FileTaskItemNumber struct {
	//FileName        string // 文件名称
	//TotalLineNumber int    // 文件总行数
	//ValidLineNumber int    // 有效行数
	//TaskLineNumber  int    // 构成任务的行数（比如用户指定有些探测不通的目标，不形成任务）
	TaskNumber int // 构成的任务数，比如网段：一行就可以构成很多任务，目前就使用了这个字段
}
