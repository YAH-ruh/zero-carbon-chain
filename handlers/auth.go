// Package handlers HTTP请求处理器
// 所有业务接口的实际处理逻辑
package handlers

import (
	"blockchain-demo/database"
	"blockchain-demo/middleware"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Register 用户注册
// POST /api/auth/register
// 安全机制(角色越权防护)：网页注册仅允许创建"小微企业"角色账号；
// 园区管理员、碳交易所、监管核查等高权限角色仅通过数据库预置产生，杜绝网页注册越权。
func Register(c *gin.Context) {
	var req models.RegisterRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if req.Role != models.RoleEnterprise {
		response.BadRequest(c, "网页注册仅允许创建小微企业角色，高权限账号请使用系统预置账号")
		return
	}

	// 参数业务校验：用户名长度、密码长度
	if len(req.Username) < 2 || len(req.Username) > 32 {
		response.BadRequest(c, "用户名长度需在2-32个字符之间")
		return
	}
	if len(req.Password) < 6 {
		response.BadRequest(c, "密码长度不能少于6位")
		return
	}

	var existing models.User
	if err := database.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		response.Conflict(c, "用户名已存在")
		return
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.ServerError(c, "密码加密失败")
		return
	}

	user := models.User{
		Username: req.Username,
		Password: string(hashedPwd),
		Role:     req.Role,
		Company:  req.Company,
		ParkID:   req.ParkID,
		Status:   1,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		logger.Error("用户注册失败: %v", err)
		response.ServerError(c, "创建用户失败")
		return
	}

	logOperation(nil, c, models.OpRegister, "user", req.Username, "", "新企业账号注册")
	response.Created(c, "注册成功", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
		"company":  user.Company,
		"park_id":  user.ParkID,
	})
}

// Login 用户登录
// POST /api/auth/login
// 返回JWT令牌和用户信息
func Login(c *gin.Context) {
	var req models.LoginRequest
	if !response.BindJSON(c, &req) {
		return
	}

	var user models.User
	// 支持账号或企业名称登录(企业名称已规范为全称，如 绿恒节能科技有限公司)
	if err := database.DB.Where("username = ? OR (role = ? AND company = ?)",
		req.Username, models.RoleEnterprise, req.Username).First(&user).Error; err != nil {
		response.Unauthorized(c, "用户名或密码错误")
		return
	}
	if user.Status == 0 {
		response.Forbidden(c, "账号已被禁用，请联系园区管理员")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	token, err := middleware.GenerateToken(&user)
	if err != nil {
		logger.Error("生成令牌失败: %v", err)
		response.ServerError(c, "生成令牌失败")
		return
	}

	logOperation(nil, c, models.OpLogin, "user", user.Username, "", "用户登录")
	response.OK(c, models.LoginResponse{Token: token, User: user})
}

// GetCurrentUser 获取当前登录用户信息
// GET /api/auth/me
func GetCurrentUser(c *gin.Context) {
	userID := currentUserID(c)
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}
	response.OK(c, user)
}
