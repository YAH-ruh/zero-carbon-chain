// Package response 统一 API 响应封装
//
// 变更说明(v2)：统一所有接口返回体为 {code, msg, data} 三元组，
// 其中 code 与 HTTP 状态码一致；统一入参绑定与错误提示，减少各处手写分支。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Body 统一响应结构体
type Body struct {
	Code int         `json:"code"` // 业务码(与 HTTP 状态一致，200 表示成功)
	Msg  string      `json:"msg"`  // 提示信息
	Data interface{} `json:"data"` // 业务数据(无数据时为 nil)
}

// OK 成功响应(200)
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: http.StatusOK, Msg: "success", Data: data})
}

// OKMsg 成功响应并携带自定义提示
func OKMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: http.StatusOK, Msg: msg, Data: data})
}

// Created 创建成功响应(201)
func Created(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusCreated, Body{Code: http.StatusCreated, Msg: msg, Data: data})
}

// Fail 统一失败响应：msg 面向用户可读，code 与 HTTP 状态保持一致
func Fail(c *gin.Context, httpStatus int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: httpStatus, Msg: msg, Data: nil})
}

// BadRequest 参数错误(400)
func BadRequest(c *gin.Context, msg string) { Fail(c, http.StatusBadRequest, msg) }

// Unauthorized 未认证(401)
func Unauthorized(c *gin.Context, msg string) { Fail(c, http.StatusUnauthorized, msg) }

// Forbidden 无权限(403)
func Forbidden(c *gin.Context, msg string) { Fail(c, http.StatusForbidden, msg) }

// NotFound 资源不存在(404)
func NotFound(c *gin.Context, msg string) { Fail(c, http.StatusNotFound, msg) }

// Conflict 资源冲突(409)
func Conflict(c *gin.Context, msg string) { Fail(c, http.StatusConflict, msg) }

// ServerError 服务器内部错误(500)，对外隐藏内部细节防止信息泄露
func ServerError(c *gin.Context, msg string) { Fail(c, http.StatusInternalServerError, msg) }

// BindJSON 绑定并校验 JSON 请求体
// 校验失败时自动返回 400 统一格式，避免重复代码；返回 false 表示已写错误响应。
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BadRequest(c, "请求参数错误: "+bindingMsg(err))
		return false
	}
	return true
}

// bindingMsg 将参数校验错误转为人话提示
func bindingMsg(err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok && len(errs) > 0 {
		e := errs[0]
		switch e.Tag() {
		case "required":
			return "字段 " + e.Field() + " 不能为空"
		case "email":
			return "字段 " + e.Field() + " 格式不正确"
		case "min":
			return "字段 " + e.Field() + " 长度不足"
		case "gt", "gte":
			return "字段 " + e.Field() + " 数值需大于设定下限"
		default:
			return e.Field() + " 校验失败(" + e.Tag() + ")"
		}
	}
	return err.Error()
}
