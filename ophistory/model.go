package ophistory

import (
	"github.com/leyle/ginbase/util"
)

// 操作历史记录
type OperationHistory struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	UserName string `json:"userName" bson:"userName"`
	Action string `json:"action" bson:"action"` // 拼接起来操作字符串
	T *util.CurTime `json:"t" bson:"t"`
}

func NewOpHistory(userId, userName, action string) *OperationHistory {
	opHis := &OperationHistory{
		Id:       util.GenerateDataId(),
		UserId:   userId,
		UserName: userName,
		Action: action,
		T:        util.GetCurTime(),
	}
	return opHis
}