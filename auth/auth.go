package auth

import (
	"errors"
	"github.com/go-redis/redis"
	"github.com/leyle/ginbase/dbandmq"
	"github.com/leyle/userandrole/roleapp"
	"github.com/leyle/userandrole/userandrole"
	"github.com/leyle/userandrole/userapp"
	. "github.com/leyle/ginbase/consolelog"
	"regexp"
	"strings"
)

// 登录信息有效性验证与角色权限验证
type Option struct {
	R   *redis.Client
	Ds *dbandmq.Ds
	db  *dbandmq.Ds // 临时存放，使用完毕会销毁
}

func (ao *Option) new() *Option {
	db := ao.Ds.CopyDs()
	newAo := &Option{
		R:   ao.R,
		Ds: ao.Ds,
		db:  db,
	}

	return newAo
}

func (ao *Option) close() {
	if ao.db != nil {
		ao.db.Close()
	}
}

// 用户验证成功，返回的值
const (
	AuthResultInValidToken = 0 // token 错误，比如用户名或密码错误
	AuthResultInValidRole = 1 // role 不对，无对应的操作权限
	AuthResultOK = 9 // 验证成功
)
type AuthResult struct {
	Result int             `json:"result"` // 验证结果，见上面字典
	User   *userapp.User   `json:"user"`   // 用户信息
	Roles  []*roleapp.Role `json:"roles"`   // 角色信息
}

func NewAuthResult() *AuthResult {
	return &AuthResult{}
}

var NoPermission = errors.New("无当前资源权限")

// resource 可以为空，为空时不校验
func AuthLoginAndRole(ao *Option, token, method, uri, resource string) *AuthResult {
	newAo := ao.new()
	defer newAo.close()

	ar := NewAuthResult()

	// 验证 token
	user, err := AuthToken(newAo, token)
	if err != nil {
		ar.Result = AuthResultInValidToken
		return ar
	}
	ar.User = user

	// 验证权限
	roles, err := AuthRole(newAo, user.Id, method, uri, resource)
	if err != nil {
		if err == NoPermission {
			ar.Roles = roles
		}
		ar.Result = AuthResultInValidRole
		return ar
	}

	ar.Roles = roles
	ar.Result = AuthResultOK

	return ar
}

// 验证 token
// token 有效时，返回 user 信息
func AuthToken(ao *Option, token string) (*userapp.User, error) {
	tkVal, err := userapp.CheckToken(ao.R, token)
	if err != nil {
		Logger.Errorf("", "AuthToken 时，token验证失败, %s", err.Error())
		return nil, err
	}

	return tkVal.User, nil
}

// 验证权限
func AuthRole(ao *Option, userId, method, uri, resource string) ([]*roleapp.Role, error) {
	userWithRoles, err := userandrole.GetUserRoles(ao.db, userId)
	if err != nil {
		Logger.Errorf("", "读取用户[%s]roles失败, %s", userId, err.Error())
		return nil, err
	}

	if len(userWithRoles.Roles) == 0 {
		return userWithRoles.Roles, NoPermission
	}

	// 检查权限，path 支持通配符，这里需要支持
	items := roleapp.UnWrapRoles(userWithRoles.Roles)
	if !hasPermission(items, method, uri, resource) {
		return userWithRoles.Roles, NoPermission
	}

	return userWithRoles.Roles, nil
}

func hasPermission(items []*roleapp.Item, method, path, resource string) bool {
	if len(items) == 0 {
		return false
	}

	// 按照 method 分组 key 是 method， value 是 uri 的列表
	infos := make(map[string][]string)
	for _, item := range items {
		tm := item.Method
		if val, ok := infos[tm]; ok {
			val = append(val, item.Path)
		} else {
			infos[tm] = []string{item.Path}
		}
	}

	uris, ok := infos[method]
	if !ok {
		// 连方法都不存在，直接就是 false
		return false
	}

	for _, uri := range uris {
		// 数据库保存的 uri 支持一个 * 通配符
		if uri == "*" {
			return true
		}

		// 包含通配符，需要正则校验
		if strings.Contains(uri, "*") {
			uri = strings.ReplaceAll(uri, "*", "\\w+")
			uri := "^" + uri + "$"
			re, err := regexp.Compile(uri)
			if err != nil {
				Logger.Errorf("", "检查用户权限时，系统配置错误，无法 compile 正则表达式, %s", err.Error())
				return false
			}
			match := re.MatchString(path)
			if match {
				return true
			}

		} else {
			// 否则直接对比
			if uri == path {
				return true
			}
		}
	}

	return false
}


