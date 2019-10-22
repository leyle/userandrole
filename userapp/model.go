package userapp

import (
	"github.com/go-redis/redis"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/ophistory"
	"github.com/silenceper/wechat"
	"github.com/silenceper/wechat/cache"
	"github.com/silenceper/wechat/oauth"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 定义一个程序管理员
const (
	AdminLoginId = "admin"
	AdminLoginPasswd = "admin" // 系统初始化后，可以修改
)

var AdminUserId = ""

const TokenRedisPrefix = "USER:TOKEN:USERID"

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
const CombineAccountBanReason = "合并账户，本账户停用"
type User struct {
	Id string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"` // 如果是 id 登录，就是 id，如果是email 登录，就是 email，如果是手机号，就是手机号，如果是微信/QQ就是暱称
	Avatar string `json:"avatar" bson:"-"`

	// 封禁
	Ban       bool   `json:"ban" bson:"ban"`
	BanT      int64  `json:"banT" bson:"banT"` // 封禁到期时间
	BanReason string `json:"banReason" bson:"banReason"`

	History []*ophistory.OperationHistory `json:"history" bson:"history"` // 操作历史记录

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`

	// 如果发生迁移，此处记录的是迁移到目标 User 的 id
	// 本账户就被标记为 ban = true
	ReferId string `json:"referId" bson:"referId"`

	// 以下内容是序列化到 redis 中需要的
	Platform string `json:"platform" bson:"-"`
	LoginType string `json:"loginType" bson:"-"`

	IdPasswd *UserLoginIdPasswdAuth `json:"idPasswd" bson:"-"`
	PhoneAuth *PhoneAuth `json:"phoneAuth" bson:"-"`
	WeChatAuth *WeChatAuth `json:"weChatAuth" bson:"-"`

	Ip string `json:"ip" bson:"-"`
}

// 账户密码登录方式
const CollectionNameIdPasswd = "idPasswdAuth"
var IKIdPasswd = &dbandmq.IndexKey{
	Collection:    CollectionNameIdPasswd,
	SingleKey:     []string{"userId", "selfReg"},
	UniqueKey:     []string{"loginId"},
}
type UserLoginIdPasswdAuth struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	LoginId string `json:"loginId" bson:"loginId"`
	Salt string `json:"-" bson:"salt"`
	Passwd string `json:"-" bson:"passwd"`
	Init bool `json:"init" bson:"init"` // 是否初始化，帮人创建的时候，是 true，修改密码后就是 false, 自主注册，是 false
	SelfReg bool `json:"selfReg" bson:"selfReg"` // 是否自己主动注册的，还是管理员后台创建的
	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 手机验证码登录
const CollectionNamePhone = "phoneAuth"
var IKPhone = &dbandmq.IndexKey{
	Collection:    CollectionNamePhone,
	SingleKey:     []string{"userId", "selfReg"},
	UniqueKey:     []string{"phone"},
}
type PhoneAuth struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`
	Phone string `json:"phone" bson:"phone"`
	Init bool `json:"init" bson:"init"` // 是否初始化，帮人创建的时候，是 true，自主注册，是 false
	SelfReg bool `json:"selfReg" bson:"selfReg"` // 是否自己主动注册的，还是管理员后台创建的
	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 微信登录
const CollectionNameWeChat = "weChatAuth"
var IKWeChat = &dbandmq.IndexKey{
	Collection:    CollectionNameWeChat,
	SingleKey:     []string{"userId", "unionId"},
	UniqueKey:     []string{"openId"},
}
type WeChatAuth struct {
	Id string `json:"id" bson:"_id"`
	UserId string `json:"userId" bson:"userId"`

	OpenId string `json:"openId" bson:"openId"`
	UnionId string `json:"unionId" bson:"unionId"`
	Nickname string `json:"nickname" bson:"nickname"`
	Sex int32 `json:"sex" bson:"sex"`
	Avatar string `json:"avatar" bson:"avatar"`
	City string `json:"city" bson:"city"`
	Province string `json:"province" bson:"province"`
	Country string `json:"country" bson:"country"`

	CreateT *util.CurTime `json:"-" bson:"createT"`
	UpdateT *util.CurTime `json:"-" bson:"updateT"`
}

// 微信配置
const (
	WeChatOptPlatformWeb = "H5" // web 授权
	WeChatOptPlatformApp = "APP" // app 登录
)
type WeChatOption struct {
	AppId string
	Secret string
	Token string
	AesKey string
}

// 其他登录方式 todo

func GetUserById(db *dbandmq.Ds, id string) (*User, error) {
	var u *User
	err := db.C(CollectionNameUser).FindId(id).One(&u)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据id[%s]读取user表失败, %s", id, err.Error())
		return nil, err
	}
	return u, nil
}

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
	user, err := GetUserById(db, ulpa.UserId)
	if err != nil {
		return nil, err
	}
	user.LoginType = LoginTypeIdPasswd
	user.Name = ulpa.LoginId
	user.IdPasswd = ulpa

	return user, nil
}

