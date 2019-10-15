package userapp

import (
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	. "github.com/leyle/ginbase/consolelog"
)

// 定义一个程序管理员
const (
	AdminLoginId = "admin"
	AdminLoginPasswd = "admin" // 系统初始化后，可以修改
)

const TokenRedisPrefix = "USER:TOKEN:USERID:"

// 存储到 redis 中的 token 信息
// 包含了 token 值外，还有用户信息
type TokenVal struct {
	Token string `json:"token"`
	User *User `json:"user"`
	T *util.CurTime `json:"t"`
}

// 登录方式
const (
	LoginTypeIdPasswd = "IDPASSWD" // 账户密码登录
	LoginTypeEmail = "EMAIL"
	LoginTypePhone = "PHONE"
	LoginTypeWeChat = "WECHAT"
	LoginTypeQQ = "QQ"
)

// 登录平台
const (
	LoginPlatformH5 = "H5" // h5 页面
	LoginPlatformPC = "PC" // pc browser
	LoginPlatformAndroid = "ANDROID"
	LoginPlatformIOS = "IOS"
)

// 用户管理，支持域
const CollectionNameUser = "user"
type User struct {
	Id string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"` // 如果是 id 登录，就是 id，如果是email 登录，就是 email，如果是手机号，就是手机号，如果是微信/QQ就是暱称

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`

	// 以下内容是序列化到 redis 中需要的
	Platform string `json:"platform" bson:"-"`
	LoginType string `json:"loginType" bson:"-"`

	IdPasswd *UserLoginIdPasswdAuth `json:"idPasswd" bson:"-"`
	Ip string `json:"ip" bson:"-"`
}

// 账户密码登录方式
const CollectionNameIdPasswd = "idPasswdAuth"
type UserLoginIdPasswdAuth struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	LoginId string `json:"loginId" bson:"loginId"`
	Salt string `json:"-" bson:"salt"`
	Passwd string `json:"-" bson:"passwd"`
	Init bool `json:"init" bson:"init"` // 是否初始化，帮人创建的时候，是 true，修改密码后就是 false, 自主注册，是 false
	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 其他登录方式 todo

// 根据 loginId 查询登录信息
func GetUserByLoginId(db *dbandmq.Ds, loginId string) (*User, error) {
	f := bson.M{
		"loginId": loginId,
	}

	var ulpa *UserLoginIdPasswdAuth
	err := db.C(CollectionNameIdPasswd).Find(f).One(&ulpa)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据loginId[%s]查询登录信息失败, %s", err.Error())
		return nil, err
	}

	if err == mgo.ErrNotFound {
		return nil, nil
	}

	// 补充成 User 信息
	user := &User{
		Id:        ulpa.UserId,
		Name:      ulpa.LoginId,
		LoginType: LoginTypeIdPasswd,
		IdPasswd:  ulpa,
	}

	return user, nil
}