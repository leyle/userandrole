[TOC]

## 用户/角色/验证

管理员 指的是拥有相关角色的用户。

部分接口需要验证权限，需要在 header 设置 key 是 token，值是 token value 的数据，比如

setHeader("token", "someValue")

同一个登录方式只能同时在一个平台上有效登录，不支持同一个登录方式在多个平台登录。

---

### 数据字典

#### platform 可选值

| 值      | 意义           |
| ------- | -------------- |
| H5      | 手机浏览器登录 |
| PC      | 电脑浏览器登录 |
| ANDROID | 安卓手机       |
| IOS     | 苹果手机       |

---

#### loginType 可选值 

| 值       | 意义             |
| -------- | ---------------- |
| IDPASSWD | 账户密码登录     |
| EMAIL    | 邮件验证         |
| PHONE    | 手机号验证码验证 |
| WECHAT   | 微信授权         |
| QQ       | qq 授权          |

---

#### AuthResult 中的 result 值

| 值（整数） | 意义                                |
| ---------- | ----------------------------------- |
| 0          | token 错误/无效（401）              |
| 1          | token正确u，但无相关操作权限（403） |
| 9          | 验证成功，token有效，有对应的权限   |

---

### 用户(user)账户接口

#### 管理员创建一个账户密码登录方式的账户

```json
// POST /api/user/idpasswd
// roleIds 中的值为调用接口用户所包含的角色列表和 childrenRole 列表值
{
  "loginId": "testuser",
  "passwd": "abc123",
  "avatar": "http://some.com/avatar.jpg",
  "roleIds": ["aaaa", "bbbb", "cccc"]
}

// 注意这种方式创建的账户，下次登录时，需要修改密码
```

---

#### 用户修改自己的密码

```json
// POST /api/user/idpasswd/changepasswd
{
  "passwd": "some new passwd"
}
```

---

#### 已登录微信情况下，绑定手机号

```json
// 首先请求短信发送接口，发送短信验证码
// 然后调用本接口进行短信验证
// 调用本接口的前置条件是已经使用微信授权进行了登录

// 关于账户合并的一些规则

// 如果 phone 不存在，则把 phone 信息追加到当前账户体系下
// 如果 phone 存在，以 phone 为主，修改 wechatAuth 的 userid
// 标记原 user 数据为被合并的数据，标记为失效
// 所以，如果想要做绑定操作，并且有资源关联，那么最好的是拉起微信授权后，立刻绑定手机号
// 调用者获取到成功响应后，就应该重新拉去登录，因为之前的信息会被删除
// POST /api/user/wx/bindphone
{
  "phone": "13812345678",
  "code": "123456"
}
```

---

#### 用户读取自己的信息（包含登录信息和角色信息）

```json
// GET /api/user/me
```

---

#### 退出登录

```json
// POST/GET /api/user/logout
// 需要在登录状态下调用此接口
```

---

#### 管理员创建一个手机号登录账户

```json
// POST /api/user/phone
{
  "phone": "13812345678",
  "avatar": "http://some.com/avatar.jpg",
  "roleIds": ["aaaa", "bbbb", "cccc"]
}
```

---

#### 管理员封禁用户

```json
// POST /api/user/ban
// t 指的是封禁到期时间，精确到秒的时间戳
{
  "userId": "userid",
  "reason": "违反xxx规则",
  "t": 1571366536
}
```

---

#### 管理员解封用户

```json
// POST /api/user/unban
{
  "userId": "userid",
  "reason": "解封理由"
}
```

---

#### 管理员替用户重置密码

```json
// POST /api/user/idpasswd/resetpasswd
// 注意这里和用户自己修改密码的区别
{
  "userId": "userId"
}
```

---

#### 管理员读取某个用户的详细信息

```json
// GET /api/user/user/:id
// 路径最后是要读取的用户的 id
```

---

#### 管理员读取某个用户的登录历史记录

```json
// GET /api/user/loginhistory/:id?page=1
// 路径中的 id 指的是用户 id，page 指的是读取第几页的数据
// 单页返回记录为 10 条数据，返回数据中无 total 字段，根据返回的数据是否为空（或是否size==10）来判断是否读取完毕
```

---

#### 管理员搜索用户列表

```json
// GET /api/user/users
// 支持的 url 参数如下
// loginid - 登录id，支持部分匹配
// phone - 支持部分匹配
// nickname - 微信登录方式的 nickname，支持部分匹配
// 上述三个参数，只能同时一个生效

// page - 分页参数，从 1 开始
// size - 单页条数，默认 10
```

---

#### 读取微信 appid

```json
// GET /api/user/wx/appid?platform=H5
// platform 可选值 H5 / APP
```

---

#### 用户使用账户密码登录

