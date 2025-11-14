package LogModule

import (
	"encoding/json"
	"time"
)

// LogModule 日志所属业务模块
// 使用 int 类型方便在数据库中直接存储
// 业务可自行定义枚举常量
type LogModule int

// LogAction 日志所属业务动作
// 不同的动作可以区分新增、修改、删除等操作
type LogAction int

// LogEntity 数据库实体
// Detail 存储 JSON 字符串
type LogEntity struct {
	ID      int64  `json:"id"`
	Module  int    `json:"module"`
	Action  int    `json:"action"`
	Message string `json:"message"`
	Detail  string `json:"detail"`

	OperatorID string `json:"operator_id"`
	IP         string `json:"ip"`

	CreateTime int64 `json:"create_time"`
}

// LogItem 对外展示结构
// Detail 保留 interface{} 便于业务直接使用反序列化后的结果
type LogItem struct {
	ID      int64       `json:"id"`
	Module  LogModule   `json:"module"`
	Action  LogAction   `json:"action"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail"`

	OperatorID string    `json:"operator_id"`
	IP         string    `json:"ip"`
	CreateTime time.Time `json:"create_time"`
}

// ConvertLogItem 将数据库实体转换为业务结构
func ConvertLogItem(in *LogEntity) (out *LogItem) {
	if in == nil {
		return nil
	}

	out = &LogItem{
		ID:         in.ID,
		Module:     LogModule(in.Module),
		Action:     LogAction(in.Action),
		Message:    in.Message,
		OperatorID: in.OperatorID,
		IP:         in.IP,
		CreateTime: time.Unix(in.CreateTime, 0),
	}

	if in.Detail != "" {
		_ = json.Unmarshal([]byte(in.Detail), &out.Detail)
	}

	return out
}

// NewLogEntityFromItem 根据业务结构构建数据库实体
// Detail 会序列化为 JSON 字符串，CreateTime 默认为当前时间
func NewLogEntityFromItem(in *LogItem) *LogEntity {
	if in == nil {
		return nil
	}

	detailBytes, _ := json.Marshal(in.Detail)
	createTime := in.CreateTime
	if createTime.IsZero() {
		createTime = time.Now()
	}

	return &LogEntity{
		Module:     int(in.Module),
		Action:     int(in.Action),
		Message:    in.Message,
		Detail:     string(detailBytes),
		OperatorID: in.OperatorID,
		IP:         in.IP,
		CreateTime: createTime.Unix(),
	}
}
