package aptossync

import (
	"errors"
	"fmt"
)

// ErrNetworkUnmatched 当配置的网络与实际连接的网络不匹配时返回
func ErrNetworkUnmatched(expected, actual string) error {
	msg := fmt.Sprintf("network mismatch: expected=%v, actual=%v", expected, actual)
	return errors.New(msg)
}

// ErrModuleNotFound 当指定的模块地址不存在时返回
func ErrModuleNotFound(moduleAddress string) error {
	return fmt.Errorf("module not found at address: %s", moduleAddress)
}

// ErrFailedToGetEvents 当获取事件失败时返回
func ErrFailedToGetEvents(eventHandle string, err error) error {
	return fmt.Errorf("failed to get events from handle %s: %v", eventHandle, err)
}