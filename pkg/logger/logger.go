// Package logger 分级日志模块
//
// 变更说明(v2)：提供 INFO/WARN/ERROR/DEBUG 分级日志，输出到控制台与日志文件，
// 供上链、积分变动、交易撮合等关键操作记录使用(操作人/时间/业务单号/哈希)。
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 日志级别常量(数值越大越严重)
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	mu      sync.Mutex
	fileOut *os.File
	level   = LevelInfo
)

// levelText 输出用的级别标签(定宽对齐)
func levelText(lv int) string {
	switch lv {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO "
	case LevelWarn:
		return "WARN "
	default:
		return "ERROR"
	}
}

// parseLevel 将配置字符串解析为级别
func parseLevel(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Init 初始化日志系统
// levelName: debug/info/warn/error；filePath: 日志文件，空则仅控制台。
// 替换原因(v2)：原实现直接使用标准 log 且无分级，无法支撑关键操作审计输出。
func Init(levelName, filePath string) {
	level = parseLevel(levelName)
	if filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err == nil {
			f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				fileOut = f
				return
			}
		}
	}
}

// write 统一写出：文件(如有) + 控制台
func write(lv int, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if lv < level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), levelText(lv), msg)

	if fileOut != nil {
		_, _ = fileOut.WriteString(line)
	}
	// 控制台同步输出，便于启动与演示时直接观察关键操作
	_, _ = os.Stdout.WriteString(line)
}

// Close 关闭日志文件句柄(进程退出时由 main 调用)
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if fileOut != nil {
		_ = fileOut.Close()
		fileOut = nil
	}
}

// Debug 输出调试日志
func Debug(format string, args ...interface{}) { write(LevelDebug, format, args...) }

// Info 输出信息日志
func Info(format string, args ...interface{}) { write(LevelInfo, format, args...) }

// Warn 输出告警日志
func Warn(format string, args ...interface{}) { write(LevelWarn, format, args...) }

// Error 输出错误日志(记录堆栈由调用方决定是否传入)
func Error(format string, args ...interface{}) { write(LevelError, format, args...) }

// Writer 返回一个 io.Writer，可将第三方库(如标准 log)输出接入分级体系
func Writer() io.Writer { return &writer{} }

// GormWriter 适配 GORM 日志接口(实现 gorm.io/gorm/logger.Writer 的 Printf)
func GormWriter() *GormAdapter { return &GormAdapter{} }

// GormAdapter GORM 日志适配器：慢查询/错误统一按分级日志输出
type GormAdapter struct{}

func (GormAdapter) Printf(format string, args ...interface{}) { Info(format, args...) }

// writer 实现 io.Writer，把写入内容按 Info 级别输出
type writer struct{}

func (writer) Write(p []byte) (n int, err error) {
	mu.Lock()
	defer mu.Unlock()
	s := strings.TrimRight(string(p), "\r\n")
	if s == "" {
		return len(p), nil
	}
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), levelText(LevelInfo), s)
	if fileOut != nil {
		_, _ = fileOut.WriteString(line)
	}
	_, _ = os.Stdout.WriteString(line)
	return len(p), nil
}
