package main

import (
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/go-redis/redis"
	"sync"
	"time"
)

func main() {
	redisCli := redis.NewClient(
		&redis.Options{
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       0,
		})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(client *redis.Client) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			t := time.Now()
			client.ZAdd("delay_queue", redis.Z{float64(t.Unix()), i})
			fmt.Printf("生产延时消息%d:%v \n", i, t.Format("2006-01-02 15:04:05"))
			time.Sleep(1 * time.Second)
		}
	}(redisCli)
	time.Sleep(5 * time.Second)
	wg.Add(1)
	go func(client *redis.Client) {
		defer wg.Done()
		tmp, _ := time.ParseDuration("-20s")
		for {
			res, err := client.ZRangeWithScores("delay_queue", 0, 1).Result()
			if err != nil {
				fmt.Println("err:", err)
				break
			}
			if len(res) == 0 {
				break
			}
			if int64(res[0].Score) < time.Now().Add(tmp).Unix() {
				fmt.Println("消费延时信息", res[0].Member, ":", time.Unix(int64(res[0].Score), 0))
				client.ZRem("delay_queue", res[0].Member)
			}
		}
	}(redisCli)
	wg.Wait()
}
