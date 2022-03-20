# 延迟队列

生产场景中延迟队列的应用很普遍

* 下单后20分钟未支付取消
* 下单后1分钟后发送通知

例如这些需要延迟一段时间的操作，都称作延时任务，可以使用延时队列实现

## 实现方式：

1、数据库轮询

创建一个线程去轮询订单表，查看订单时间是否符合要求，然后进行表的更新操作(如超时则修改订单为已取消等)

问题：数据量大的时候，轮询数据库太慢，数据库压力大

2、使用redis做延时队列

redis 的有序集合zset可以用来做延时队列

​	 ZADD key score1 member1 [score2 member2]

​	向有序集合中的key添加成员member1，排序的依据的score1

​	ZRANGE key start stop [WITHSCORES] 

​	通过索引区间返回有序集合指定区间内的成员,按照score从小到大

​	ZREVRANGE key start stop [WITHSCORES] 

​	通过索引区间返回有序集合指定区间内的成员,按照score从大到小

所以把商品的时间戳作为score放入redis，每次处理最小的时间戳，即最早的商品

Example:

​	brew services start redid --启动redis服务

​	redis-cli --启动redis 客户端

​	flushdb --清空数据库

```go
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
```

3、利用rabbitmq的死信队列做延时

![img](https://upload-images.jianshu.io/upload_images/5015984-367dd717d89ae5db.png?imageMogr2/auto-orient/strip|imageView2/2/format/webp)

![image-20220320153424458](image/image-20220320153424458.png)

go get github.com/streadway/amqp

先实现一个基本的生产消费队列：

https://blog.csdn.net/bestmy/article/details/84304964

```go
package main

import (
   "fmt"
   "github.com/streadway/amqp"
   "log"
   "sync"
)

const (
   rmqUrl = "amqp://guest:guest@127.0.0.1:5672/"
)

type RabittMq struct {
   Conn      *amqp.Connection
   Channel   *amqp.Channel
   QueueName string
   Exchang   string
   Key       string
   Mqurl     string
}

//创建一个rabittmq 实例
func NewRabitMq(queneName, exchange, key string) *RabittMq {
   rmq := &RabittMq{
      QueueName: queneName,
      Exchang:   exchange,
      Key:       key,
      Mqurl:     rmqUrl,
   }
   var err error
   //连接消息队列mq
   rmq.Conn, err = amqp.Dial(rmqUrl)
   if err != nil {
      panic(fmt.Sprintf("Dial error:%v", err))
   }
   //channel()开启一个处理消息的信道
   rmq.Channel, err = rmq.Conn.Channel()
   if err != nil {
      panic(fmt.Sprintf("open channel error:%v", err))
   }
   return rmq

}

// 关闭实例
func DestroyMq(rmq *RabittMq) error {
   err := rmq.Channel.Close()
   err = rmq.Conn.Close()
   return err
}

// 简单的例子
func NewSimpleMq(queueName string) *RabittMq {
   return NewRabitMq(queueName, "", "")
}

//生成者
func (mq *RabittMq) Publish(message string) {
   //申请一个queue
   _, err := mq.Channel.QueueDeclare(
      //队列名
      mq.QueueName,
      //持久化
      false,
      //自动删除
      false,
      //p排他性
      false,
      //阻塞
      false,
      nil,
   )
   if err != nil {
      panic(fmt.Sprintf("queue declare error(public):%v", err))
   }
   //发消息到队列中
   mq.Channel.Publish(
      mq.Exchang,
      mq.QueueName,
      false,
      false,
      amqp.Publishing{
         ContentType: "text/plain",
         Body:        []byte(message),
      },
   )
}

//消费者
func (mq *RabittMq) Comsume() {
   //申请一个queue
   _, err := mq.Channel.QueueDeclare(
      //队列名
      mq.QueueName,
      //持久化
      false,
      //自动删除
      false,
      //p排他性
      false,
      //阻塞
      false,
      nil,
   )
   if err != nil {
      panic(fmt.Sprintf("queue declare error(consume):%v", err))
   }
   //发消息到队列中
   msgs, err := mq.Channel.Consume(
      mq.QueueName,
      "",
      true,
      false,
      false,
      false,
      nil,
   )
   if err != nil {
      fmt.Println(err)
   }

   // 启用携程处理消息
   wg := sync.WaitGroup{}
   wg.Add(1)
   go func() {
      defer wg.Done()
      for d := range msgs {
         // 实现我们要实现的逻辑函数
         log.Printf("Received a message: %s", d.Body)
         fmt.Println(d.Body)
      }
   }()
   log.Printf("[*] Waiting for message, To exit press CTRL+C")
   wg.Wait()
}
func main() {
   wg := sync.WaitGroup{}
   wg.Add(2)
   go func() {
      defer wg.Done()
      rabbitmq := NewSimpleMq("imoocSimple")
      rabbitmq.Publish("Hello goFrame!")
      fmt.Println("发送成功")
   }()
   go func() {
      defer wg.Done()
      rabbitmq := NewSimpleMq("imoocSimple")

      rabbitmq.Comsume()
   }()
   wg.Wait()
}
```

上面是使用rabittmq的默认exchange进行分发，下面定义一个exchange来转发

```go
package main

import (
   "fmt"
   "github.com/streadway/amqp"
   "log"
   "sync"
)

const (
   rmqUrl = "amqp://guest:guest@127.0.0.1:5672/"
)

type RabittMq struct {
   Conn      *amqp.Connection
   Channel   *amqp.Channel
   QueueName string
   Exchang   string
   Key       string
   Mqurl     string
}

//创建一个rabittmq 实例
func NewRabitMq(queneName, exchange, key string) *RabittMq {
   rmq := &RabittMq{
      QueueName: queneName,
      Exchang:   exchange,
      Key:       key,
      Mqurl:     rmqUrl,
   }
   var err error
   //连接消息队列mq
   rmq.Conn, err = amqp.Dial(rmqUrl)
   if err != nil {
      panic(fmt.Sprintf("Dial error:%v", err))
   }
   //channel()开启一个处理消息的信道
   rmq.Channel, err = rmq.Conn.Channel()
   if err != nil {
      panic(fmt.Sprintf("open channel error:%v", err))
   }
   return rmq

}

// 关闭实例
func DestroyMq(rmq *RabittMq) error {
   err := rmq.Channel.Close()
   err = rmq.Conn.Close()
   return err
}

// 简单的例子
func NewSimpleMq(queueName string) *RabittMq {
   return NewRabitMq(queueName, "", "")
}

//生成者
func (mq *RabittMq) Publish(message string) {
   //申请一个exchange
   err := mq.Channel.ExchangeDeclare(
      "exchange1",
      "direct",
      false,
      false,
      false,
      false, nil)
   if err != nil {
      panic(fmt.Sprintf("exchange declare error :%v", err))
   }
   //申请一个queue
   _, err = mq.Channel.QueueDeclare(
      //队列名
      mq.QueueName,
      //持久化
      false,
      //自动删除
      false,
      //p排他性
      false,
      //阻塞
      false,
      nil,
   )
   if err != nil {
      panic(fmt.Sprintf("queue declare error(public):%v", err))
   }
   //把queue和exchange绑定，binding key 是 testkey1
   mq.Channel.QueueBind(mq.QueueName, "testkey1", "exchange1", false, nil)
   //发消息到队列中,使用exchange1，routinekey是testkey1，因为是direc使用使用全匹配
   mq.Channel.Publish(
      "exchange1",
      "testkey1",
      false,
      false,
      amqp.Publishing{
         ContentType: "text/plain",
         Body:        []byte(message),
      },
   )
}

//消费者
func (mq *RabittMq) Comsume() {
   //申请一个queue
   _, err := mq.Channel.QueueDeclare(
      //队列名
      mq.QueueName,
      //持久化
      false,
      //自动删除
      false,
      //p排他性
      false,
      //阻塞
      false,
      nil,
   )
   if err != nil {
      panic(fmt.Sprintf("queue declare error(consume):%v", err))
   }
   //到那个queue收消息
   msgs, err := mq.Channel.Consume(
      mq.QueueName,
      "any",
      true,
      false,
      false,
      false,
      nil,
   )
   if err != nil {
      fmt.Println(err)
   }

   // 启用携程处理消息
   wg := sync.WaitGroup{}
   wg.Add(1)
   go func() {
      defer wg.Done()
      for d := range msgs {
         // 实现我们要实现的逻辑函数
         log.Printf("Received a message: %s", d.Body)
         fmt.Println(d.Body)
      }
   }()
   log.Printf("[*] Waiting for message, To exit press CTRL+C")
   wg.Wait()
}
func main() {
   wg := sync.WaitGroup{}
   wg.Add(2)
   go func() {
      defer wg.Done()

      rabbitmq := NewSimpleMq("imoocSimple")
      rabbitmq.Publish("Hello goFrame!")
      fmt.Println("发送成功")
      DestroyMq(rabbitmq)
   }()
   go func() {
      defer wg.Done()
      rabbitmq := NewSimpleMq("imoocSimple")

      rabbitmq.Comsume()
      DestroyMq(rabbitmq)
   }()
   wg.Wait()
}
```

而延时队列利用rabittmq的死信队列实现

![image-20220320153405123](image/image-20220320153405123.png)

```go
package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"sync"
)

const (
	rmqUrl = "amqp://guest:guest@127.0.0.1:5672/"
)

type RabittMq struct {
	Conn               *amqp.Connection
	Channel            *amqp.Channel
	DeadQueueName      string
	NormalQueueName    string
	DeadExchangeName   string
	DeadKey            string
	NormalExchangeName string
	NormalKey          string
	Mqurl              string
}

//创建一个rabittmq 实例
func NewRabitMq(deadqueuename, deadexchange, deadkey, normalqueneName, normalexchange, normalkey string) *RabittMq {
	rmq := &RabittMq{
		DeadQueueName:      deadqueuename,
		DeadExchangeName:   deadexchange,
		DeadKey:            deadkey,
		NormalQueueName:    normalqueneName,
		NormalExchangeName: normalexchange,
		NormalKey:          normalkey,
		Mqurl:              rmqUrl,
	}
	var err error
	//连接消息队列mq
	rmq.Conn, err = amqp.Dial(rmqUrl)
	if err != nil {
		panic(fmt.Sprintf("Dial error:%v", err))
	}
	//channel()开启一个处理消息的信道
	rmq.Channel, err = rmq.Conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("open channel error:%v", err))
	}
	return rmq

}

// 关闭实例
func DestroyMq(rmq *RabittMq) error {
	err := rmq.Channel.Close()
	err = rmq.Conn.Close()
	return err
}

// 简单的例子
func NewSimpleMq(deadqueueName, deadexchange, deadkey, normalqueuename, normalexchange, normalkey string) *RabittMq {
	return NewRabitMq(deadqueueName, deadexchange, deadkey, normalqueuename, normalexchange, normalkey)
}

//生成者
func (mq *RabittMq) Publish(message string) {
	//申请一个deadexchange
	err := mq.Channel.ExchangeDeclare(
		mq.DeadExchangeName,
		"direct",
		false,
		false,
		false,
		false, nil)
	if err != nil {
		panic(fmt.Sprintf("exchange declare error :%v", err))
	}
	//申请一个noramlxchange
	err = mq.Channel.ExchangeDeclare(
		mq.NormalExchangeName,
		"direct",
		false,
		false,
		false,
		false, nil)
	if err != nil {
		panic(fmt.Sprintf("exchange declare error :%v", err))
	}
	//申请一个queue
	_, err = mq.Channel.QueueDeclare(
		//队列名
		mq.DeadQueueName,
		//持久化
		false,
		//自动删除
		false,
		//p排他性
		false,
		//阻塞
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("queue declare error(public):%v", err))
	}
	//申请一个normalqueue; 到期后放入私信队列
	_, err = mq.Channel.QueueDeclare(
		//队列名
		mq.NormalQueueName,
		//持久化
		false,
		//自动删除
		false,
		//p排他性
		false,
		//阻塞
		false,
		amqp.Table{
			"x-message-ttl":             5000,                // 消息过期时间,毫秒
			"x-dead-letter-exchange":    mq.DeadExchangeName, // 指定死信交换机
			"x-dead-letter-routing-key": mq.DeadKey,          // 指定死信routing-key
		},
	)
	if err != nil {
		panic(fmt.Sprintf("queue declare error(public):%v", err))
	}
	//把queue和exchange绑定，设置binding key
	mq.Channel.QueueBind(mq.NormalQueueName, mq.NormalKey, mq.NormalExchangeName, false, nil)

	mq.Channel.QueueBind(mq.DeadQueueName, mq.DeadKey, mq.DeadExchangeName, false, nil)
	//发消息到队列中,使用exchange1，routinekey是testkey1，因为是direc使用使用全匹配
	mq.Channel.Publish(
		mq.NormalExchangeName,
		mq.NormalKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)
}

//消费者
func (mq *RabittMq) Comsume() {
	//申请一个queue
	_, err := mq.Channel.QueueDeclare(
		//队列名
		mq.DeadQueueName,
		//持久化
		false,
		//自动删除
		false,
		//p排他性
		false,
		//阻塞
		false,
		nil,
	)
	if err != nil {
		panic(fmt.Sprintf("queue declare error(consume):%v", err))
	}
	//收消息
	msgs, err := mq.Channel.Consume(
		mq.DeadQueueName,
		"any",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Println(err)
	}

	// 启用携程处理消息
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for d := range msgs {
			// 实现我们要实现的逻辑函数
			log.Printf("Received a message from dead: %s", d.Body)
			fmt.Println(d.Body)
		}
	}()
	log.Printf("[*] Waiting for message, To exit press CTRL+C")
	wg.Wait()
}
func main() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()

		rabbitmq := NewSimpleMq("deadqueue", "deadexchange", "deadkey", "normalqueue", "normalexchange", "normalkey")
		rabbitmq.Publish("Hello goFrame!")
		fmt.Println("发送成功")
		DestroyMq(rabbitmq)
	}()
	go func() {
		defer wg.Done()
		rabbitmq := NewSimpleMq("deadqueue", "deadexchange", "deadkey", "normalqueue", "normalexchange", "normalkey")

		rabbitmq.Comsume()
		DestroyMq(rabbitmq)
	}()
	wg.Wait()
}

```