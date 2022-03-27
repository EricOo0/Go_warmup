# 常用限流方式

在高并发的场景下，一般有三种保护系统的方式，分别是`缓存,限流，降级`；

## 缓存

在高并发的场景下，如果每一次请求都要直接请求DB查询数据，那数据库肯定会超负荷被挤爆。而对于绝大多数的系统，都是读操作远大于写操作，所以可以在数据库前面增加一层缓存，减少对数据库的直接访问频率，保护系统。

常用的像redis，消息队列都可以用来作为缓存

但也会出现像缓存击穿等一系列问题



## 降级

服务降级是指在服务器压力骤增的情况下，对一些服务提供的能力进行适当的降低，以保证核心服务不受影响。

降级往往会指定不同的级别，面临不同的异常等级执行不同的处理。根据服务方式：可以拒接服务，可以延迟服务，也有时候可以随机服务。根据服务范围：可以砍掉某个功能，也可以砍掉某些模块。总之服务降级需要根据不同的业务需求采用不同的降级策略。主要的目的就是服务虽然有损但是总比没有好。



## 限流

限流，顾名思义，就是限制高并发的流量了，常见限流方法有：

1、计数器：通过控制最大并发数来进行限流

​	在一个时间窗口内，限制访问的次数。实现方式：在开始的时候设置一个计数器，每来一个请求，计数器+1，如果计数器大于阈值切与第一个请求在一个时间窗口内，则不允许访问，反之重置计数器。

​	也可以利用redis设置key过期时间来做

​	问题：单纯的计数器容易存在临界阈值问题

![image-20220326113312848](/Users/weizhifeng/Library/Application Support/typora-user-images/image-20220326113312848.png)

所以可以使用滑动窗口优化：

利用redis做：

```
public Response limitFlow() {
    Long  currentTime = new Date().getTime();
    if (redisTemplate.hasKey("limit")) {
        Integer count = redisTemplate.opsForZset().rangeByScore("limit", currentTime - intervalTime, currentTime).size();
        if (count != null && count > 5) {
            return Response.ok("每分钟最多只能访问 5 次！");
        }
    }
    redisTemplate.opsForZSet().add("limit", UUID.randomUUID().toString(), currentTime);
    return Response.ok("访问成功");
}
```

2、 令牌桶算法

令牌桶算法是比较常见的限流算法之一，大概描述如下：
1）、所有的请求在处理之前都需要拿到一个可用的令牌才会被处理；
2）、根据限流大小，设置按照一定的速率往桶里添加令牌；
3）、桶设置最大的放置令牌限制，当桶满时、新添加的令牌就被丢弃或者拒绝；
4）、请求达到后首先要获取令牌桶中的令牌，拿着令牌才可以进行其他的业务逻辑，处理完业务逻辑之后，将令牌直接删除；

有令牌才能处理，没令牌就等着

```go
package main

import (
	"fmt"
	"time"
)

type TokenBucket struct {
	Bucket chan struct{}
}

//单例模式

var RefBucket TokenBucket

func InjectBucket(_Bucket chan struct{}) {
	RefBucket.Bucket = _Bucket
}
func GetBucket() TokenBucket {
	return RefBucket
}

func (t *TokenBucket) Init(limit int, rate int) {
	bucket := GetBucket()
	// 初始化令牌桶
	for i := 0; i < limit; i++ {
		bucket.Bucket <- struct{}{}
	}
	//每秒新增rate个令牌
	go func() {
		tic := time.NewTicker(1 * time.Second)
		for {
			fmt.Println("1second add token ")
			for i := 0; i < rate; i++ {
				bucket.Bucket <- struct{}{}
			}
			<-tic.C
		}
	}()
}
func Comsume(t TokenBucket, rate int) {
	for i := 0; i < rate; i++ {
		select {
		case <-t.Bucket:
			fmt.Println("consume sucess")
		default:
			fmt.Println("failed,no token")
			time.Sleep(100 * time.Millisecond)
		}

	}
}
func main() {
	buketLimit := 10
	ch := make(chan struct{}, buketLimit)

	InjectBucket(ch)
	bucket := GetBucket()
	bucket.Init(buketLimit, 10)

	// 请求来了,每秒20个请求
	for {
		Comsume(bucket, 20)
	}

}

```

3、漏桶算法

类似令牌桶算法，但是漏桶装的是请求，漏桶满了，新到的请求直接拒绝；

可以用消息队列实现，服务器已恒定速率处理请求，请求先放入消息队列等待处理

伪代码：

```go
// 定义漏桶结构
type leakyBucket struct {
  timestamp time.Time // 当前注水时间戳 （当前请求时间戳）
  capacity float64  // 桶的容量（接受缓存的请求总量）
  rate  float64// 水流出的速度（处理请求速度）
  water float64 // 当前水量（当前累计请求数）
}

// 判断是否加水（是否处理请求）
func addWater(bucket leakyBucket) bool {
  now := time.Now()
  // 先执行漏水，计算剩余水量
  leftWater := math.Max(0,bucket.water - now.Sub(bucket.timestamp).Seconds()*bucket.rate)
  bucket.timestamp = now
  if leftWater + 1 < bucket.water {
    // 尝试加水，此时水桶未满
    bucket.water = leftWater +1
    return true
  }else {
    // 水满了，拒绝加水
    return false
  }
}
```

简单的实现：

```go
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
```

