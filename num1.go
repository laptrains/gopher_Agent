package main

import (
	"fmt"
	"sync"
)

// Singleton 单例结构体
type Singleton struct {
	data string
}

var (
	instance *Singleton
	once     sync.Once
)

// GetInstance 获取单例实例 (线程安全的懒汉式)
func GetInstance() *Singleton {
	// 使用 sync.Once 确保初始化逻辑在整个程序生命周期中只执行一次，无论多少协程并发调用
	once.Do(func() {
		instance = &Singleton{
			data: "我是单例实例",
		}
	})
	return instance
}

// SetData 设置数据
func (s *Singleton) SetData(data string) {
	s.data = data
}

// GetData 获取数据
func (s *Singleton) GetData() string {
	return s.data
}

// 另一种实现方式：饿汉式单例
// 在包加载阶段就完成初始化，天然线程安全，但可能提前占用资源
var eagerInstance = &Singleton{
	data: "我是饿汉式单例实例",
}

// GetEagerInstance 获取饿汉式单例实例
func GetEagerInstance() *Singleton {
	return eagerInstance
}

func main() {
	fmt.Println("=== 单例模式示例 ===")

	// 懒汉式单例测试
	fmt.Println("\n--- 懒汉式单例测试 ---")
	s1 := GetInstance()
	s2 := GetInstance()

	// 打印实例的内存地址，验证是否为同一个对象
	fmt.Printf("s1 地址: %p\n", s1)
	fmt.Printf("s2 地址: %p\n", s2)
	fmt.Printf("s1 == s2: %t\n", s1 == s2)

	fmt.Printf("s1 数据: %s\n", s1.GetData())
	s1.SetData("修改后的数据")
	// 验证 s1 和 s2 共享同一份数据
	fmt.Printf("s2 数据: %s\n", s2.GetData())

	// 饿汉式单例测试
	fmt.Println("\n--- 饿汉式单例测试 ---")
	e1 := GetEagerInstance()
	e2 := GetEagerInstance()

	fmt.Printf("e1 地址: %p\n", e1)
	fmt.Printf("e2 地址: %p\n", e2)
	fmt.Printf("e1 == e2: %t\n", e1 == e2)
	fmt.Printf("e1 数据: %s\n", e1.GetData())
}
