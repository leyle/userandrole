package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/smsapp"
	"github.com/leyle/userandrole/ophistory"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userandrole"
	"github.com/leyle/userandrole/userapp"
	"github.com/silenceper/wechat"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
)

// 通过账户密码数据，没有自己注册的，都是通过接口创建的
type CreateIdPasswdForm struct {
	LoginId string `json:"loginId" binding:"required"`
	Passwd string `json:"passwd" binding:"required"`
}
func CreateLoginIdPasswdAccountHandler(c *gin.Context, ro *UserOption) {
	var form CreateIdPasswdForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	// 悲观锁
	lockVal, ok := dbandmq.AcquireLock(ro.R, form.LoginId, dbandmq.DEFAULT_LOCK_ACQUIRE_TIMEOUT, dbandmq.DEFAULT_LOCK_KEY_TIMEOUT)
	if !ok {
		returnfun.ReturnErrJson(c, "锁定数据失败")
		return
	}
	defer dbandmq.ReleaseLock(ro.R, form.LoginId, lockVal)

	db := ro.Ds.CopyDs()
	defer db.Close()

	// 检查账户是否已存在
	dbuser, err := userapp.GetUserByLoginId(db, form.LoginId)
	middleware.StopExec(err)
	if dbuser != nil {
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "账户已存在", gin.H{"id": dbuser.Id})
		return
	}

	if len(form.Passwd) < 6 {
		returnfun.ReturnErrJson(c, "密码过短")
		return
	}

	salt := util.GenerateDataId()
	passwd := form.Passwd + salt
	hashP := util.Sha256(passwd)

	user := &userapp.User{
		Id:        util.GenerateDataId(),
		Name:      form.LoginId,
		CreateT:   util.GetCurTime(),
	}
	user.UpdateT = user.CreateT

	ulpa := &userapp.UserLoginIdPasswdAuth{
		Id:      util.GenerateDataId(),
		UserId:  user.Id,
		LoginId: form.LoginId,
		Salt:    salt,
		Passwd:  hashP,
		Init: true,
		CreateT: user.CreateT,
		UpdateT: user.CreateT,
	}

	// op history
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取当前用户失败")
		return
	}

	opAction := fmt.Sprintf("管理员新建账户密码登录方式，loginId[%s]", form.LoginId)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	user.History = append(user.History, opHis)

	err = db.C(userapp.CollectionNameUser).Insert(user)
	if err != nil {
		Logger.Errorf(middleware.GetReqId(c), "注册账户[%s]失败, %s", form.LoginId, err.Error())
		returnfun.ReturnErrJson(c, err.Error())
		return
	}

	err = db.C(userapp.CollectionNameIdPasswd).Insert(ulpa)
	if err != nil {
		Logger.Errorf(middleware.GetReqId(c), "注册账户[%s]失败，%s", form.LoginId, err.Error())
		returnfun.ReturnErrJson(c, err.Error())
		return
	}
	ulpa.Passwd = ""
	user.IdPasswd = ulpa

	returnfun.ReturnOKJson(c, user)
	return
}

// 修改自己的密码
type UpdatePasswdForm struct {
	Passwd string `json:"passwd" binding:"required"`
}
func UpdatePasswdHandler(c *gin.Context, ro *UserOption) {
	var form UpdatePasswdForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	if len(form.Passwd) < 6 {
		returnfun.ReturnErrJson(c, "密码过短")
		return
	}

	curUser, _ := GetCurUserAndRole(c)
	userId := curUser.Id
	db := ro.Ds.CopyDs()
	defer db.Close()

	err = resetPasswd(db, ro.R, userId, form.Passwd)
	middleware.StopExec(err)

	// op history
	opAction := fmt.Sprintf("用户修改自己账户的登录密码")
	opHis := ophistory.NewOpHistory(userId, curUser.Name, opAction)

	updateOp := bson.M{
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(userapp.CollectionNameUser).UpdateId(userId, updateOp)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, "")
	return
}

// 重置或修改密码
// 修改成功后，要删除调用 redis 中存储的 token 信息
func resetPasswd(db *dbandmq.Ds, r *redis.Client, userId, newP string) error {
	salt := util.GenerateDataId()
	p := newP + salt
	hashP := util.Sha256(p)

	f := bson.M{
		"userId": userId,
	}

	update := bson.M{
		"$set": bson.M{
			"salt": salt,
			"passwd": hashP,
			"init": false,
			"updateT": util.GetCurTime(),
		},
	}

	err := db.C(userapp.CollectionNameIdPasswd).Update(f, update)
	if err != nil {
		Logger.Errorf("", "reset 用户[%s]passwd失败, %s", userId, err.Error())
		return err
	}

	// 删除token
	err = userapp.DeleteToken(r, userId)
	if err != nil {
		return err
	}

	Logger.Infof("", "reset 用户[%s]密码成功", userId)

	return nil
}

