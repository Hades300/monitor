package monitor

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Sampling Gap
var GapMileSecond uint64 = 500

// Time Out Ticker
var Wait = time.Second * 20

/*
	虚拟出来的loop disk 用来挂载文件形式的文件系统, 比如 fs.img fs.iso等等 默认隐藏
*/
var AllowLoopDisk = false

/*
	false -> 仅显示合计信息
	true -> 显示合计和各个硬盘信息
*/
var AllowSingle = false

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Resource struct {
	Name string
	Data interface{}
}

type Inspector func() Resource

// return sth like "2.335" unit: %
func CpuInfo() Resource {
	var currentIdle, currnetTotal uint64
	currentIdle, currnetTotal, res := cpuSample(currentIdle, currnetTotal)
	time.Sleep(time.Millisecond * time.Duration(GapMileSecond))
	currentIdle, currnetTotal, res = cpuSample(currentIdle, currnetTotal)
	return Resource{
		Name: "CPU",
		Data: res,
	}
}

func cpuSample(prevIdle uint64, prevTotal uint64) (currentIdle uint64, currentTotal uint64, info string) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	firstLine := scanner.Text()[5:] // get rid of cpu plus 2 spaces
	file.Close()
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	split := strings.Fields(firstLine)
	idleTime, _ := strconv.ParseUint(split[3], 10, 64)
	totalTime := uint64(0)
	for _, s := range split {
		u, _ := strconv.ParseUint(s, 10, 64)
		totalTime += u
	}
	deltaIdleTime := idleTime - prevIdle
	deltaTotalTime := totalTime - prevTotal
	cpuUsage := (1.0 - float64(deltaIdleTime)/float64(deltaTotalTime)) * 100.0
	return idleTime, totalTime, fmt.Sprintf("%6.3f", cpuUsage)
}

/**
	cat /proc/net/dev then you will see the each field name
	$1 -> Interface
	$2 -> Receive Bytes
	$10 -> Transmit Bytes
**/

// Return sth like {Name:NetWork Data:map[docker0:map[Download:0 Upload:0] eth0:map[Download:0 Upload:0]...[}  uint: Bytes per second
func NetworkInfo() Resource {
	results := MustExec("cat /proc/net/dev | grep \":\" | awk '{gsub(\":\", \" \");print $1 \":\" $2 \":\"  $3 \":\" $10 \":\" $11}'")
	oldRaw := splitByEach(results, ":\n")
	oldNumbers, names := ToNumber(oldRaw)
	time.Sleep(time.Millisecond * time.Duration(GapMileSecond))
	results = MustExec("cat /proc/net/dev | grep \":\" | awk '{gsub(\":\", \" \");print $1 \":\" $2 \":\"  $3 \":\" $10 \":\" $11}'")
	newRaw := splitByEach(results, "\n:")
	newNumbers, _ := ToNumber(newRaw)
	netCardNum := len(names)
	records := map[string]interface{}{}
	for i := 0; i < netCardNum; i++ {
		records[names[i]] = map[string]interface{}{
			"Download": float64(newNumbers[0+i*2]-oldNumbers[0+i*2]) / (float64(GapMileSecond) / 1000),
			"Upload":   float64(newNumbers[1+i*2]-oldNumbers[1+i*2]) / (float64(GapMileSecond) / 1000),
		}
	}
	return Resource{
		Name: "NetWork",
		Data: records,
	}
}

//  The /proc/diskstats file displays the I/O statistics
//	of block devices. Each line contains the following 14
//	fields:
//	1 - major number
//	2 - minor mumber
//	3 - device name
//	4 - reads completed successfully
//	5 - reads merged
//	6 - sectors read
//	7 - time spent reading (ms)
//	8 - writes completed
//	9 - writes merged
//	10 - sectors written
//	11 - time spent writing (ms)
//	12 - I/Os currently in progress
//	13 - time spent doing I/Os (ms)
//	14 - weighted time spent doing I/Os (ms)

// return sth like this {"Total":1151532115,"Used":11188335...} unit: Kib
func MemInfo() Resource {
	res := MustExec("free | grep \"Mem\" | awk '{print $2}'")
	memUsageTotal, _ := strconv.ParseUint(strings.Trim(res, "\n"), 10, 64)
	res = MustExec("free | grep \"Mem\" | awk '{print $3}'")
	memUsageUsed, _ := strconv.ParseUint(strings.Trim(res, "\n"), 10, 64)
	res = MustExec("free | grep \"Mem\" | awk '{print $4}'")
	memUsageFree, _ := strconv.ParseUint(strings.Trim(res, "\n"), 10, 64)
	res = MustExec("cat /proc/meminfo | grep Buffers: | awk '{print $2}'")
	memUsageBuff, _ := strconv.ParseUint(strings.Trim(res, "\n"), 10, 64)
	res = MustExec("cat /proc/meminfo | grep Cached: | head -n1 | awk '{print $2}'")
	memUsageCache, _ := strconv.ParseUint(strings.Trim(res, "\n"), 10, 64)
	data := map[string]interface{}{
		"Total":  memUsageTotal,
		"Used":   memUsageUsed,
		"Free":   memUsageFree,
		"Buffer": memUsageBuff,
		"Cache":  memUsageCache,
	}
	return Resource{
		Name: "Memory",
		Data: data,
	}
}

