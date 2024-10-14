package models

import "gorm.io/gorm"

type GroupBasic struct {
	gorm.Model
	Name   string
	OwerId uint
	Icon   string
	Type   int
	Desc   string
}

func (table *GroupBasic) TableName() string {
	return "groupp_basic"
}
