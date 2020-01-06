package migrate

import "github.com/leyle/userandrole/roleapp"

// 数据的导出和导入结构
type Migrate struct {
	Items []*roleapp.Item `json:"items"`
	Permissions []*roleapp.Permission `json:"permissions"`
	Roles []*roleapp.Role `json:"roles"`
}