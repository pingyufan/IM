package models

import (
	"IM/utils"
	"gorm.io/gorm"
)

// 人员关系
type Contact struct {
	gorm.Model
	OwnerId  uint // 谁的关系信息
	TargetId uint // 对应的 人/群 Id
	Type     int  // 对应的类型  1，私聊  2， 群聊。。。
	Desc     string
}

// 检查 GORM 初始化：确保在程序启动时，GORM 已正确连接到数据库并设置了相关的模型。
// 重写默认的表名，使 GORM 使用 contact 表。
func (table *Contact) TableName() string {
	return "contact"
}

func SearchFriend(userId uint) []UserBasic {
	contacts := make([]Contact, 0)
	objIds := make([]uint64, 0)
	utils.DB.Where("owner_id = ? and type=1", userId).Find(&contacts)
	for _, v := range contacts {
		objIds = append(objIds, uint64(v.TargetId))
	}
	users := make([]UserBasic, 0)
	utils.DB.Where("id in ?", objIds).Find(&users)
	return users
}

func AddFriend(userId uint, targetName string) (int, string) {
	if targetName != "" {
		targetUser := FindUserByName(targetName)

		if targetUser.Salt != "" {
			if targetUser.ID == userId {
				return -1, "不能添加自己为好友"
			}
			contact0 := Contact{}
			utils.DB.Where("owner_id = ? and target_Id = ? and type = 1", userId, targetUser.ID).First(&contact0)
			if contact0.ID != 0 {
				return -1, "不能重复添加"
			}
			tx := utils.DB.Begin()
			//事务一旦开始，不论什么异常最终都会 Rollback
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			contact := Contact{}
			contact.OwnerId = userId
			contact.TargetId = targetUser.ID
			contact.Type = 1
			if err := utils.DB.Create(&contact).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			contact1 := Contact{}
			contact1.OwnerId = targetUser.ID
			contact1.TargetId = userId
			contact1.Type = 1
			if err := utils.DB.Create(&contact1).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			tx.Commit()
			return 0, "添加好友成功"
		}
		return -1, "没有找到该用户"
	}
	return -1, "好友ID不能为空"
}

func SearchUserByGroupId(communityId uint) []uint {
	contacts := make([]Contact, 0)
	objIds := make([]uint, 0)
	utils.DB.Where("target_id = ? and type=2", communityId).Find(&contacts)
	for _, v := range contacts {
		objIds = append(objIds, v.OwnerId)
	}
	return objIds
}
