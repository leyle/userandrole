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
	"math"
	"strings"
	"time"
)

// 通过账户密码数据，没有自己注册的，都是通过接口创建的
type CreateIdPasswdForm struct {
	LoginId string `json:"loginId" binding:"required"`
	Passwd string `json:"passwd" binding:"required"`
	Avatar string `json:"avatar"` // 头像，非必输
	RoleIds []string `json:"roleIds"` // 角色列表，非必输，此处选择的角色只能是当前用户的自身或下属角色，api 管理员不受此规则的控制
}
func CreateLoginIdPasswdAccountHandler(c *gin.Context, ro *UserOption) {
	var form CreateIdPasswdForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	form.LoginId = strings.TrimSpace(form.LoginId)

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
		Avatar: form.Avatar,
		CreateT:   util.GetCurTime(),
	}
	user.UpdateT = user.CreateT

	ulpa := &userapp.UserLoginIdPasswdAuth{
		Id:      util.GenerateDataId(),
		UserId:  user.Id,
		LoginId: form.LoginId,
		Avatar: form.Avatar,
		Salt:    salt,
		Passwd:  hashP,
		Init: true,
		SelfReg: false,
		CreateT: user.CreateT,
		UpdateT: user.CreateT,
	}

	curUser, curRoles := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取当前用户失败")
		return
	}

	// 检查是否有赋予 role 的信息，如果有，需要检查是否有权限赋予相应的权限
	if len(form.RoleIds) > 0 {
		if !shareRoleIsValid(curUser, curRoles, form.RoleIds) {
			returnfun.Return403Json(c, "当前用户无权赋予用户某些权限")
			return
		}
	}

	// op history
	opAction := fmt.Sprintf("新建账户密码登录方式，loginId[%s]", form.LoginId)
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
	user.IdPasswd = ulpa

	// 如果有 roleids 信息，同步赋予
	if len(form.RoleIds) > 0 {
		_, err = addRoleToUser(db, curUser, user.Id, form.RoleIds)
		middleware.StopExec(err)
	}

	returnfun.ReturnOKJson(c, user)
	return
}

