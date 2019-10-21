package userapp

import (
	"github.com/leyle/ginbase/dbandmq"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	userId := "5da41d400ce239748629d9d3"

	token, err := GenerateToken(userId, LoginTypeIdPasswd)
	if err != nil {
		t.Error(err)
	}

	t.Log(token)

	pid, _, pt, err := ParseToken(token)
	t.Log(pid)
	t.Log(pt)
}

func TestSaveToken(t *testing.T) {
	ro := &dbandmq.RedisOption{
		Host:   "192.168.100.233",
		Port:   "6380",
		Passwd: "56grTbvMYaOQ",
		DbNum:  14,
	}
	r, err := dbandmq.NewRedisClient(ro)
	if err != nil {
		t.Error(err)
	}

	userId := "5da41d400ce239748629d9d3"

	token, err := GenerateToken(userId, LoginTypeIdPasswd)
	if err != nil {
		t.Error(err)
	}
	t.Log(token)

	idpasswd := &UserLoginIdPasswdAuth{
		Id:      "5da41d400ce239748629d9d1",
		UserId:  "5da41d400ce239748629d9d3",
		LoginId: "test",
	}

	user := &User{
		Id:        userId,
		Name:      "test",
		Platform:  "WEB",
		LoginType: "IDPASSWD",
		IdPasswd:  idpasswd,
		Ip:        "192.168.100.188",
	}

	err = SaveToken(r, token, user)
	if err != nil {
		t.Error(err)
	}

	t.Log("OK")
}

func TestCheckToken(t *testing.T) {
	ro := &dbandmq.RedisOption{
		Host:   "192.168.100.233",
		Port:   "6380",
		Passwd: "56grTbvMYaOQ",
		DbNum:  14,
	}
	r, err := dbandmq.NewRedisClient(ro)
	if err != nil {
		t.Error(err)
	}

	token := "QVJNRERxQ3ZQT1pGeXlYU3YtMndMTlJfQ0lkaUZkTzZzUURPbTZTaFNGYlpTN3RzLXdyUXRwV0lzb211ZU1ZbWFDbmhDckVqSGJJNHd3eldGZG82c3c="
	tk, err := CheckToken(r, token)
	if err != nil {
		t.Error(err)
	}

	t.Log(tk.T)
	t.Log(tk.Token)
	t.Log(tk.User)
}