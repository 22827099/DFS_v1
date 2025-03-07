package disk

import "fmt"

// 磁盘空间管理
type DiskManager struct {
	TotalSpace int64
	UsedSpace  int64
	FreeSpace  int64
}

func (dm *DiskManager) CheckSpace() error {
	// 检查磁盘空间
	if dm.FreeSpace < 1024 {
		return fmt.Errorf("磁盘空间不足")
	}
	return nil
}
