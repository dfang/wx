// 微信小程序登录态维护
// https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/login.html
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/login/auth.code2Session.html
// https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// code2Session wx.joi
func code2SessionHandler(w http.ResponseWriter, r *http.Request) {
	queryValues, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		io.WriteString(w, err.Error())
		log.Println(err)
		return
	}

	code := queryValues.Get("code")
	if code == "" {
		log.Println("用户禁止授权")
		return
	}

	// "https://api.weixin.qq.com/sns/jscode2session?appid=APPID&secret=SECRET&js_code=JSCODE&grant_type=authorization_code"
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", weappID, weappSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	// body, err := ioutil.ReadAll(resp.Body)

	var rsp Code2SessionResponse
	json.NewDecoder(resp.Body).Decode(&rsp)
}

// Code2SessionResponse 响应结构体
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/login/auth.code2Session.html
type Code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrorCode  int    `json:"errcode"`
	ErrorMsg   int    `json:"errmsg"`
}
