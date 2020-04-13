
# Monitor

Simple system resource monitor

## Feature 


* [x] Built-in Websocket Handler

* [x] Disk \ CPU \ Memory Monitoring

* [ ] NetWork Monitoring

* [ ] Threshold value setting and warning info

## Get Start

```go
package main

import (
    "github.com/hades300/monitor"
    "net/http"
)

func main(){
    http.HandleFunc("/monitor",monitor.Handle)
    err:=http.ListenAndServe(":8080",nil)
    if err!=nil{panic(err)}
}
```

### Data Sample
```
{Name:Memory Data:map[Buffer:237768 Cache:578076 Free:1020176 Total:4039436 Used:2039544]}
{Name:CPU Data: 1.020}
{Name:Disk Data:map[vda:map[Read:0 Write:0]]}
```
Default unit: 

- <b> Kib </b> for Disk and Memory
- <b>%</b>  for cpu usage

## ScreenShot

![My VPS Resource Info](http://q8ptr9gz2.bkt.clouddn.com/monitor.gif)

Simple and Rough Self-used Frontend build with vue Contact at by1018987488@gmail.com if you really need that ...