// 读取微信配置
func GetWeChatConfig(r *redis.Client, platform string, opt *WeChatOption) *wechat.Config {
	rOpt := r.Options()
	rc := &cache.RedisOpts{
		Host:        rOpt.Addr,
		Password:    rOpt.Password,
		Database:    rOpt.DB,
	}

	rCache := cache.NewRedis(rc)
	if platform == WeChatOptPlatformWeb {
		cf := &wechat.Config{
			AppID:          opt.AppId,
			AppSecret:      opt.Secret,
			Token:          opt.Token,
			EncodingAESKey: opt.AesKey,
			Cache:          rCache,
		}
		return cf
	} else {
		cf := &wechat.Config{
			AppID:          opt.AppId,
			AppSecret:      opt.Secret,
		}
		return cf
	}
}

// 存储或更新微信登录
// 返回 token 和 user 结构
func SaveWeChatLogin(db *dbandmq.Ds, r *redis.Client, wxInfo *oauth.UserInfo) (*User, string, error) {
	openId := wxInfo.OpenID
	user, err := GetUserByOpenId(db, openId)
	if err != nil {
		return nil, "", err
	}

	if user == nil {
		user, err = saveWeChatLogin(db, wxInfo)
		if err != nil {
			return nil, "", err
		}
	}

	// 生成 token
	token, err := GenerateToken(user.Id, LoginTypeWeChat)
	if err != nil {
		return nil, "", err
	}
	err = SaveToken(r, token, user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func saveWeChatLogin(db *dbandmq.Ds, wxInfo *oauth.UserInfo) (*User, error) {
	user := &User{
		Id:        util.GenerateDataId(),
		Name:      wxInfo.Nickname,
		Ban:       false,
		BanT:      0,
		BanReason: "",
		CreateT:   util.GetCurTime(),
	}
	user.UpdateT = user.CreateT

	wxa := &WeChatAuth{
		Id:       util.GenerateDataId(),
		UserId:   user.Id,
		OpenId:   wxInfo.OpenID,
		UnionId:  wxInfo.Unionid,
		Nickname: wxInfo.Nickname,
		Sex:      wxInfo.Sex,
		Avatar:   wxInfo.HeadImgURL,
		City:     wxInfo.City,
		Province: wxInfo.Province,
		Country:  wxInfo.Country,
		CreateT:  user.CreateT,
		UpdateT:  user.UpdateT,
	}

	err := db.C(CollectionNameUser).Insert(user)
	if err != nil {
		Logger.Errorf("", "微信[%s][%s]登录时，创建user信息失败, %s", wxInfo.OpenID, wxInfo.Nickname, err.Error())
		return nil, err
	}

	err = db.C(CollectionNameWeChat).Insert(wxa)
	if err != nil {
		Logger.Errorf("", "微信登录时，保存微信[%s][%s]信息失败, %s", wxInfo.OpenID, wxInfo.Nickname, err.Error())
		return nil, err
	}
	user.LoginType = LoginTypeWeChat
	user.Name = wxa.Nickname
	user.Avatar = wxa.Avatar
	user.WeChatAuth = wxa

	return user, nil
}

func GetUserByOpenId(db *dbandmq.Ds, openId string) (*User, error) {
	f := bson.M{
		"openId": openId,
	}

	var wx *WeChatAuth
	err := db.C(CollectionNameWeChat).Find(f).One(&wx)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据openId[%s]读取登录信息失败, %s", openId, err.Error())
		return nil, err
	}

	if wx == nil {
		return nil, nil
	}

	user, err := GetUserById(db, wx.UserId)
	if err != nil {
		return nil, err
	}

	user.LoginType = LoginTypeWeChat
	user.Name = wx.Nickname
	user.Avatar = wx.Avatar
	user.WeChatAuth = wx

	return user, nil
}

// 返回 token 和 user 结构
func SavePhoneLogin(db *dbandmq.Ds, r *redis.Client, phone string) (*User, string, error) {
	user, err := GetUserByPhone(db, phone)
	if err != nil {
		return nil, "", err
	}

	if user == nil {
		user, err = savePhoneLogin(db, phone)
		if err != nil {
			return nil, "", err
		}
	}

	// 生成 token
	token, err := GenerateToken(user.Id, LoginTypePhone)
	if err != nil {
		return nil, "", err
	}
	err = SaveToken(r, token, user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func savePhoneLogin(db *dbandmq.Ds, phone string) (*User, error) {
	user := &User{
		Id:        util.GenerateDataId(),
		Name:      phone,
		Avatar:    "",
		Ban:       false,
		BanT:      0,
		BanReason: "",
		CreateT:   util.GetCurTime(),
	}
	user.UpdateT = user.CreateT

	pa := &PhoneAuth{
		Id:      util.GenerateDataId(),
		UserId:  user.Id,
		Phone:   phone,
		Init:    false,
		CreateT: user.CreateT,
		UpdateT: user.UpdateT,
	}

	err := db.C(CollectionNameUser).Insert(user)
	if err != nil {
		Logger.Errorf("", "创建phone[%s]登录信息时，保存user信息失败, %s", phone, err.Error())
		return nil, err
	}

	err = db.C(CollectionNamePhone).Insert(pa)
	if err != nil {
		Logger.Errorf("", "创建phone[%s]登录信息时，保存phoneauth信息失败, %s", phone, err.Error())
		return nil, err
	}

	user.LoginType = LoginTypePhone
	user.Name = phone
	user.PhoneAuth = pa

	return user, nil
}

// 管理员创建 phone 账户
func InitPhoneAuth(db *dbandmq.Ds, phone string) (*User, error) {
	return savePhoneLogin(db, phone)
}

func GetUserByPhone(db *dbandmq.Ds, phone string) (*User, error) {
	f := bson.M{
		"phone": phone,
	}

	var pa *PhoneAuth
	err := db.C(CollectionNamePhone).Find(f).One(&pa)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Errorf("", "根据phone[%s]读取登录信息失败, %s", phone, err.Error())
		return nil, err
	}

	if pa == nil {
		return nil, nil
	}

	user, err := GetUserById(db, pa.UserId)
	if err != nil {
		return nil, err
	}

	user.LoginType = LoginTypePhone
	user.Name = phone
	user.PhoneAuth = pa

	return user, nil
}

// 读取用户的所有可能的信息
func GetUserFullInfoById(db *dbandmq.Ds, userId string) (*User, error) {
	user, err := GetUserById(db, userId)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, nil
	}

	// id passwd 信息
	idp, _ := getIdPasswdAuthByUserId(db, userId)
	user.IdPasswd = idp

	// phone
	pa, _ := getPhoneAuthByUserId(db, userId)
	user.PhoneAuth = pa

	// wechat
	wca, _ := getWeChatAuthByUserId(db, userId)
	user.WeChatAuth = wca

	return user, nil
}

func getIdPasswdAuthByUserId(db *dbandmq.Ds, userId string) (*UserLoginIdPasswdAuth, error) {
	f := bson.M{
		"userId": userId,
	}

	var ulp *UserLoginIdPasswdAuth
	err := db.C(CollectionNameIdPasswd).Find(f).One(&ulp)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}

	return ulp, nil
}

