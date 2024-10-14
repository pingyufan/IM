package utils

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

var (
	DB  *gorm.DB
	Red *redis.Client
)

func InitConfig() {
	//设置配置文件名和路径
	viper.SetConfigName("app")
	viper.AddConfigPath("config")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	fmt.Println("config app inited ...")
}

func InitMySQL() {
	//自定义日志模板 打印SQL语句

	//日志记录器
	newlogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second, //慢SQL阈值
			LogLevel:      logger.Info, //级别
			Colorful:      true,        //彩色
		})

	var err error
	DB, err = gorm.Open(mysql.Open(viper.GetString("mysql.dsn")), &gorm.Config{Logger: newlogger})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
}

func InitRedis() {
	Red = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.DB"),
		PoolSize:     viper.GetInt("redis.poolSize"),
		MinIdleConns: viper.GetInt("redis.minIdleConns"),
	})
}

const (
	PublishKey = "wobsocket"
)

// Publish 发布消息到Redis
func Publish(ctx context.Context, channel string, msg string) error {
	fmt.Println("Publishing message to channel:", channel, "Message:", msg)

	err := Red.Publish(ctx, channel, msg).Err()
	if err != nil {
		log.Printf("Error publishing message: %v", err)
		return fmt.Errorf("failed to publish message to channel %s: %w", channel, err)
	}
	return nil
}

// 获取Redis消息
func Subscribe(ctx context.Context, channel string) (string, error) {
	sub := Red.Subscribe(ctx, channel)
	fmt.Println("Subscribe ...", ctx)
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		log.Printf("Error receiving message: %v", err)
		return "", fmt.Errorf("failed to receive message from channel %s: %w", channel, err)
	}
	fmt.Println("Subscribe 。。。。", msg.Payload)
	return msg.Payload, err
}
