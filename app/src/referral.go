package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// 邀请逻辑
// 在/invite 页面邀请人通过邀请朋友或朋友圈的方式
// 被邀请人打开 /join?referral=1111, 点加入
// 跳转至此 wecaht.wx.zhidaikeji.com/join?referrer=<openid>&referee=<openid>
// 此处读取 referrerID, referreeID
// 如果referree的parent_id为空, 则设置为referrer.id
func referralHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)

		cookie, err := r.Cookie("referree")
		if err != nil {
			fmt.Printf("Cant find cookie :/\r\n")
			return
		}
		log.Printf("%s=%s\r\n", cookie.Name, cookie.Value)

		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			log.Println(err)
			return
		}
		referrer := queryValues.Get("referrer")
		// referree := queryValues.Get("referree")
		referree := ""
		if cookie.Value != "" {
			referree = cookie.Value
		} else {
			referree = queryValues.Get("referree")
		}

		if referree == "" {
			// LOG TO DATABASE
			log.Fatalln(err)
			return
		}

		log.Println("referrer openid :" + referrer)
		log.Println("referree openid :" + referree)

		// 查找推荐人信息 referrer
		var referrerID int64
		var referrerOpenID string
		var referrerUnionID string
		var referrerParentID sql.NullInt64
		sqlStmt := `select id, openid, unionid, coalesce(parent_id, 0) from user_profiles where openid=$1 or unionid = $2;`
		row1 := db.QueryRow(sqlStmt, referrer, referrer)
		switch err := row1.Scan(&referrerID, &referrerOpenID, &referrerUnionID, &referrerParentID); err {
		case sql.ErrNoRows:
			log.Println("No rows were returned!")
		case nil:
			log.Println(referrerID, referrerOpenID, referrerUnionID, referrerParentID)
		default:
			panic(err)
		}

		// 查找被推荐人信息 referree
		var referreeID int64
		var referreeOpenID string
		var referreeUnionID string
		var referreeParentID int64
		// sqlStmt := `select id, openid, unionid from user_profiles where openid=$1 or unionid = $2;`
		row2 := db.QueryRow(sqlStmt, referree, referree)
		switch err := row2.Scan(&referreeID, &referreeOpenID, &referreeUnionID, &referreeParentID); err {
		case sql.ErrNoRows:
			log.Println("No rows were returned!")
		case nil:
			log.Println(referrerID, referrerOpenID, referrerUnionID, referreeParentID)
		default:
			panic(err)
		}

		log.Println("referrer id : ", referrerID)
		log.Printf("referree id : %d, referree parent_id %d \n", referreeID, referreeParentID)

		if referreeParentID == 0 {
			log.Println("设置referree 的parent_id 为 referrer 的 id 然后跳转到个人中心")
			//设置referree 的parent_id 为 referrer 的 id
			sqlStatement := `
                  UPDATE user_profiles
                  SET parent_id = $1
                  WHERE id = $2;`
			_, err = db.Exec(sqlStatement, referrerID, referreeID)
			if err != nil {
				panic(err)
			}
		}

		// http.SetCookie(w, &cookie)
		http.Redirect(w, r, "https://wx.zhidaikeji.com/#/profile", http.StatusFound)
	})
}
