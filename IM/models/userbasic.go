package models

import (
	"IM/utils"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type UserBasic struct {
	gorm.Model
	Name         string
	PassWord     string `valid:"matches(^1[3-9]{1}\\d{9}$)" json:"password"`
	Phone        string `gorm:"type:varchar(11)" json:"phone"`
	Email        string `valid:"email" json:"email"`
	Avatar       string `gorm:"type:varchar(255)" json:"avatar"` //头像
	Identity     string
	ClientIP     string
	ClientPort   string
	Salt         string
	LoginTime    time.Time `json:"login_time"`
	HearbeatTime time.Time `json:"heartbeat_time"`
	LogoutTime   time.Time `gorm:"column:login_out_time;default NULL" json:"login_out_time"`
	IsLogout     bool      `json:"is_logout"`                            // 默认值为 false
	DeviceInfo   string    `gorm:"type:varchar(255)" json:"device_info"` // 设备信息
}

// 自动映射结构体与数据库表
func (table *UserBasic) TableName() string {
	return "user_basic"
}

func CreateUser(user UserBasic) *gorm.DB {
	result := utils.DB.Create(&user)
	if result.Error != nil {
		fmt.Println("Error creating user:", result.Error)
	} else {
		fmt.Println("success creating user: ", user)
	}
	return result
}
func DeleteUser(user UserBasic) *gorm.DB {
	return utils.DB.Delete(&user)
}
func UpdateUser(user UserBasic) *gorm.DB {
	return utils.DB.Model(&user).Updates(UserBasic{Name: user.Name, PassWord: user.PassWord, Phone: user.Phone,
		Email: user.Email, Avatar: user.Avatar})
}

func GetUserList() []*UserBasic {
	data := make([]*UserBasic, 0)
	utils.DB.Find(&data)
	for _, v := range data {
		fmt.Println(v)
	}
	return data
}

// 登录
func FindUserByNameAndPwd(name string, pwd string) UserBasic {
	user := UserBasic{}
	utils.DB.Where("name = ? and pass_word = ?", name, pwd).First(&user)

	//token加密
	str := fmt.Sprintf("%d", time.Now().Unix())
	temp := utils.MD5Encode(str)
	utils.DB.Model(&user).Where("id = ?", user.ID).Update("identity", temp)
	return user
}

// 查找
func FindUserByName(name string) UserBasic {
	user := UserBasic{}
	utils.DB.Where("name = ?", name).First(&user)
	return user
}

// 根据ID查找
func FindByID(id uint) *UserBasic {
	user := UserBasic{}
	utils.DB.Where("id = ?", id).First(&user)
	return &user
}
