package logger

import (
	"fmt"
	"log"
	"time"
)

// timeFormat единый формат времени для всех логов
const timeFormat = "2006-01-02 15:04:05"

// LogWithTime выводит сообщение с единым форматом времени
func LogWithTime(format string, args ...interface{}) {
	timestamp := time.Now().Format(timeFormat)
	message := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] %s\n", timestamp, message)
}

// LogPlain выводит сообщение без времени (для многострочных выводов)
func LogPlain(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// LogError выводит ошибку с временной меткой
func LogError(format string, args ...interface{}) {
	timestamp := time.Now().Format(timeFormat)
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", timestamp, message)
}

// LogInfo выводит информационное сообщение с временной меткой
func LogInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format(timeFormat)
	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", timestamp, message)
}
