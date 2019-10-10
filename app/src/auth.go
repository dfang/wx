package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// try to implement custom auth provider
func loginHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// {
		//   "openid":"oeNgh1eT9ABsjezl82HWnNpJQGq4",
		//   "unionid":"oeNgh1eT9ABsjezl82HWnNpJQGq4",
		//   "nickname":"Fang",
		//   "sex":1,
		//   "city":"武汉",
		//   "province":"湖北",
		//   "country":"中国",
		//   "headimgurl":"http://thirdwx.qlogo.cn/mmopen/vi_32/PiajxSqBRaEIcGGiaXIFH4JIEnbNHLOIbzdAtYHUTLSc8zvQujpxfHJW9a9ycMwUcs19NIu5heSQEH6kCTcia3O8g/132"
		// }

		c, err := r.Cookie("u")
		if err != nil {
			w.Write([]byte("error in reading cookie : " + err.Error() + "\n"))
		}

		// Base64 Standard Decoding
		sDec, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			fmt.Printf("Error decoding string: %s ", err.Error())
			return
		}

		user := WechatMPAuth{}
		err = json.Unmarshal([]byte(sDec), &user)
		// json.NewDecoder(r.Body).Decode(&user)
		log.Printf("cookie decoded as string: %v", user)

		sqlStmt := `SELECT ID FROM WECHAT_PROFILES WHERE UNIONID=$1`
		id := 0
		err = db.QueryRow(sqlStmt, &user.UnionID).Scan(&id)
		if err != nil {
			log.Println(err)
		}
		switch err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned!")
			sqlStmt2 := `INSERT INTO WECHAT_PROFILES (openid, unionid, nickname, sex, city, province, country, headimgurl)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id`
			err := db.QueryRow(sqlStmt2, &user.OpenID, &user.UnionID, &user.Nickname, &user.Sex, &user.City, &user.Province, &user.Country, &user.Headimgurl).Scan(&id)
			if err != nil {
				log.Println(err)
			}
			fmt.Println("New record ID is:", id)
			fmt.Fprint(w, struct {
				UserID int
			}{UserID: id})
			return
		default:
			log.Println(err)
		}

		var mobile sql.NullString
		sqlStmt3 := `SELECT mobile_phone FROM WECHAT_PROFILES WHERE UNIONID=$1`
		err = db.QueryRow(sqlStmt3, &user.UnionID).Scan(&mobile)
		// sql: Scan error on column index 0, name "mobile_phone": unsupported Scan, storing driver.Value type <nil> into type *string
		if err != nil {
			log.Println(err)
		}

		if !mobile.Valid {
			// redirect to bind mobile phone page
			http.Redirect(w, r, "http://mp.xsjd123.com/#/pages/bindPhone/index", http.StatusUnauthorized)
			return
		}

		userID := 0
		sqlStmt4 := `SELECT id FROM users WHERE mobile_phone=$1`
		err = db.QueryRow(sqlStmt4, &user.UnionID).Scan(&userID)
		if err != nil {
			log.Println(err)
		}

		au := AuthInfo{
			WechatMPAuth: user,
			MobilePhone:  mobile.String,
			UserID:       strconv.Itoa(userID),
		}

		// json.NewEncoder(w).Encode(struct {
		// 	UserID int `json:"user_id"`
		// }{UserID: id})

		by, err := json.Marshal(au)
		if err != nil {
			log.Println(err)
		}

		auCookie := http.Cookie{
			Name:     "i",
			Value:    base64.StdEncoding.EncodeToString(by),
			Domain:   wxMpAuthDomain,
			HttpOnly: false,
		}
		http.SetCookie(w, &auCookie)
		http.Redirect(w, r, "/xxxxx", http.StatusFound)
		return
	})
}
