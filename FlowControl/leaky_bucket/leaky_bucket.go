package main

import (
	"fmt"
	"sync"
	"time"
)

type LeakyBucket struct {
	Water      float64       // 漏桶容量
	Rate       float64       // 漏桶速率
	cur        float64       // 漏桶当前剩余容量
	perReqtime time.Duration //每个请求的时间
	lasttime   time.Time     //上次请求时间
	sleepfor   time.Duration // 睡眠时间
	mut        sync.Mutex    // 互斥锁
}

// Take函数返回这次请求执行时间
/*
	漏桶已恒定流速流出桶--恒定速度处理请求

*/
func (l *LeakyBucket) Take() time.Time {
	t := time.Now()
	l.sleepfor = l.perReqtime - t.Sub(l.lasttime)
	if l.sleepfor > 0 {
		time.Sleep(l.sleepfor)
		l.lasttime = t.Add(l.sleepfor)
		l.sleepfor = 0
		l.cur--
	} else {
		l.sleepfor = 0
		l.lasttime = t
	}
	return l.lasttime

}

// 检查漏桶是否满了
func (l *LeakyBucket) Check() bool {
	l.mut.Lock()
	defer l.mut.Unlock()
	now := time.Now()
	used := now.Sub(l.lasttime).Seconds() * l.Rate //桶里最后一个请求到现在这段时间流出的量
	l.cur += used                                  //余量增加
	l.lasttime = now
	if l.cur > l.Water {
		l.cur = l.Water //不能超过容量
	}
	if l.cur >= 1 {
		//能够处理请求
		l.cur -= 1
		return true
	}
	return false
}

// new一个漏桶，规定漏桶的流速
func NewLeckyBucket(limit int) LeakyBucket {
	return LeakyBucket{
		perReqtime: time.Second / time.Duration(limit), // 每次执行请求需要的时间
		Rate:       1 / float64(limit),
		Water:      float64(limit),
		cur:        float64(limit),
	}
}
func main() {
	limiter := NewLeckyBucket(10) //流速为 10请求/s
	t := time.Now()
	count := 0
	//现在进来请求
	for i := 0; i < 1e3; i++ {
		if limiter.Check() {
			//漏桶没满
			fmt.Println(i)
			count++
		}
		time.Sleep(time.Millisecond)
	}
	fmt.Println("count:", count)             //处理了几个包
	fmt.Println(time.Now().Sub(t).Seconds()) //耗时
}
