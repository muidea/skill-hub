package adapter

import (
	"testing"
)

func TestAdapterManager(t *testing.T) {
	// 测试获取支持的target
	targets := GetSupportedTargets()
	if len(targets) == 0 {
		t.Error("没有注册任何Adapter")
	}

	// 测试获取每个target的Adapter
	for _, target := range targets {
		adapter, err := GetAdapterForTarget(target)
		if err != nil {
			t.Errorf("获取Adapter失败: %s: %v", target, err)
			continue
		}

		// 测试Adapter接口方法
		if adapter.GetTarget() != target {
			t.Errorf("Adapter target不匹配: 期望 %s, 实际 %s", target, adapter.GetTarget())
		}

		// 测试模式设置
		adapter.SetProjectMode()
		if adapter.GetMode() != "project" {
			t.Errorf("设置项目模式失败: %s", target)
		}

		adapter.SetGlobalMode()
		if adapter.GetMode() != "global" {
			t.Errorf("设置全局模式失败: %s", target)
		}

		// 测试Supports方法
		supported := adapter.Supports()
		t.Logf("Adapter %s 支持当前环境: %v", target, supported)

		// 测试GetBackupPath方法
		backupPath := adapter.GetBackupPath()
		t.Logf("Adapter %s 备份路径: %s", target, backupPath)
	}

	// 测试获取可用Adapter
	availableAdapters := GetAvailableAdapters()
	t.Logf("当前环境可用Adapter数量: %d", len(availableAdapters))
}

func TestAdapterRegistration(t *testing.T) {
	// 创建一个测试Adapter
	testAdapter := &testAdapterImpl{}

	// 注册测试Adapter
	RegisterAdapter("test_target", testAdapter)

	// 验证是否注册成功
	adapter, err := GetAdapterForTarget("test_target")
	if err != nil {
		t.Errorf("获取测试Adapter失败: %v", err)
	}

	if adapter.GetTarget() != "test_target" {
		t.Errorf("测试Adapter target不匹配")
	}
}

// testAdapterImpl 测试用的Adapter实现
type testAdapterImpl struct {
	mode string
}

func (t *testAdapterImpl) Apply(skillID string, content string, variables map[string]string) error {
	return nil
}

func (t *testAdapterImpl) Extract(skillID string) (string, error) {
	return "", nil
}

func (t *testAdapterImpl) Remove(skillID string) error {
	return nil
}

func (t *testAdapterImpl) List() ([]string, error) {
	return []string{}, nil
}

func (t *testAdapterImpl) Supports() bool {
	return true
}

func (t *testAdapterImpl) Cleanup() error {
	return nil
}

func (t *testAdapterImpl) GetBackupPath() string {
	return ""
}

func (t *testAdapterImpl) GetTarget() string {
	return "test_target"
}

func (t *testAdapterImpl) GetSkillPath(skillID string) (string, error) {
	return "", nil
}

func (t *testAdapterImpl) SetProjectMode() {
	t.mode = "project"
}

func (t *testAdapterImpl) SetGlobalMode() {
	t.mode = "global"
}

func (t *testAdapterImpl) GetMode() string {
	return t.mode
}
