// Package models 定义数据模型
package models

import "time"

// 用户角色常量
const (
	RoleEnterprise    = "enterprise"     // 小微企业企业角色
	RoleParkAdmin     = "park_admin"     // 工业园区管理员角色
	RoleExchange      = "exchange"       // 碳交易所交易角色
	RoleRegulator     = "regulator"      // 监管核证角色
)

// User 用户账号模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Password  string    `json:"-" gorm:"size:256;not null"` // JSON序列化时忽略密码
	Role      string    `json:"role" gorm:"size:32;not null;index"`
	Company   string    `json:"company" gorm:"size:128"`                  // 企业名称
	ParkID    uint      `json:"park_id" gorm:"default:0;index"`           // 所属园区ID(企业角色用)
	Status    int       `json:"status" gorm:"default:1"`                  // 1-正常 0-禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Company  string `json:"company"`
	Role     string `json:"role" binding:"required"`
	ParkID   uint   `json:"park_id"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}