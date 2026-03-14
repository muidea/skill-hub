package service

import adapterpkg "github.com/muidea/skill-hub/internal/adapter"

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ForTarget(target string) (adapterpkg.Adapter, error) {
	return adapterpkg.GetAdapterForTarget(target)
}

func (a *Adapter) SupportedTargets() []string {
	return adapterpkg.GetSupportedTargets()
}

func (a *Adapter) Available() []adapterpkg.Adapter {
	return adapterpkg.GetAvailableAdapters()
}

func (a *Adapter) CleanupTimestampedBackupDirs(basePath string) error {
	return adapterpkg.CleanupTimestampedBackupDirs(basePath)
}
