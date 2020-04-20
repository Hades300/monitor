
# Monitor

Simple system resource monitor

## Feature 


* [x] Built-in Websocket Handler

* [x] Disk \ CPU \ Memory Monitoring

* [x] NetWork Monitoring

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
{Name:Memory Data:map[Buffer:301268 Cache:1059852 Free:223552 Total:4039436 Used:2009616]}
{Name:CPU Data: 6.122}
{Name:NetWork Data:map[docker0:map[Download:0 Upload:0] eth0:map[Download:0 Upload:0] lo:map[Download:0 Upload:0] veth4e46e22:map[Download:0 Upload:0] veth9f0bfc5:map[Download:120 Upload:2] vethbdc60a0:map[Download:0 Upload:0] vethfdb15d5:map[Download:284 Upload:2] zt2lrrddgb:map[Download:0 Upload:0]]}
{Name:Disk Data:map[vda:map[Read:0 Write:0]]}
```
Default unit: 

- <b> Kib </b> for Disk and Memory
- <b>%</b>  for cpu usage
- <b>bytes per second</b> for Network

## ScreenShot

![My VPS Resource Info](http://q8ptr9gz2.bkt.clouddn.com/monitor.gif)

Simple and Rough Self-used Frontend build with vue Contact at by1018987488@gmail.com if you really need that ...