// 使用账户密码登录
type LoginIdPasswdForm struct {
	LoginId string `json:"loginId" binding:"required"`
	Passwd string `json:"passwd" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}
func LoginByIdPasswdHandler(c *gin.Context, uo *UserOption) {
	var form LoginIdPasswdForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := uo.Ds.CopyDs()
	defer db.Close()

	dbuser, err := userapp.GetUserByLoginId(db, form.LoginId)
	middleware.StopExec(err)

	if dbuser == nil {
		returnfun.Return401Json(c, "账户或密码错误")
		return
	}

	// 先检查是否 ban
	if dbuser.Ban {
		returnfun.Return401Json(c, "banned")
		return
	}

	// 检查密码是否一致
	tmpP := util.Sha256(form.Passwd + dbuser.IdPasswd.Salt)

	if tmpP != dbuser.IdPasswd.Passwd {
		returnfun.Return401Json(c, "账户或密码错误")
		return
	}
	dbuser.Platform = form.Platform

	// 检查一致，生成 token ，存储到数据库，返回用户token信息
	token, err := userapp.GenerateToken(dbuser.Id)
	middleware.StopExec(err)

	err = userapp.SaveToken(uo.R, token, dbuser)
	middleware.StopExec(err)

	// 记录登录信息

	retData := gin.H{
		"token": token,
		"user": dbuser,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 微信拉起授权
type LoginByWeChatForm struct {
	Code string `json:"code" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}
