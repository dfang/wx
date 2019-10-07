package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// try to implement Hasura auth provider
// wechat authorization like github

func loginHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fmt.Fprintln(w, "hello login")
		user := WechatMPAuth{}
		json.NewDecoder(r.Body).Decode(&user)
		log.Printf("%v", user)

		sqlStmt := `SELECT ID FROM USER_PROFILES WHERE OPENID=$1`
		id := 0
		err := db.QueryRow(sqlStmt, &user.Data.Openid).Scan(&id)
		if err != nil {
			log.Println(err)
		}
		switch err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned!")
			sqlStmt2 := `INSERT INTO USER_PROFILES (openid, unionid, nickname, sex, city, province, country, headimgurl)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id`

			err := db.QueryRow(sqlStmt2, &user.Data.Openid, &user.Data.Unionid, &user.Data.Nickname, &user.Data.Sex, &user.Data.City, &user.Data.Province, &user.Data.Country, &user.Data.Headimgurl).Scan(&id)
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

		json.NewEncoder(w).Encode(struct {
			UserID int `json:"user_id"`
		}{UserID: id})
	})
}

func signUpHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// {
		//   "provider": "wechat_mp",
		//   "data": {
		//               "openid":"oeNgh1eT9ABsjezl82HWnNpJQGq4",
		//               "nickname":"Fang","sex":1,"city":"武汉","province":"湖北","country":"中国",
		//               "headimgurl":"http://thirdwx.qlogo.cn/mmopen/vi_32/PiajxSqBRaEIcGGiaXIFH4JIEnbNHLOIbzdAtYHUTLSc8zvQujpxfHJW9a9ycMwUcs19NIu5heSQEH6kCTcia3O8g/132"
		//           }
		// }
		user := WechatMPAuth{}
		json.NewDecoder(r.Body).Decode(&user)
		log.Printf("%v", user)

		sqlStmt := `SELECT ID FROM USER_PROFILES WHERE OPENID=$1`
		id := 0
		err := db.QueryRow(sqlStmt, &user.Data.Openid).Scan(&id)
		if err != nil {
			log.Println(err)

		}
		switch err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned!")
			sqlStmt2 := `INSERT INTO USER_PROFILES (openid, unionid, nickname, sex, city, province, country, headimgurl)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id`

			err := db.QueryRow(sqlStmt2, &user.Data.Openid, &user.Data.Unionid, &user.Data.Nickname, &user.Data.Sex, &user.Data.City, &user.Data.Province, &user.Data.Country, &user.Data.Headimgurl).Scan(&id)
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

		json.NewEncoder(w).Encode(struct {
			UserID int `json:"user_id"`
		}{UserID: id})
	})
}

func mergeHandler(w http.ResponseWriter, r *http.Request) {
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
}