```json
// POST /api/user/idpasswd/login
// platform 可选值  H5 / PC / ANDROID / IOS
{
  "loginId": "testauser",
  "passwd": "abc123456",
  "platform": "H5"
}

// 登录成功后，返回的用户信息中，如果 init 字段为 true，那么需要提示用户进行密码修改
```

---

#### 微信登录

```json
// POST /api/user/wx/login
// platform 可选值 H5 - 网页拉起微信授权 / APP - app 微信授权方式
{
  "code": "wechat code....",
  "platform": "H5"
}
```

---

#### 手机号注册登录

```json
// 手机号注册登录分为两个步骤
// 1、获取验证码
// 2、验证验证码有效性
// 如果用户未注册过，验证通过后注册账户

// 1、发送短信验证码
// POST /api/user/phone/sendsms
// 如果运行在 debug 模式，不会真发送短信，同时会返回 code
// 否则会真实发送短信验证码，仅返回发送成功的提示给调用者
{
  "phone": "13812345678"
}

// 2、验证验证码有效性（及同步创建账户，如果不存在的话）
// POST /api/user/phone/checksms
{
  "phone": "13812345678",
  "code": "123456",
  "platform": "H5"
}
```

---

#### token 有效性验证

```json
// POST /api/user/token/check
{
  "token": "some token value"
}

// 返回数据中，根据 valid 字段判断 token 是否有效，true - 有效，false - 无效，无效时，reason 可能有值
// 有效时，同步返回用户信息和角色信息
```

---

#### 验证用户是否具有某接口的权限 auth

```json
// POST /api/user/auth
// 本接口是提供给调用者一种验证机制，验证token对应的用户有无某接口权限
{
  "token": "some token value",
  "method": "GET",
  "path": "/api/course/info/abcdefghi"
}

// 返回的数据包含了三部分，指示了 token 是否有效，用户是否具有权限
// 如果验证成功，权限有效，同步返回用户信息和 role 信息
```



---

###角色(role)接口

角色由三部分组成，分别是 item、permission、role

- item 中定义了具体的api的 method、path，页面的 menu、button 等信息
- permission 是多个 item 组合起来的一个容器，也有自己的 menu\button 信息
- role 是多个 permission 组合起来的容器，也有自己的 menu\button 信息



path 支持一个通配符 `*`，比如接口为 `/api/user/:id`，配置时，就可以写成 `/api/user/*`即可。

---

#### 新建 item

```json
// POST /api/role/item
// name/method/path 为必输字段
// name 字段不可重复
{
  "name": "读取用户明细",
  "method": "GET",
  "path": "/api/user/*",
  "resource": "",
  "menu": "some menu",
  "button": "some button"
}
```

---

#### 修改 item / 取消删除item

```json
// PUT /api/role/item/:id
// 路径中的 id 指需要被修改的 item id
// 修改是一个全量操作，即使数据没有发生变化，也需要传递回来，否则会被置空
// name/method/path 为必输字段
// name 字段不可重复
// 如果原来的数据的状态是 deleted，修改后，就会取消删除状态
{
  "name": "读取用户明细",
  "method": "GET",
  "path": "/api/user/*",
  "resource": "",
  "menu": "some menu",
  "button": "some button"
}
```

---

#### 删除 item

```json
// DELETE /api/role/item/:id
// 路径中的 id 指的是需要被删除的 item id
// 删除是对数据做一个 deleted 标记
// 已删除的数据，可以使用 修改 item 接口重新上线
```

---

#### 读取 item 明细

```json
// GET /api/role/item/:id
// 路径中的 id 指的是需要读取信息的 item id
```

---

#### 搜索 item

```json
// GET /api/role/items
// 支持的 url 参数如下，这些参数可以同时传递
// name - 支持部分匹配
// path - 支持部分匹配
// method - 可选值为 http method，比如 GET POST 等
// menu - 支持部分匹配
// button - 支持部分匹配
// deleted - 是否删除，可选值为 true/false

// page - 分页，从 1 开始
// size - 单页条数，默认 10

```

---

#### 新建 permission

```json
// POST /api/role/permission
// name 为必输参数
// itemIds 指的是 item 的 id，本接口中可以选择输入，也可以不输入
{
  "name": "管理用户基本信息",
  "itemIds": ["ida", "idb"],
  "menu": "xxx",
  "button": "yyy"
}
```

---

#### 给 permission 添加多个 items

```json
// POST /api/role/permission/:id/additems
// 路径中 id 指的是被修改的 permission 的 id
// itemIds 指的是要添加的 item 的 id 列表
{
  "itemIds": ["ida", "idb"]
}
```

---

#### 取消 permission 的某些 items

```json
// POST /api/role/permission/:id/delitems
// 路径中 id 指的是被修改的 permission 的 id
// itemIds 指的是要添加的 item 的 id 列表
{
  "itemIds": ["ida", "idb"]
}
```