func LoginByWeChatHandler(c *gin.Context, uo *UserOption) {
	var form LoginByWeChatForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	platform := strings.ToUpper(form.Platform)

	wxOpt, ok := uo.WeChatOpt[platform]
	if !ok {
		Logger.Errorf(middleware.GetReqId(c), "发生严重错误，微信授权未做相关信息配置")
		returnfun.ReturnJson(c, 500, 500, "微信登录未配置相关信息", "")
		return
	}

	cf := userapp.GetWeChatConfig(uo.R, platform, wxOpt)
	wc := wechat.NewWechat(cf)

	wxOauth := wc.GetOauth()
	code := form.Code
	resToken, err := wxOauth.GetUserAccessToken(code)
	if err != nil {
		Logger.Errorf(middleware.GetReqId(c), "根据code[%s]读取access token失败, %s", code, err.Error())
		returnfun.Return401Json(c, err.Error())
		return
	}

	wxInfo, err := wxOauth.GetUserInfo(resToken.AccessToken, resToken.OpenID)
	if err != nil {
		Logger.Errorf(middleware.GetReqId(c), "根据accessToken[%s]和openId[%s]读取用户信息失败, %s", resToken.AccessToken, resToken.OpenID)
		returnfun.Return401Json(c, err.Error())
		return
	}

	// 新建或更新登录信息
	db := uo.Ds.CopyDs()
	defer db.Close()
	user, token, err := userapp.SaveWeChatLogin(db, uo.R, &wxInfo)
	if err != nil {
		returnfun.Return401Json(c, err.Error())
		return
	}

	if user.Ban {
		returnfun.Return401Json(c, "banned")
		return
	}

	retData := gin.H{
		"token": token,
		"user": user,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 手机号登录
// 发送验证码
type SendSmsForm struct {
	Phone string `json:"phone" binding:"required"`
}
func SendSmsHandler(c *gin.Context, uo *UserOption) {
	var form SendSmsForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	err = uo.PhoneOpt.SendSms(form.Phone, "", "")
	middleware.StopExec(err)

	if uo.PhoneOpt.Debug {
		code, _ := uo.R.Get(smsapp.PhoneRedisPrefix + form.Phone).Result()
		returnfun.ReturnOKJson(c, gin.H{"code": code})
		return
	}

	returnfun.ReturnOKJson(c, "")
	return
}

// 验证手机号
type CheckSmsForm struct {
	Phone string `json:"phone" binding:"required"`
	Code string `json:"code" binding:"required"`
}
func CheckSmsHandler(c *gin.Context, uo *UserOption) {
	var form CheckSmsForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	ok, err := uo.PhoneOpt.CheckSms(form.Phone, form.Code)
	middleware.StopExec(err)

	if !ok {
		returnfun.Return401Json(c, "验证码错误")
		return
	}

	// 新建或更新 phone 账户
	db := uo.Ds.CopyDs()
	defer db.Close()
	user, token, err := userapp.SavePhoneLogin(db, uo.R, form.Phone)
	middleware.StopExec(err)

	if user.Ban {
		returnfun.Return401Json(c, "banned")
		return
	}

	retData := gin.H{
		"token": token,
		"user": user,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 用户读取自己的信息，包含 role
func MeHandler(c *gin.Context, uo *UserOption) {
	user, roles := GetCurUserAndRole(c)

	retData := gin.H{
		"user": user,
		"roles": roles,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 退出登录
func LogoutHandler(c *gin.Context, uo *UserOption) {
	// 删除 token 即可
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取用户信息失败")
		return
	}
	err := userapp.DeleteToken(uo.R, curUser.Id)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, "")
	return
}

// 管理员创建 phone 账户
type CreateLoginPhoneForm struct {
	Phone string `json:"phone" binding:"required"`
}
func CreateLoginPhoneHandler(c *gin.Context, uo *UserOption) {
	var form CreateLoginPhoneForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	// 检查 phone 是否已存在，如果存在，返回存在的提示
	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserByPhone(db, form.Phone)
	middleware.StopExec(err)
	if user != nil {
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "phone已存在", gin.H{"id": user.Id})
		return
	}

	user, err = userapp.InitPhoneAuth(db, form.Phone)
	middleware.StopExec(err)

	// 记录 ophistory
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("管理员给手机号[%s]初始化账户", form.Phone)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	updateOp := bson.M{
		"$push": bson.M{
			"history": opHis,
		},
	}

	_ = db.C(userapp.CollectionNameUser).UpdateId(user.Id, updateOp)

	returnfun.ReturnOKJson(c, user)
	return
}

// 已登录微信情况下，绑定手机号
func AppendPhoneToWeChatHandler(c *gin.Context, uo *UserOption) {

}

// token 验证
type TokenCheckForm struct {
	Token string `json:"token" binding:"required"`
}
func TokenCheckHandler(c *gin.Context, uo *UserOption) {
	var form TokenCheckForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	type Ret struct {
		Valid bool `json:"valid"`
		Reason string `json:"reason"`
		User *userapp.User `json:"user"`
		Roles []*roleapp.Role `json:"roles"`
	}

	retData := &Ret{}

	tVal, err := userapp.CheckToken(uo.R, form.Token)
	if err != nil {
		retData.Valid = false
		retData.Reason = err.Error()
		returnfun.ReturnOKJson(c, retData)
		return
	}

	// 读取角色
	db := uo.Ds.CopyDs()
	defer db.Close()
	uwr, err := userandrole.GetUserRoles(db, tVal.User.Id)
	if err != nil {
		retData.Valid = false
		retData.Reason = err.Error()
		returnfun.ReturnOKJson(c, retData)
		return
	}

	retData.Valid = true
	retData.User = tVal.User
	retData.Roles = uwr.Roles

	returnfun.ReturnOKJson(c, retData)
	return
}

// 管理员封禁用户
type BanForm struct {
	UserId string `json:"userId" binding:"required"`
	Reason string `json:"reason"`
	T int64 `json:"t"` // 截至时间
}
func BanUserHandler(c *gin.Context, uo *UserOption) {
	var form BanForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserById(db, form.UserId)
	middleware.StopExec(err)

	if user == nil {
		returnfun.ReturnErrJson(c, "用户不存在")
		return
	}

	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取当前用户失败")
		return
	}

	if user.Id == curUser.Id || user.Id == userapp.AdminUserId {
		returnfun.ReturnErrJson(c, "不能禁用自己")
		return
	}

	t := time.Now().Unix() + 365 * 24 * 60 * 60
	if form.T > 0 && form.T > time.Now().Unix() {
		t = form.T
	}

	// op history
	opAction := fmt.Sprintf("封禁用户[%s][%s], reason[%s],到期时间[%d]", user.Id, user.Name, form.Reason, t)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	update := bson.M{
		"$set": bson.M{
			"ban": true,
			"banReason": form.Reason,
			"banT": t,
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(userapp.CollectionNameUser).UpdateId(user.Id, update)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, "")
	return
}

// 管理员解禁用户
type UnBanForm struct {
	UserId string `json:"userId" binding:"required"`
	Reason string `json:"reason"`
}
func UnBanUserHandler(c *gin.Context, uo *UserOption) {
	var form UnBanForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserById(db, form.UserId)
	middleware.StopExec(err)

	if user == nil {
		returnfun.ReturnErrJson(c, "用户不存在")
		return
	}

	// op history
	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取当前用户失败")
		return
	}
	opAction := fmt.Sprintf("解禁用户[%s][%s], reason[%s]", user.Id, user.Name, form.Reason)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	update := bson.M{
		"$set": bson.M{
			"ban": false,
			"banReason": form.Reason,
			"banT": 0,
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(userapp.CollectionNameUser).UpdateId(user.Id, update)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, "")
	return
}

// 管理员重置密码
type ResetPasswdForm struct {
	UserId string `json:"userId" binding:"required"`
}
func ResetPasswdHandler(c *gin.Context, uo *UserOption) {
	var form ResetPasswdForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserById(db, form.UserId)
	middleware.StopExec(err)

	if user == nil {
		returnfun.ReturnErrJson(c, "无指定id用户")
		return
	}

	if user.Id == userapp.AdminUserId {
		returnfun.Return403Json(c, "无权做此操作")
		return
	}

	// 直接 update，如果不存在 loginid 登录方式就会报错
	salt := util.GenerateDataId()
	passwd := smsapp.GenerateSmsCode(8)
	hashP := util.Sha256(passwd + salt)

	f := bson.M{
		"userId": form.UserId,
	}
	update := bson.M{
		"$set": bson.M{
			"salt": salt,
			"passwd": hashP,
			"init": true, // 这里要求重新登录修改密码
			"updateT": util.GetCurTime(),
		},
	}

	err = db.C(userapp.CollectionNameIdPasswd).Update(f, update)
	middleware.StopExec(err)

	// op history
	curUser, _ := GetCurUserAndRole(c)
	opAction := fmt.Sprintf("重置用户[%s][%s]的密码", user.Id, user.Name)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)

	updateOpHis := bson.M{
		"$set": bson.M{
			"updateT": util.GetCurTime(),
		},
		"$push": bson.M{
			"history": opHis,
		},
	}

	err = db.C(userapp.CollectionNameUser).UpdateId(user.Id, updateOpHis)
	middleware.StopExec(err)

	// 删除已经登录的 token
	_ = userapp.DeleteToken(uo.R, user.Id)

	retData := gin.H{
		"passwd": passwd,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 读取某个用户详细信息
func GetUserInfoHandler(c *gin.Context, uo *UserOption) {
	userId := c.Param("id")

	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserFullInfoById(db, userId)
	middleware.StopExec(err)

	if user == nil {
		returnfun.ReturnErrJson(c, "无指定id用户")
		return
	}

	// 权限
	uwr, err := userandrole.GetUserRoles(db, userId)
	middleware.StopExec(err)

	retData := gin.H{
		"user": user,
		"roles": uwr.Roles,
	}
	returnfun.ReturnOKJson(c, retData)
	return
}

// 搜素用户
func QueryUserHandler(c *gin.Context, uo *UserOption) {
	// 按登录 id / phone / wechat nickname 搜索用户
	// 上面只能选择一种来搜索
	var userIds []string

	db := uo.Ds.CopyDs()
	defer db.Close()

	hasArg := false

	loginId := c.Query("loginid")
	if loginId != "" {
		hasArg = true
		userIds = []string{}
		lps, err := userapp.QueryLoginIdAuthByLoginId(db, loginId)
		middleware.StopExec(err)
		for _, lp := range lps {
			userIds = append(userIds, lp.UserId)
		}
	}

	phone := c.Query("phone")
	if phone != "" {
		hasArg = true
		userIds = []string{}
		pas, err := userapp.QueryPhoneAuthByPhone(db, phone)
		middleware.StopExec(err)
		for _, pa := range pas {
			userIds = append(userIds, pa.UserId)
		}
	}

	nickname := c.Query("nickname")
	if nickname != "" {
		hasArg = true
		userIds = []string{}
		wcas, err := userapp.QueryWeChatAuthByNickname(db, nickname)
		middleware.StopExec(err)
		for _, wca := range wcas {
			userIds = append(userIds, wca.UserId)
		}
	}

	query := bson.M{}
	if hasArg {
		query = bson.M{
			"_id": bson.M{
				"$in": userIds,
			},
		}
	}

	Q := db.C(userapp.CollectionNameUser).Find(query)
	total, err := Q.Count()
	middleware.StopExec(err)

	page, size, skip := util.GetPageAndSize(c)
	var users []*userapp.User
	err = Q.Sort("-_id").Skip(skip).Limit(size).All(&users)
	middleware.StopExec(err)

	retData := gin.H{
		"total": total,
		"page": page,
		"size": size,
		"data": users,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}