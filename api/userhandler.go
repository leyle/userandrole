package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/ginbase/middleware"
	"github.com/leyle/ginbase/returnfun"
	"github.com/leyle/ginbase/util"
	"github.com/leyle/userandrole/userapp"
	"gopkg.in/mgo.v2/bson"
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
		returnfun.ReturnErrJson(c, "账户已存在")
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

	retData := gin.H{
		"token": token,
		"user": dbuser,
	}

	returnfun.ReturnOKJson(c, retData)
	return
}