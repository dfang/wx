package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	mpCore "github.com/chanxuehong/wechat/mp/core"
	jssdk "github.com/chanxuehong/wechat/mp/jssdk"
	util "github.com/chanxuehong/wechat/util"
)

// mp.weixin.qq.com 设置授权回调域名
func mpVerifyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "YxLL1ZR1olEyFcp0")
}

// SignatureHandler 微信JS SDK生成签名
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115
// GET /jssdk_signature?url= + encodeURIComponent(window.location.href)
// will send back
// {
//   nonceStr: "9f6bfeacc652054f74ff85bc1260335a",
//   timestamp: "20190329225349",
//   signature: "a51c9db2e3a9a3dbab50b0888679d4ff6701b35e"
// }
func jssdkSignatureHandler(w http.ResponseWriter, r *http.Request) {
	nonceStr := util.NonceStr()
	timestamp := time.Now().Format("20060102150405")
	url := r.URL.Query().Get("url")
	log.Println("url to sign :", url)

	srv := mpCore.NewDefaultAccessTokenServer(wxAppID, wxAppSecret, nil)
	client := mpCore.NewClient(srv, nil)

	ticketServer := jssdk.NewDefaultTicketServer(client)
	ticket, err := ticketServer.Ticket()

	if err != nil {
		log.Println(err)
		log.Println("获取 jsapi_ticket 出错")
	}
	log.Println("ticket: ", ticket)

	signature := jssdk.WXConfigSign(ticket, nonceStr, timestamp, url)
	log.Println("signature: ", signature)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(struct {
		AppID     string `json:"appId"`
		NonceStr  string `json:"nonceStr"`
		Timestamp string `json:"timestamp"`
		Signature string `json:"signature"`
	}{
		AppID:     wxAppID,
		NonceStr:  nonceStr,
		Timestamp: timestamp,
		Signature: signature,
	})
}
