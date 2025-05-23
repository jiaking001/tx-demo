package model

import (
	"gorm.io/gorm"
	"time"
)

var db *gorm.DB

// User 定义用户模型
type User struct {
	gorm.Model
	ID            int64          `gorm:"primaryKey;autoIncrement:true"`
	UserID        string         `gorm:"type:varchar(36);notNull;unique"`
	Username      string         `gorm:"type:varchar(100);notNull;unique"`
	Password      string         `gorm:"type:varchar(255);notNull"`
	Like          string         `gorm:"type:varchar(255);column:like"` // 使用 column 标签指定列名
	LikeEmbedding []float64      `gorm:"type:vector(1536)"`
	CreatedAt     time.Time      `gorm:"type:timestamp;notNull;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time      `gorm:"type:timestamp;notNull;default:CURRENT_TIMESTAMP"`
	DeletedAt     gorm.DeletedAt `gorm:"index"` // 软删除字段
}

// TableName 指定默认表名
func (User) TableName() string {
	return "users"
}
