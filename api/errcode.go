package api

import "errors"

var NameExistErr = errors.New("数据已存在")

const (
	ErrCodeNameExist = 4000 // 名字比如 item role loginid 已经存在
	ErrCodeXiaoChengXuNeedProfile = 2000 // 小程序登录时，需要进一步的 profile 信息
)
