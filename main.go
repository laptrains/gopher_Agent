package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "GopherAI/docs"
)

func StartServer(addr string, port int) error {
	r := router.InitRouter()
	//服务器静态资源路径映射关系，这里目前不需要
	// r.Static(config.GetConfig().HttpFilePath, config.GetConfig().MusicFilePath)
	return r.Run(fmt.Sprintf("%s:%d", addr, port))
}

// 从数据库加载消息并初始化 AIHelperManager
func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	// 从数据库读取所有消息
	msgs, err := message.GetAllMessages()
	if err != nil {
		return err
	}
	// 遍历数据库消息
	for i := range msgs {
		m := &msgs[i]
		//默认openai模型
		modelType := "1"
		config := make(map[string]interface{})

		// 创建对应的 AIHelper
		helper, err := manager.GetOrCreateAIHelper(m.UserName, m.SessionID, modelType, config)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		log.Println("readDataFromDB init:  ", helper.SessionID)
		// 添加消息到内存中(不开启存储功能)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	log.Println("AIHelperManager init success ")
	return nil
}

// @title GopherAI API
// @version 1.0
// @description GopherAI AI 聊天后端接口文档
// @host localhost:9090
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	//初始化mysql
	if err := mysql.InitMysql(); err != nil {
		log.Println("InitMysql error , " + err.Error())
		return
	}
	//初始化AIHelperManager
	if err := readDataFromDB(); err != nil {
		log.Printf("readDataFromDB error: %v", err)
	}

	//初始化redis
	redis.Init()
	log.Println("redis init success  ")

	// 初始化 RabbitMQ，失败时重试，仍失败则退出
	var mqErr error
	for i := 1; i <= 3; i++ {
		if mqErr = rabbitmq.InitRabbitMQ(); mqErr == nil {
			break
		}
		log.Printf("rabbitmq init failed (attempt %d/3): %v", i, mqErr)
		time.Sleep(2 * time.Second)
	}
	if mqErr != nil {
		log.Fatalf("rabbitmq init failed after retries: %v", mqErr)
	}
	log.Println("rabbitmq init success  ")

	// 优雅关闭：监听退出信号，释放 RabbitMQ 连接后再退出
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("received shutdown signal, closing rabbitmq...")
		rabbitmq.DestroyRabbitMQ()
		os.Exit(0)
	}()

	err := StartServer(host, port) // 启动 HTTP 服务
	if err != nil {
		panic(err)
	}
}
