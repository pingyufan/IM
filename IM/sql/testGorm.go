package main

import (
	"IM/models"
	"IM/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(mysql.Open("root:030428@tcp(localhost:3306)/ginchat?charset=utf8mb4&parseTime=True&loc=UTC"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	utils.InitConfig()
	utils.InitMySQL()
	utils.InitRedis()

	// 迁移 schema
	db.AutoMigrate(&models.Community{})
	//if err := db.AutoMigrate(&models.UserBasic{}); err != nil {
	//	panic("failed to migrate database: " + err.Error())
	//}
	//db.AutoMigrate(&models.Message{})
	//db.AutoMigrate(&models.GroupBasic{})
	//db.AutoMigrate(&models.Contact{})

	// Create
	//user := models.UserBasic{
	//	Name:         "fyf",
	//	LoginTime:    time.Now(),
	//	HearbeatTime: time.Now(),
	//}
	//
	//user.LogoutTime = time.Now()
	//
	//fmt.Println("User data:", user)
	//
	//// 将用户存入数据库
	//if err := models.CreateUser(user); err != nil { // 使用指针传递
	//	panic(err)
	//} else {
	//	fmt.Println("User created successfully:", user)
	//}
	//fmt.Println(">>>>>>>>>>>>>>")

	// // Read
	// fmt.Println(db.First(user, 1)) // 根据整型主键查找
	// //db.First(user, "code = ?", "D42") // 查找 code 字段值为 D42 的记录

	// // Update - 将 product 的 price 更新为 200
	// db.Model(user).Update("PassWord", "1234")
	// Update - 更新多个字段
	//db.Model(&product).Updates(Product{Price: 200, Code: "F42"}) // 仅更新非零值字段
	//db.Model(&product).Updates(map[string]interface{}{"Price": 200, "Code": "F42"})

	// Delete - 删除 product
	//db.Delete(&product, 1)
}