func getPhoneAuthByUserId(db *dbandmq.Ds, userId string) (*PhoneAuth, error) {
	f := bson.M{
		"userId": userId,
	}

	var pa *PhoneAuth
	err := db.C(CollectionNamePhone).Find(f).One(&pa)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}

	return pa, nil
}

func getWeChatAuthByUserId(db *dbandmq.Ds, userId string) (*WeChatAuth, error) {
	f := bson.M{
		"userId": userId,
	}

	var wca *WeChatAuth
	err := db.C(CollectionNameWeChat).Find(f).One(&wca)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}

	return wca, nil
}

// 搜索 loginid 模糊匹配信息
func QueryLoginIdAuthByLoginId(db *dbandmq.Ds, lid string) ([]*UserLoginIdPasswdAuth, error) {
	f := bson.M{
		"loginId": bson.M{
			"$regex": lid,
		},
	}

	var lps []*UserLoginIdPasswdAuth
	err := db.C(CollectionNameIdPasswd).Find(f).All(&lps)
	if err != nil {
		return nil, err
	}

	return lps, nil
}

// 搜索 phone 模糊匹配信息
func QueryPhoneAuthByPhone(db *dbandmq.Ds, phone string) ([]*PhoneAuth, error) {
	f := bson.M{
		"phone": bson.M{
			"$regex": phone,
		},
	}

	var pas []*PhoneAuth
	err := db.C(CollectionNamePhone).Find(f).All(&pas)
	if err != nil {
		return nil, err
	}

	return pas, nil
}

// 搜索 wechat 模糊匹配信息
func QueryWeChatAuthByNickname(db *dbandmq.Ds, nickname string) ([]*WeChatAuth, error) {
	f := bson.M{
		"nickname": bson.M{
			"$regex": nickname,
		},
	}

	var wcas []*WeChatAuth
	err := db.C(CollectionNameWeChat).Find(f).All(&wcas)
	if err != nil {
		return nil, err
	}

	return wcas, nil
}

func IsValidPlatform(p string) bool {
	vals := []string{
		LoginPlatformH5,
		LoginPlatformAndroid,
		LoginPlatformIOS,
		LoginPlatformPC,
	}

	for _, val := range vals {
		if val == p {
			return true
		}
	}

	return false
}