// 检查当前用户是否有权限操作指定的 roleIds
func shareRoleIsValid(curUser *userapp.User, curRoles []*roleapp.Role, roleIds []string) bool {
	if curUser.Id == userapp.AdminUserId {
		return true
	}

	var validRoleIds []string
	for _, curRole := range curRoles {
		validRoleIds = append(validRoleIds, curRole.Id)
		for _, cr := range curRole.ChildrenRoles {
			validRoleIds = append(validRoleIds, cr.Id)
		}
	}
	validRoleIds = util.UniqueStringArray(validRoleIds)

	findR := func(rid string) bool {
		for _, r := range validRoleIds {
			if r == rid {
				return true
			}
		}
		return false
	}

	for _, rid := range roleIds {
		if !findR(rid) {
			return false
		}
	}

	return true
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
	err = userapp.DeleteToken(r, userId, userapp.LoginTypeIdPasswd)
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

	// 检查 platform 值的有效性
	if !userapp.IsValidPlatform(form.Platform) {
		returnfun.ReturnErrJson(c, "错误的 platform 值")
		return
	}

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
	dbuser.LoginType = userapp.LoginTypeIdPasswd

	// 读取用户角色
	uwr, err := userandrole.GetUserRoles(db, dbuser.Id)
	middleware.StopExec(err)

	// 检查一致，生成 token ，存储到数据库，返回用户token信息
	token, err := userapp.GenerateToken(dbuser.Id, userapp.LoginTypeIdPasswd)
	middleware.StopExec(err)

	err = userapp.SaveToken(uo.R, token, dbuser)
	middleware.StopExec(err)

	// 记录登录信息
	lh := &ophistory.LoginHistory{
		Id:        util.GenerateDataId(),
		UserId:    dbuser.Id,
		UserName:  dbuser.Name,
		LoginType: userapp.LoginTypeIdPasswd,
		Platform:  form.Platform,
		Ip:        c.Request.RemoteAddr,
		UserAgent: c.Request.UserAgent(),
		LoginT:    util.GetCurTime(),
	}
	_ = ophistory.SaveLoginHistory(db, lh)

	retData := gin.H{
		"token": token,
		"user": dbuser,
		"roles": uwr.Roles,
		"childrenRole": uwr.ChildrenRole,
		"menus": uwr.Menus,
		"buttons": uwr.Buttons,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 读取微信 appid
func GetWeChatAppIdHandler(c *gin.Context, uo *UserOption) {
	platform := c.Query("platform")
	if platform == "" {
		returnfun.ReturnErrJson(c, "缺少 platform 值")
		return
	}
	platform = strings.ToUpper(platform)
	opt, ok := uo.WeChatOpt[platform]
	if !ok {
		returnfun.ReturnErrJson(c, "错误的 platform 值")
		return
	}

	retData := gin.H{
		"platform": platform,
		"appId": opt.AppId,
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

	// 读取用户角色
	uwr, err := userandrole.GetUserRoles(db, user.Id)
	middleware.StopExec(err)

	// 保存登录成功的信息
	lh := &ophistory.LoginHistory{
		Id:        util.GenerateDataId(),
		UserId:    user.Id,
		UserName:  user.Name,
		LoginType: userapp.LoginTypeWeChat,
		Platform:  form.Platform,
		Ip:        c.Request.RemoteAddr,
		UserAgent: c.Request.UserAgent(),
		LoginT:    util.GetCurTime(),
	}
	_ = ophistory.SaveLoginHistory(db, lh)

	retData := gin.H{
		"token": token,
		"user": user,
		"roles": uwr.Roles,
		"childrenRole": uwr.ChildrenRole,
		"menus": uwr.Menus,
		"buttons": uwr.Buttons,
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
	Platform string `json:"platform" binding:"required"`
}
func CheckSmsHandler(c *gin.Context, uo *UserOption) {
	var form CheckSmsForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	if !userapp.IsValidPlatform(form.Platform) {
		returnfun.ReturnErrJson(c, "错误的 platform 值")
		return
	}

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

	// 读取用户角色信息
	uwr, err := userandrole.GetUserRoles(db, user.Id)
	middleware.StopExec(err)


	// 保存登录信息
	lh := &ophistory.LoginHistory{
		Id:        util.GenerateDataId(),
		UserId:    user.Id,
		UserName:  user.Name,
		LoginType: userapp.LoginTypePhone,
		Platform:  form.Platform,
		Ip:        c.Request.RemoteAddr,
		UserAgent: c.Request.UserAgent(),
		LoginT:    util.GetCurTime(),
	}
	_ = ophistory.SaveLoginHistory(db, lh)

	retData := gin.H{
		"token": token,
		"user": user,
		"roles": uwr.Roles,
		"childrenRole": uwr.ChildrenRole,
		"menus": uwr.Menus,
		"buttons": uwr.Buttons,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}

// 用户读取自己的信息，包含 role
func MeHandler(c *gin.Context, uo *UserOption) {
	user, roles := GetCurUserAndRole(c)
	childrenRoles := userandrole.UnWrapChildrenRole(roles)

	retData := gin.H{
		"user": user,
		"roles": roles,
		"childrenRole": childrenRoles,
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
	err := userapp.DeleteToken(uo.R, curUser.Id, curUser.LoginType)
	middleware.StopExec(err)
	returnfun.ReturnOKJson(c, "")
	return
}

// 管理员创建 phone 账户
type CreateLoginPhoneForm struct {
	Phone string `json:"phone" binding:"required"`
	Avatar string `json:"avatar"`
	RoleIds []string `json:"roleIds"` // 角色列表，非必输，此处选择的角色只能是当前用户的自身或下属角色，api 管理员不受此规则的控制
}
func CreateLoginPhoneHandler(c *gin.Context, uo *UserOption) {
	var form CreateLoginPhoneForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	// 检查手机号格式的正确性 todo

	// 检查是否有赋予 role 的信息，如果有，需要检查是否有权限赋予相应的权限
	curUser, curRoles := GetCurUserAndRole(c)
	if len(form.RoleIds) > 0 {
		if !shareRoleIsValid(curUser, curRoles, form.RoleIds) {
			returnfun.Return403Json(c, "当前用户无权赋予用户某些权限")
			return
		}
	}

	// 检查 phone 是否已存在，如果存在，返回存在的提示
	db := uo.Ds.CopyDs()
	defer db.Close()

	user, err := userapp.GetUserByPhone(db, form.Phone)
	middleware.StopExec(err)
	if user != nil {
		returnfun.ReturnJson(c, 400, ErrCodeNameExist, "phone已存在", gin.H{"id": user.Id})
		return
	}

	user, err = userapp.InitPhoneAuth(db, form.Phone, form.Avatar)
	middleware.StopExec(err)

	// 记录 ophistory
	opAction := fmt.Sprintf("管理员给手机号[%s]初始化账户", form.Phone)
	opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
	updateOp := bson.M{
		"$push": bson.M{
			"history": opHis,
		},
	}

	_ = db.C(userapp.CollectionNameUser).UpdateId(user.Id, updateOp)

	// 如果有 roleids 信息，同步赋予
	if len(form.RoleIds) > 0 {
		_, err = addRoleToUser(db, curUser, user.Id, form.RoleIds)
		middleware.StopExec(err)
	}

	returnfun.ReturnOKJson(c, user)
	return
}

// 已登录微信情况下，绑定手机号
// 处理逻辑
// 如果 phone 不存在，则 phoneAuth 存储的 userid 使用当前登录账户的
// 如果 phone 存在，以 phone 为主，修改 wechatAuth 的 userid
// 标记原 user 数据为被合并的数据，标记为失效
// 调用者获取到成功响应后，就应该重新拉去登录，因为之前的信息会被删除
type WeChatBindPhoneForm struct {
	Phone string `json:"phone" binding:"required"`
	Code string `json:"code" binding:"required"`
}
func WeChatBindPhoneHandler(c *gin.Context, uo *UserOption) {
	var form WeChatBindPhoneForm
	err := c.BindJSON(&form)
	middleware.StopExec(err)

	curUser, _ := GetCurUserAndRole(c)
	if curUser == nil {
		returnfun.ReturnErrJson(c, "获取当前用户信息失败")
		return
	}

	if curUser.LoginType != userapp.LoginTypeWeChat {
		returnfun.ReturnErrJson(c, "当前登录类型不是微信授权")
		return
	}

	// 验证短信有效性
	ok, err := uo.PhoneOpt.CheckSms(form.Phone, form.Code)
	middleware.StopExec(err)

	if !ok {
		returnfun.ReturnErrJson(c, "验证码错误")
		return
	}

	// 验证通过，检查手机号是否存在
	db := uo.Ds.CopyDs()
	defer db.Close()

	phoneUser, err := userapp.GetUserByPhone(db, form.Phone)
	middleware.StopExec(err)
	if phoneUser == nil {
		// 账户不存在，直接创建一个 userId 是当前微信号所属的数据
		ph := &userapp.PhoneAuth{
			Id:      util.GenerateDataId(),
			UserId:  curUser.Id,
			Phone:   form.Phone,
			Init:    false,
			SelfReg: true,
			CreateT: util.GetCurTime(),
		}
		ph.UpdateT = ph.CreateT

		err = db.C(userapp.CollectionNamePhone).Insert(ph)
		middleware.StopExec(err)

		// op history
		opAction := fmt.Sprintf("微信绑定手机号[%s]时，手机号不存在，创建新的 phoneAuth，并关联到当前用户下", form.Phone)
		opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opAction)
		update := bson.M{
			"$set": bson.M{
				"updateT": util.GetCurTime(),
			},
			"$push": bson.M{
				"history": opHis,
			},
		}

		err = db.C(userapp.CollectionNameUser).UpdateId(curUser.Id, update)
		middleware.StopExec(err)
	} else {
		// 以 phone 为主，迁移微信登录信息
		// 1. 更新原微信 userId
		userId := phoneUser.Id
		updateA := bson.M{
			"$set": bson.M{
				"userId": userId,
				"updateT": util.GetCurTime(),
			},
		}
		fA := bson.M{
			"userId": curUser.Id,
		}
		err = db.C(userapp.CollectionNameWeChat).Update(fA, updateA)
		middleware.StopExec(err)

		// 2. 更新原 wechat user 信息
		opActionB := fmt.Sprintf("微信绑定手机号[%s]，手机账户已存在，本账户就被禁用", form.Phone)
		opHis := ophistory.NewOpHistory(curUser.Id, curUser.Name, opActionB)

		updateB := bson.M{
			"$set": bson.M{
				"referId": phoneUser.Id,
				"ban": true,
				"banT": math.MaxInt64,
				"banReason": userapp.CombineAccountBanReason,
				"updateT": util.GetCurTime(),
			},
			"$push": bson.M{
				"history": opHis,
			},
		}
		err = db.C(userapp.CollectionNameUser).UpdateId(curUser.Id, updateB)
		middleware.StopExec(err)

		// 3. 更新 phone user 信息
		opActionC := fmt.Sprintf("微信[%s]绑定手机号[%s]，从原账户[%s]迁移过来微信信息", curUser.WeChatAuth.OpenId, form.Phone, curUser.Id)
		opHis = ophistory.NewOpHistory(curUser.Id, curUser.Name, opActionC)

		updateC := bson.M{
			"$set": bson.M{
				"updateT": util.GetCurTime(),
			},
			"$push": bson.M{
				"history": opHis,
			},
		}

		err = db.C(userapp.CollectionNameUser).UpdateId(phoneUser.Id, updateC)
		middleware.StopExec(err)
	}
	returnfun.ReturnOKJson(c, "")
	return
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
		ChildrenRole []*roleapp.ChildRole `json:"childrenRole"`
		Menus []string `json:"menus"`
		Buttons []string `json:"buttons"`
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
	retData.ChildrenRole = uwr.ChildrenRole
	retData.Menus = uwr.Menus
	retData.Buttons = uwr.Buttons

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
	_ = userapp.DeleteToken(uo.R, user.Id, userapp.LoginTypeIdPasswd)

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
		"menus": uwr.Menus,
		"buttons": uwr.Buttons,
	}
	returnfun.ReturnOKJson(c, retData)
	return
}

// 读取某个用户的登录历史记录
func GetUserLoginHistoryHandler(c *gin.Context, uo *UserOption) {
	userId := c.Param("id")
	db := uo.Ds.CopyDs()
	defer db.Close()

	page, _, _ := util.GetPageAndSize(c)

	lhs, err := ophistory.GetLoginHistoryByUserId(db, userId, page)
	middleware.StopExec(err)

	returnfun.ReturnOKJson(c, lhs)
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