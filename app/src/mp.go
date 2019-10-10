// 微信网页授权流程
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
// https://github.com/chanxuehong/wechat/blob/master/mp/oauth2/README.md

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/sid"
	mpoauth2 "github.com/chanxuehong/wechat/mp/oauth2"
	"github.com/chanxuehong/wechat/oauth2"
)

// 建立必要的 session, 然后跳转到授权页面
func page1Handler(w http.ResponseWriter, r *http.Request) {
	sid := sid.New()
	state := string(rand.NewHex())

	if err := sessionStorage.Add(sid, state); err != nil {
		io.WriteString(w, err.Error())
		log.Println(err)
		return
	}

	cookie := http.Cookie{
		Name:     "sid",
		Value:    sid,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	AuthCodeURL := mpoauth2.AuthCodeURL(wxAppID, oauth2RedirectURI, oauth2Scope, state)
	log.Println("AuthCodeURL:", AuthCodeURL)
	http.Redirect(w, r, AuthCodeURL, http.StatusFound)
}

// 授权后回调页面
func page2Handler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)

		DumpHTTPRequest(r)

		cookie, err := r.Cookie("sid")
		if err != nil {
			io.WriteString(w, err.Error())
			log.Println(err)
			return
		}

		session, err := sessionStorage.Get(cookie.Value)
		if err != nil {
			io.WriteString(w, err.Error())
			log.Println(err)
			return
		}

		savedState := session.(string) // 一般是要序列化的, 这里保存在内存所以可以这么做

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

		queryState := queryValues.Get("state")
		if queryState == "" {
			log.Println("state 参数为空")
			return
		}
		if savedState != queryState {
			str := fmt.Sprintf("state 不匹配, session 中的为 %q, url 传递过来的是 %q", savedState, queryState)
			io.WriteString(w, str)
			log.Println(str)
			return
		}

		oauth2Client := oauth2.Client{
			Endpoint: oauth2Endpoint,
		}
		token, err := oauth2Client.ExchangeToken(code)
		if err != nil {
			io.WriteString(w, err.Error())
			log.Println(err)
			return
		}
		log.Printf("token: %+v\r\n", token)

		userinfo, err := mpoauth2.GetUserInfo(token.AccessToken, token.OpenId, "", nil)
		if err != nil {
			io.WriteString(w, err.Error())
			log.Println(err)
			return
		}

		// json.NewEncoder(w).Encode(userinfo)
		log.Printf("\n\nuserinfo: %+v\r\n", userinfo)
		// return

		// user := WechatMPAuth{}
		// json.NewDecoder(r.Body).Decode(&user)
		// log.Printf("%v", user)

		u, _ := json.Marshal(userinfo)
		log.Printf("\n\nuserinfo to be encoded in cookie: %s\n\n", string(u))

		mpCookie := http.Cookie{
			Name:  "u",
			Value: base64.StdEncoding.EncodeToString(u),
			// Domain:   "xsjd123.com",
			Domain:   cookieDomain,
			HttpOnly: false,
		}
		http.SetCookie(w, &mpCookie)
		http.Redirect(w, r, wxMpRedirectURL, http.StatusFound)
		return
	})
}

// type UserProfile struct {
// 	UserInfo *mpoauth2.UserInfo `json:",omitempty"`
// 	ID       int64              `json:",omitempty"`
// }

// func (u UserProfile) MarshalJSON() ([]byte, error) {
// 	type Alias UserProfile

// 	return json.Marshal(&struct {
// 		Openid     string `json:"openid"`
// 		Unionid     string `json:"unionid"`
// 		Nickname   string `json:"nickname"`
// 		Sex        int    `json:"sex"`
// 		City       string `json:"city"`
// 		Province   string `json:"province"`
// 		Country    string `json:"country"`
// 		Headimgurl string `json:"headimgurl"`
// 		ID         int64  `json:"id"`
// 	}{
// 		u.UserInfo.OpenId,
// 		u.UserInfo.Nickname,
// 		u.UserInfo.Sex,
// 		u.UserInfo.City,
// 		u.UserInfo.Province,
// 		u.UserInfo.Country,
// 		u.UserInfo.HeadImageURL,
// 		u.ID,
// 	})
// }