/*
/proc/diskstats

Field  3 -- # of sectors read
    This is the total number of sectors read successfully.
Field  4 -- # of milliseconds spent reading
    This is the total number of milliseconds spent by all reads (as
    measured from __make_request() to end_that_request_last()).
Field  7 -- # of sectors written
    This is the total number of sectors written successfully.
Field  8 -- # of milliseconds spent writing
    This is the total number of milliseconds spent by all writes (as
    measured from __make_request() to end_that_request_last()).
*/

// return Resource {Name:"vda",Data:{"Read":"8",Write:"8"}} unit: Kib
func DiskInfo() Resource {
	// var info = make(map[string]map[string]interface{}, 10)
	info := map[string]map[string]interface{}{}
	res := MustExec("cat /proc/diskstats | awk '{print $3 \"\\n\" $6 \"\\n\" $7 \"\\n\" $10 \"\\n\" $11}'")
	before, names := ToNumber(strings.Fields(res))
	time.Sleep(time.Millisecond * time.Duration(GapMileSecond))
	res = MustExec("cat /proc/diskstats | awk '{print $3 \"\\n\" $6 \"\\n\" $7 \"\\n\" $10 \"\\n\" $11}'")
	current, _ := ToNumber(strings.Fields(res))
	diskNum := len(current) / 4
	for i := 1; i <= diskNum; i++ {
		offset := i - 1
		if AllowLoopDisk == false && strings.Contains(names[offset], "loop") {
			continue
		}
		if AllowSingle == false && endWithNumber(names[offset]) {
			continue
		}
		diskInfo := map[string]interface{}{
			"Read":  float64(current[1+offset*4]-before[1+offset*4]) / 2 / (float64(GapMileSecond) / 1000), // speed unit: 1024 bytes per second
			"Write": float64(current[2+offset*4]-before[2+offset*4]) / 2 / (float64(GapMileSecond) / 1000),
		}
		info[names[offset]] = diskInfo
	}
	return Resource{
		Name: "Disk",
		Data: info,
	}
}

func MustExec(command string) string {
	cmd := exec.Command("bash", "-c", command)
	data, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return string(data)
}

// revice a string slice with ["vda" "1" "23" "65"]
// return a uint64 slice [1 23 65] and a string slice ["vda"]
func ToNumber(origin []string) ([]uint64, []string) {
	var res []uint64
	var names []string
	for _, val := range origin {
		val = strings.Trim(val, "\n")
		number, err := strconv.ParseUint(val, 10, 64)
		if err != nil {

			names = append(names, val)
			continue
		} else {
			res = append(res, number)
		}
	}
	return res, names
}

// Any char showed in sep will be regarded as a delimiter
func splitByEach(source, sep string) []string {
	length := len(source)
	res := []string{}
	start := 0
	for i := 0; i < length; i++ {
		// ptr run into delimiter
		if strings.Contains(sep, string(source[i])) {
			res = append(res, source[start:i])
			start = i + 1
		} else if i == length-1 {
			// when ptr meet the end of string
			res = append(res, source[start:i])
		} else {
			continue
		}
	}
	return res
}

func endWithNumber(name string) bool {
	find, err := regexp.MatchString("[0-9]", name)
	if err != nil {
		panic(err)
	}
	return find
}

// Require a pipe , it will execute one test every GapMileSecond and put the result into channel
func Start(pipe chan Resource) {
	for {
		done := make(chan struct{})
		go func() {
			res := CpuInfo()
			pipe <- res
			done <- struct{}{}
		}()
		pipe <- DiskInfo()
		pipe <- MemInfo()
		<-done
	}
}

// TODO:handle abnormal exit when user close web page
func Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	pipe := make(chan Resource)
	go Start(pipe)
	t := time.NewTicker(Wait)
	for {
		select {
		case item := <-pipe:
			fmt.Printf("%+v\n", item)
			t = time.NewTicker(Wait)
			err := conn.WriteJSON(item)
			if err != nil {
				panic(err)
			}
		case <-t.C:
			panic("Time Out")
		case <-c:
			log.Fatal("OS Signal Captured --> Exiting ~")
		}
	}
}

// Return Resource Slice
func Once() []Resource {
	var tests []Resource
	wg := sync.WaitGroup{}
	inspectors := []Inspector{CpuInfo, DiskInfo, NetworkInfo, MemInfo}
	for _, item := range inspectors {
		a := item
		wg.Add(1)
		go func() {
			tests = append(tests, a())
			wg.Done()
		}()
	}
	wg.Wait()
	return tests
}
