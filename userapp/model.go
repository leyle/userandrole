package userapp

import "github.com/leyle/ginbase/util"

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
	Domain string `json:"domain" bson:"domain"`
	Name string `json:"name" bson:"name"` // 如果是 id 登录，就是 id，如果是email 登录，就是 email，如果是手机号，就是手机号，如果是微信/QQ就是暱称

	// 以下内容是序列化到 redis 中需要的
	Platform string `json:"platform" bson:"-"`
	LoginType string `json:"loginType" bson:"-"`

	IdPasswd *UserLoginIdPasswdAuth `json:"idPasswd" bson:"-"`
	UserEmail *UserEmailAuth `json:"userEmail" bson:"-"`
	Ip string `json:"ip" bson:"-"`
}

// 账户密码登录方式
const CollectionNameIdPasswd = "idPasswdAuth"
type UserLoginIdPasswdAuth struct {
	Id string `json:"id" bson:"id"`
	UserId string `json:"userId" bson:"userId"`
	LoginId string `json:"loginId" bson:"loginId"`
	Salt string `json:"-" bson:"salt"`
	Passwd string `json:"-" bson:"passwd"`
	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 邮箱登录方式
// 填写邮箱，发送一个验证码到邮箱，填写进来即可登录
const CollectionNameEmail = "emaliAuth"
type UserEmailAuth struct {
	Id string `json:"id" bson:"id"`
	UserId string `json:"userId" bson:"userId"`
	Email string `json:"email" bson:"email"`
	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 其他登录方式 todo