package adapter

import (
	"fmt"
	"sort"
)

// Manager 管理所有Adapter实例
type Manager struct {
	adapters map[string]Adapter
}

// NewManager 创建新的Adapter管理器
func NewManager() *Manager {
	return &Manager{
		adapters: make(map[string]Adapter),
	}
}

// Register 注册Adapter
func (m *Manager) Register(target string, adapter Adapter) {
	m.adapters[target] = adapter
}

// GetAdapter 获取指定target的Adapter
func (m *Manager) GetAdapter(target string) (Adapter, error) {
	adapter, exists := m.adapters[target]
	if !exists {
		return nil, fmt.Errorf("不支持的目标环境: %s", target)
	}
	return adapter, nil
}

// GetSupportedTargets 获取所有支持的target
func (m *Manager) GetSupportedTargets() []string {
	targets := make([]string, 0, len(m.adapters))
	for target := range m.adapters {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

// GetAdapterForProject 根据项目路径获取合适的Adapter
func (m *Manager) GetAdapterForProject(projectPath string) (Adapter, error) {
	// 这里可以根据项目配置或自动检测来选择合适的Adapter
	// 目前返回默认的OpenCode Adapter
	return m.GetAdapter("open_code")
}

// GetAvailableAdapters 获取当前环境中可用的Adapter
func (m *Manager) GetAvailableAdapters() []Adapter {
	var available []Adapter
	for _, adapter := range m.adapters {
		if adapter.Supports() {
			available = append(available, adapter)
		}
	}
	return available
}

// DefaultManager 默认的Adapter管理器
var DefaultManager = NewManager()

// RegisterAdapter 便捷函数：注册Adapter
func RegisterAdapter(target string, adapter Adapter) {
	DefaultManager.Register(target, adapter)
}

// GetAdapterForTarget 便捷函数：获取指定target的Adapter
func GetAdapterForTarget(target string) (Adapter, error) {
	return DefaultManager.GetAdapter(target)
}

// GetSupportedTargets 便捷函数：获取所有支持的target
func GetSupportedTargets() []string {
	return DefaultManager.GetSupportedTargets()
}

// GetAvailableAdapters 便捷函数：获取当前环境中可用的Adapter
func GetAvailableAdapters() []Adapter {
	return DefaultManager.GetAvailableAdapters()
}