---

#### 修改 permission 基本信息

```json
// PUT /api/role/permission/:id
// 修改的是除了包含的 item id 外的其他信息
{
  "name": "some name",
  "menu": "some menu",
  "button": "some button"
}
```

---

#### 删除 permission

```json
// DELETE /api/role/permission/:id
// 删除是标记操作
// 已经删除的数据可以通过 修改 permission 基本信息 接口再重新上线
```

---

#### 读取 permission 明细

```json
// GET /api/role/permission/:id
```

---

#### 搜索 permission 列表

```json
// GET /api/role/permissions
// 支持的 url 参数如下
// 以下参数支持同时传递生效
// name - 支持部分匹配
// menu - 支持部分匹配
// button - 支持部分匹配
// deleted - 是否删除，可选值为 true/false

// page - 当前页，默认 1
// size - 单页条数，默认 10
```

---

#### 新建 role

```json
// POST /api/role/role
// name 为必填，不可重复，其他为选填
{
  "name": "role name",
  "pids": ["pid1", "pid2"],
  "menu": "some menu",
  "button": "some button"
}
```

---

#### 给 role 添加 permission

```json
// POST /api/role/role/:id/addps
// 路径中 id 指的是 role id
// 可以同时添加多个 permission， pids 指的是 permission 的 id
{
  "pids": ["pid1", "pid2"]
}
```

---

#### 取消 role 的 permission

```json
// POST /api/role/role/:id/delps
// 路径中的 id 指的是 role id
// 可以同时删除多个 permission
{
  "pids": ["pid1", "pid2"]
}
```

---

#### 修改 role 信息

```json
// PUT /api/role/role/:id
// 路径中的 id 指的是 role id
{
  "name": "role name",
  "menu": "some munu",
  "button": "some button"
}
```

---

#### 删除 role 

```json
// DELETE /api/role/role/:id
// 删除是标记操作
// 已经删除的数据可以通过 修改 role信息 接口再重新上线
```

---

#### 给 role 添加 childrole

```json
// POST /api/role/:id/addchildrole
// 此处添加的子角色，是为了辅助用户创建账户时，方便给账户赋予角色，赋予的角色只能是此处的角色
// :id 指的是目标 role id
{
  "childrenRole": [
    {
      "id": "abccc",
      "name": "角色1"
    },
    {
      "id": "iddd2",
      "name": "角色2"
    }
  ]
}
```

---

#### 移除 role 的某些 childrole

```json
// POST /api/user/:id/delchildrole
// 移除此 :id 角色包含的 children role
{
  "childrenRole": [
    {
      "id": "abccc",
      "name": "角色1"
    },
    {
      "id": "iddd2",
      "name": "角色2"
    }
  ]
}
```

---

#### 查看 role 信息

```json
// GET /api/role/role/:id
```

---

####  搜索 role

```json
// GET /api/role/roles
// 支持的 url 参数如下
// 以下参数支持同时传递生效
// name - 支持部分匹配
// menu - 支持部分匹配
// button - 支持部分匹配
// deleted - 是否删除，可选值为 true/false

// page - 当前页，默认 1
// size - 单页条数，默认 10
```



---

### 用户 - 角色关联接口

管理用户和角色的映射关系



---

#### 给 user 添加 roles

```json
// POST /api/uwr/addroles
// userId 与 roleIds 为必填
{
  "userId": "userid",
  "userName": "some user name",
  "avatar": "user avatar url",
  "roleIds": ["roleid1", "roleid2"]
}
```

---

#### 取消 user 的某些 roles

```json
// POST /api/uwr/delroles
// 两个参数都是必输
{
  "userId": "userid",
  "roleIds": ["roleid1", "roleid2"]
}
```

---

#### 读取指定 user 的 roles

```json
// GET /api/uwr/user/:id
// 路径中的 id 指的是 userid
```

---

#### 搜索已经添加 role 的用户列表

```json
// GET /api/uwr/users
// 仅支持 page size 参数
```



---

### 程序接入与验证方法

AuthOpton 结构体

```go
type Option struct {
	R   *redis.Client
	Mgo *dbandmq.MgoOption
	db  *dbandmq.Ds // 临时存放，使用完毕会销毁
}

var AuthOption = &auth.Option{} // 调用本包，需要给这个变量赋值
```



---

#### gin 框架接入方法

1、初始化数据库连接需要的参数信息，给 AuthOption 变量赋值

2、直接调用 Auth(c *gin.Context) 即可

---

#### 其他程序调用方法

如果想要更加详细的调用，第一步也是给 AuthOption 变量赋值

然后调用 AuthLoginAndRole(ao *Option, token, method, uri, resource string)

根据返回的 AuthResult 结构自行判断。

```go
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
```

