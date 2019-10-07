// 微信网页授权流程
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
// https://github.com/chanxuehong/wechat/blob/master/mp/oauth2/README.md

package main

import (
	"database/sql"
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
func page2Handler(db *sql.DB) http.HandlerFunc {
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

		up := getUserProfileByOpenID(db, userinfo.OpenId)
		if (up == UserProfile{}) {
			id := 0
			sqlStmt2 := `INSERT INTO USER_PROFILES (openid, unionid, nickname, sex, city, province, country, headimgurl)
      VALUES($1, $2, $3, $4, $5, $6, $7, $8)
      RETURNING id`
			err := db.QueryRow(sqlStmt2, &userinfo.OpenId, &userinfo.UnionId, &userinfo.Nickname, &userinfo.Sex, &userinfo.City, &userinfo.Province, &userinfo.Country, &userinfo.HeadImageURL).Scan(&id)
			if err != nil {
				log.Println(err)
			}
			up.ID = id
			up.Openid = sql.NullString{String: userinfo.OpenId}
			up.Nickname = sql.NullString{String: userinfo.Nickname}
			up.Sex = sql.NullInt64{Int64: int64(userinfo.Sex)}
			up.City = sql.NullString{String: userinfo.City}
			up.Country = sql.NullString{String: userinfo.Country}
			up.Unionid = sql.NullString{String: userinfo.UnionId}
			up.HeadImageURL = sql.NullString{String: userinfo.HeadImageURL}
			// up.MobilePhone = sql.NullString{String: mobilePhone}
			// up.HasuraID = sql.NullInt64{Int64: hasuraID}
			// up.ParentID = sql.NullInt64{Int64: parentID}
			// up.Qrcode = sql.NullString{String: qrCode}
			log.Printf("\n\nuserinfo not exists in database, inserted it, id is %d\n\n", up.ID)
		} else {
			// UPDATE profile
			sqlStmt2 := `UPDATE USER_PROFILES
                    SET nickname=$1, sex=$2, city=$3, province=$4, country=$5, headimgurl=$6
                    WHERE openid=$7 AND unionid =$8`
			_, err = db.Exec(sqlStmt2, up.Nickname.String, up.Sex.Int64, up.City.String, up.Province.String, up.Country.String, up.HeadImageURL.String, up.Openid.String, up.Unionid.String)
			if err != nil {
				panic(err)
			}
			log.Printf("\n\nuserinfo exists in database, id is %d\n\n", up.ID)
			log.Printf("update it with nickname=%s, sex=%d, city=%s, province=%s, country=%s, headimgurl=%s, openid=%s, unionid=%s\n\n", up.Nickname.String, up.Sex.Int64, up.City.String, up.Province.String, up.Country.String, up.HeadImageURL.String, up.Openid.String, up.Unionid.String)
		}

		u, _ := json.Marshal(up)
		log.Printf("\n\nuserinfo to be encoded in cookie: %s\n\n", string(u))

		mpCookie := http.Cookie{
			Name:     "u",
			Value:    base64.StdEncoding.EncodeToString(u),
			Domain:   "wx.zhidaikeji.com",
			HttpOnly: false,
		}
		http.SetCookie(w, &mpCookie)
		http.Redirect(w, r, "https://wx.zhidaikeji.com/#/auth_callback", http.StatusFound)
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
