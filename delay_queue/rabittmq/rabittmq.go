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
