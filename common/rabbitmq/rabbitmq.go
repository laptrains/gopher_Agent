package rabbitmq

import (
	"GopherAI/config"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

// 全局connection对象
// 所有RabbitMQ都会复用该对象
var conn *amqp.Connection

// 初始化connection
func initConn() error {
	c := config.GetConfig()
	mqUrl := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		c.RabbitmqUsername, c.RabbitmqPassword, c.RabbitmqHost, c.RabbitmqPort, c.RabbitmqVhost,
	)
	// 不打印完整 URL，避免密码明文泄露到日志
	log.Printf("connecting to RabbitMQ: %s:%d/%s", c.RabbitmqHost, c.RabbitmqPort, c.RabbitmqVhost)
	var err error
	conn, err = amqp.Dial(mqUrl)
	if err != nil {
		return fmt.Errorf("RabbitMQ connection failed: %w", err)
	}
	return nil
}

// RabbitMQ RabbitMQ结构体
type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	Exchange string
	Key      string
}

// NewRabbitMQ 创建RabbitMQ对象
func NewRabbitMQ(exchange string, key string) *RabbitMQ {
	return &RabbitMQ{Exchange: exchange, Key: key}
}

// Destroy 断开 channel 和 connection
func (r *RabbitMQ) Destroy() {
	_ = r.channel.Close()
	_ = r.conn.Close()
}

// NewWorkRabbitMQ 创建Work模式的RabbitMQ实例
func NewWorkRabbitMQ(queue string) (*RabbitMQ, error) {
	// new rabbitmq
	rabbitmq := NewRabbitMQ("", queue)

	// get connection
	if conn == nil {
		if err := initConn(); err != nil {
			return nil, err
		}
	}
	rabbitmq.conn = conn

	// get channel
	var err error
	rabbitmq.channel, err = rabbitmq.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel failed: %w", err)
	}

	return rabbitmq, nil
}

// Publish 发送消息
func (r *RabbitMQ) Publish(message []byte) error {
	// 创建队列（不存在时）
	// 使用默认交换机的情况下，queue即为key
	_, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 调用 channel 发送消息到队列
	return r.channel.Publish(r.Exchange, r.Key, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
}

// Consume 消费者
// handle: 消息的消费业务函数，用于消费消息
func (r *RabbitMQ) Consume(handle func(msg *amqp.Delivery) error) {
	// 创建队列
	q, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 接收消息
	// autoAck=false：关闭自动确认，改为处理成功后手动 Ack，失败时 Nack 重新入队重试
	msgs, err := r.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 处理消息
	for msg := range msgs {
		if err := handle(&msg); err != nil {
			fmt.Println("consume error: ", err.Error())
			retryCount := getRetryCount(msg)
			if retryCount >= maxRetryCount {
				// 超过最大重试次数，丢弃消息（避免"毒消息"无限重试死循环）
				fmt.Printf("message retried %d times, drop it: %s\n", retryCount, string(msg.Body))
				_ = msg.Nack(false, false)
				continue
			}
			// 未超限：重新发布一条带递增重试计数的消息到队尾，实现"延迟重试"
			msg.Headers = setRetryCount(msg, retryCount+1)
			publishing := amqp.Publishing{
				Headers:         msg.Headers,
				ContentType:     msg.ContentType,
				ContentEncoding: msg.ContentEncoding,
				DeliveryMode:    msg.DeliveryMode,
				Priority:        msg.Priority,
				CorrelationId:   msg.CorrelationId,
				ReplyTo:         msg.ReplyTo,
				Expiration:      msg.Expiration,
				MessageId:       msg.MessageId,
				Timestamp:       msg.Timestamp,
				Type:            msg.Type,
				UserId:          msg.UserId,
				AppId:           msg.AppId,
				Body:            msg.Body,
			}
			if pubErr := r.channel.Publish(r.Exchange, r.Key, false, false, publishing); pubErr != nil {
				// 重新发布失败，Nack 并重新入队保留原消息
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		} else {
			_ = msg.Ack(false)
		}
	}
}

const (
	retryHeader    = "x-retry-count"
	maxRetryCount  = 3
)

// getRetryCount 从消息 header 中读取已重试次数
func getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}
	v, ok := msg.Headers[retryHeader]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// setRetryCount 更新消息 header 中的重试次数
func setRetryCount(msg amqp.Delivery, count int) amqp.Table {
	if msg.Headers == nil {
		msg.Headers = amqp.Table{}
	}
	msg.Headers[retryHeader] = int32(count)
	return msg.Headers
}
