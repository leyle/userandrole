package ophistory

import (
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"gopkg.in/mgo.v2/bson"
	. "github.com/leyle/ginbase/consolelog"
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

// 账户登录历史记录
const CollectionNameLoginHistory = "loginHistory"
var IKLoginHistory = &dbandmq.IndexKey{
	Collection:    CollectionNameLoginHistory,
	SingleKey:     []string{"userId", "loginType", "platform", "loginT"},
}
type LoginHistory struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	UserName string `json:"userName" bson:"userName"`
	LoginType string `json:"loginType" bson:"loginType"` // 登录类型，账户密码、手机号、微信号等
	Platform string `json:"platform" bson:"platform"` // 平台
	Ip string `json:"ip" bson:"ip"`
	UserAgent string `json:"userAgent" bson:"userAgent"`
	LoginT *util.CurTime `json:"loginT" bson:"loginT"`
}

// 根据用户id读取历史记录
func GetLoginHistoryByUserId(db *dbandmq.Ds, userId string, page int) ([]*LoginHistory, error) {
	f := bson.M{
		"userId": userId,
	}

	size := 10
	skip := (page - 1) * size

	var lhs []*LoginHistory
	err := db.C(CollectionNameLoginHistory).Find(f).Sort("-_id").Skip(skip).Limit(size).All(&lhs)
	if err != nil {
		Logger.Errorf("", "根据用户id[%s]和page[%d]读取登录历史失败, %s", userId, page, err.Error())
		return nil, err
	}

	return lhs, nil
}

func SaveLoginHistory(db *dbandmq.Ds, lh *LoginHistory) error {
	return db.C(CollectionNameLoginHistory).Insert(lh)
}