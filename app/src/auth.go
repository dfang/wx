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
		log.Printf("cookie decoded as string: %+v", user)

		sqlStmt := `SELECT ID FROM WECHAT_PROFILES WHERE UNIONID=$1`
		id := 0
		err = db.QueryRow(sqlStmt, &user.UnionID).Scan(&id)
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
		fmt.Printf("SELECT mobile_phone FROM WECHAT_PROFILES WHERE UNIONID=%s\n", user.UnionID)
		err = db.QueryRow(sqlStmt3, user.UnionID).Scan(&mobile)
		// sql: Scan error on column index 0, name "mobile_phone": unsupported Scan, storing driver.Value type <nil> into type *string
		if err != nil {
			log.Println(err)
		}

		if !mobile.Valid {
			// redirect to bind mobile phone page
			http.Redirect(w, r, "http://mp.xsjd123.com/#/pages/bindPhone/index", http.StatusSeeOther)
			return
		}

		fmt.Println("moible_phone: ", mobile.String)

		var userID sql.NullInt64
		sqlStmt4 := `SELECT id FROM users WHERE mobile_phone=$1`
		fmt.Printf("SELECT id FROM users WHERE mobile_phone=%s\n", mobile.String)
		err = db.QueryRow(sqlStmt4, mobile.String).Scan(&userID)
		if err != nil {
			log.Println(err)
		}

		if !userID.Valid {
			// http.Redirect(w, r, "http://mp.xjsd123.com/", http.StatusFound)
			w.Write([]byte(`请联系客客服申请售后资质`))
			return
		}

		fmt.Println("user_id: ", userID.Int64)

		au := AuthInfo{
			WechatMPAuth: user,
			MobilePhone:  mobile.String,
			UserID:       strconv.FormatInt(userID.Int64, 10),
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
			Domain:   "xsjd123.com",
			HttpOnly: false,
		}
		http.SetCookie(w, &auCookie)
		http.Redirect(w, r, "http://mp.xsjd123.com/", http.StatusFound)
		return
	})
}

// try to implement custom auth provider
func bindPhoneHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// https://stackoverflow.com/questions/15672556/handling-json-post-request-in-go
		// 绑定手机号码

		DumpHTTPRequest(r)

		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields() // catch unwanted fields
		// anonymous struct type: handy for one-time use
		t := struct {
			MobilePhone *string `json:"mobile_phone"` // pointer so we can test for field absence
			CookieValue *string `json:"cookie"`
		}{}
		err := d.Decode(&t)
		if err != nil {
			// bad JSON or unrecognized json field
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if t.MobilePhone == nil {
			http.Error(w, "missing field 'mobile_phone' from JSON object", http.StatusBadRequest)
			return
		}

		if t.CookieValue == nil {
			http.Error(w, "missing field 'cookie' from JSON object", http.StatusBadRequest)
			return
		}

		// mobilePhone := r.PostFormValue("mobile_phone")
		// cookieValue := r.PostFormValue("cookie")

		mobilePhone := *t.MobilePhone
		cookieValue := *t.CookieValue

		// Base64 Standard Decoding
		sDec, err := base64.StdEncoding.DecodeString(cookieValue)
		if err != nil {
			fmt.Printf("Error decoding string: %s ", err.Error())
			return
		}

		user := WechatMPAuth{}
		err = json.Unmarshal([]byte(sDec), &user)
		// json.NewDecoder(r.Body).Decode(&user)
		log.Printf("cookie decoded as string: %v", user)

		sqlStmt3 := `UPDATE WECHAT_PROFILES SET MOBILE_PHONE=$1 WHERE UNIONID=$2`
		res, err := db.Exec(sqlStmt3, mobilePhone, user.UnionID)
		if err != nil {
			panic(err)
		}
		count, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}
		fmt.Println(count)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"status\": \"ok\"}"))
		// http.Redirect(w, r, "http://mp.xjsd123.com/", http.StatusFound)
		return
	})
}
