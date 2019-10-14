package userapp

import (
	"encoding/base64"
	"fmt"
	"github.com/go-redis/redis"
	jsoniter "github.com/json-iterator/go"
	. "github.com/leyle/ginbase/consolelog"
	"github.com/leyle/ginbase/util"
	"strconv"
	"strings"
	"time"
)

var AesKey = util.Md5("www.hbbclub.com") // 32 byte 使用加密方法就是 aes-256-cfb

// 生成 token
// 使用 aes-256-cfb 加密来生成 token
func GenerateToken(userId string) (string, error) {
	t := time.Now().Unix()
	text := fmt.Sprintf("%s|%d", userId, t)

	token, err := util.Encrypt([]byte(AesKey), text)
	if err != nil {
		Logger.Errorf("", "给用户[%s]生成token时，调用aes加密失败, %s", userId, err.Error())
		return "", err
	}

	// 在用 base64 编码
	b64Token := base64.StdEncoding.EncodeToString([]byte(token))

	return b64Token, nil
}

// 解析 token
func ParseToken(token string) (string, int64, error) {
	// 先 base64 解码
	de64Token, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		Logger.Errorf("", "base64解码token[%s]失败, %s", token, err.Error())
		return "", 0, err
	}

	// 再 aes 解密
	text, err := util.Decrypt([]byte(AesKey), string(de64Token))
	if err != nil {
		Logger.Errorf("", "aes解密token[%s]失败, %s", de64Token, err.Error())
		return "", 0, err
	}
	infos := strings.Split(text, "|")
	userId := infos[0]
	st := infos[1]

	t, _ := strconv.ParseInt(st, 10, 64)

	return userId, t, nil
}

// 存储token
// 存储为 key 是 userid， 值是 tokenvalue
func SaveToken(r *redis.Client, token string, user *User) error {
	tkVal := &TokenVal{
		Token: token,
		User:  user,
		T:     util.GetCurTime(),
	}

	tkDump, _ := jsoniter.Marshal(&tkVal)

	key := TokenRedisPrefix + user.Id
	_, err := r.Set(key, tkDump, 0).Result()
	if err != nil {
		Logger.Errorf("", "存储用户[%s]的token到redis失败, %s", user.Id, err.Error())
		return err
	}

	return nil
}

// 验证 token
func CheckToken(r *redis.Client, token string) (*TokenVal, error) {
	// 先解析 token
	userId, t, err := ParseToken(token)
	if err != nil {
		return nil, err
	}
	Logger.Debugf("", "CheckToken 时，parsetoken成功，用户[%s]，token生成时间[%s]", userId, util.FmtTimestampTime(t))

	// 从 redis 中读取 tokenval 信息
	key := TokenRedisPrefix + userId
	data, err := r.Get(key).Result()
	if err != nil {
		Logger.Errorf("", "CheckToken 时，从redis读取指定用户[%s]的tokenval失败, %s", userId, err.Error())
		return nil, err
	}

	var tkVal *TokenVal
	err = jsoniter.UnmarshalFromString(data, &tkVal)
	if err != nil {
		Logger.Errorf("", "CheckToken 时，反序列化从 redis 读取回来的用户[%s]的数据失败, %s", userId, err.Error())
		return nil, err
	}

	return tkVal, nil
}
