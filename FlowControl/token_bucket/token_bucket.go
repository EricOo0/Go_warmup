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
		fmt.Println(i)
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
