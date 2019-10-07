package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	mchCore "github.com/chanxuehong/wechat/mch/core"
	"github.com/chanxuehong/wechat/mch/pay"
	util "github.com/chanxuehong/wechat/util"
)

// 统一下单接口
func unifiedOrderHandler(w http.ResponseWriter, r *http.Request) {
	DumpHTTPRequest(r)

	// var q pay.UnifiedOrderRequest
	var q map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		log.Printf("Error decoding body: %s", err)
		return
	}
	printRequestHeaders(r)

	// log.Println(q)
	log.Printf("\n用户请求参数 %+v\n\n", q)
	// b, _ := json.Marshal(q)
	// log.Println(b)
	ip := "127.0.0.1"
	client := mchCore.NewClient(wxAppID, mchID, apiKey, nil)
	req := pay.UnifiedOrderRequest{
		Body:       q["body"].(string),
		OutTradeNo: q["out_trade_no"].(string),
		TotalFee:   int64(q["total_fee"].(float64)),
		TradeType:  "JSAPI",
		DeviceInfo: "WEB",
		// NotifyURL:      q["notify_url"].(string),
		NotifyURL: "https://wechat.wx.zhidaikeji.com/pay_notify",
		// SpbillCreateIP: q["spbill_create_ip"].(string),
		SpbillCreateIP: ip,
		OpenId:         q["openid"].(string),
	}

	log.Printf("调用UnifiedOrder时的参数 %+v\n\n", req)

	resp, err := pay.UnifiedOrder2(client, &req)
	if err != nil {
		// https://pay.weixin.qq.com/wiki/doc/api/jsapi.php?chapter=9_1
		// https://godoc.org/gopkg.in/chanxuehong/wechat.v2/mch/pay#UnifiedOrderRequest
		log.Printf("Error when unified order: %s\n\n", err)
	}

	log.Printf("UnifiedOrder 返回的 Response: %+v\n\n", resp)

	// result, _ := json.Marshal(resp)
	// log.Printf("%+v\n", result)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	json.NewEncoder(w).Encode(resp)
}

func paymentNotifyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("payment success notify callback")

	bodyBuffer, _ := ioutil.ReadAll(r.Body)
	log.Println(string(bodyBuffer))
	w.Write([]byte("SUCCESS"))
}

// wx.chooseWXPay 发送一个支付请求, 需要一些参数
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115
// wx.chooseWXPay({
//   timestamp: 0, // 支付签名时间戳，注意微信jssdk中的所有使用timestamp字段均为小写。但最新版的支付后台生成签名使用的timeStamp字段名需大写其中的S字符
//   nonceStr: '', // 支付签名随机串，不长于 32 位
//   package: '', // 统一支付接口返回的prepay_id参数值，提交格式如：prepay_id=\*\*\*）
//   signType: '', // 签名方式，默认为'SHA1'，使用新版支付需传入'MD5'
//   paySign: '', // 支付签名
//   success: function (res) {
//   // 支付成功后的回调函数
//   }
// });
// package 通过统一下单接口拿到的
// paySign 采用统一的微信支付 Sign 签名生成方法
// timestamp, nonceStr, signType 直接后台生成就行了, 无需前端生成
// 前端只需要传package
func jsSdkPaySignHandler(w http.ResponseWriter, r *http.Request) {
	// timestamp: 0, // 支付签名时间戳，注意微信jssdk中的所有使用timestamp字段均为小写。但最新版的支付后台生成签名使用的timeStamp字段名需大写其中的S字符
	// nonceStr: '', // 支付签名随机串，不长于 32 位
	// package: '', // 统一支付接口返回的prepay_id参数值，提交格式如：prepay_id=\*\*\*）
	// signType: '', // 签名方式，默认为'SHA1'，使用新版支付需传入'MD5'
	// paySign: '', // 支付签名
	var q paySignResponse
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		log.Printf("Error decoding body: %s", err)
		return
	}
	q.NonceStr = util.NonceStr()
	q.Timestamp = time.Now().Format("20060102150405")
	q.SignType = "MD5"
	// packageStr := ""

	q.PaySign = mchCore.JsapiSign(wxAppID, q.Timestamp, q.NonceStr, q.PackageStr, q.SignType, apiKey)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	json.NewEncoder(w).Encode(&q)
}

type paySignResponse struct {
	Timestamp  string `json:"timestamp"`
	NonceStr   string `json:"nonceStr"`
	PackageStr string `json:"package"`
	SignType   string `json:"signType"`
	PaySign    string `json:"paySign"`
}

func orderQueryHandler(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			log.Printf("Error decoding body: %s", err)
			return
		}

		client := mchCore.NewClient(wxAppID, mchID, apiKey, nil)
		req := pay.OrderQueryRequest{
			OutTradeNo: q["out_trade_no"].(string),
		}
		resp, err := pay.OrderQuery2(client, &req)
		if err != nil {
			// https://github.com/chanxuehong/wechat/blob/v2/mch/pay/orderquery.go
			log.Printf("Error when order query: %s\n\n", err)
		}
		log.Printf("Order Query 返回的 Response: %+v\n\n", resp)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST")

		// {
		//   Attach: ""
		//   BankType: "CFT"
		//   CashFee: 1
		//   CashFeeType: ""
		//   Detail: ""
		//   DeviceInfo: "WEB"
		//   FeeType: "CNY"
		//   IsSubscribe: true
		//   OpenId: "oeNgh1eT9ABsjezl82HWnNpJQGq4"
		//   OutTradeNo: "20190405120612"
		//   SettlementTotalFee: null
		//   SubIsSubscribe: null
		//   SubOpenId: ""
		//   TimeEnd: "2019-04-05T00:06:19+08:00"
		//   TotalFee: 1
		//   TradeState: "SUCCESS"
		//   TradeStateDesc: "支付成功"
		//   TradeType: "JSAPI"
		//   TransactionId: "4200000294201904056922854164"
		// }
		if resp.TradeState == "SUCCESS" {
			u := getUserProfileByOpenID(db, resp.OpenId)
			levelID, _ := strconv.Atoi(resp.Attach)
			if levelID == 0 {
				levelID = 1
			}
			setMembership(db, u.ID, levelID)
		}
		json.NewEncoder(w).Encode(resp)
	})
}

type UserProfile struct {
	ID           int            `json:"id"`
	Nickname     sql.NullString `json:"nickname"`
	Sex          sql.NullInt64  `json:"sex"`
	City         sql.NullString `json:"city"`
	Province     sql.NullString `json:"province"`
	Country      sql.NullString `json:"country"`
	MobilePhone  sql.NullString `json:"mobile_phone"`
	Openid       sql.NullString `json:"openid"`
	Unionid      sql.NullString `json:"unionid"`
	HasuraID     sql.NullInt64  `json:"hasura_id"`
	ParentID     sql.NullInt64  `json:"parent_id"`
	HeadImageURL sql.NullString `json:"headimageurl"`
	Qrcode       sql.NullString `json:"qrcode"`
}

func getUserProfileByOpenID(db *sql.DB, openid string) UserProfile {
	u := UserProfile{}
	sqlStmt := `select id, coalesce(nickname, ''), coalesce(sex, 0), coalesce(city, ''), coalesce(province, ''), coalesce(country, ''), coalesce(headimgurl, ''), coalesce(mobile_phone, ''), coalesce(openid, ''), coalesce(unionid, ''), coalesce(hasura_id, 0), coalesce(parent_id, 0), coalesce(qrcode, '') from user_profiles where openid=$1;`
	row := db.QueryRow(sqlStmt, openid)
	switch err := row.Scan(&u.ID, &u.Nickname, &u.Sex, &u.City, &u.Province, &u.Country, &u.HeadImageURL, &u.MobilePhone, &u.Openid, &u.Unionid, &u.HasuraID, &u.ParentID, &u.Qrcode); err {
	case sql.ErrNoRows:
		log.Println("No rows were returned!")
	case nil:
		log.Printf("%+v\n", u)
	default:
		panic(err)
	}
	return u
}

func setMembership(db *sql.DB, userProfileID int, membershipLevelsID int) bool {
	var id int
	sqlstmt := `SELECT id FROM memberships WHERE membership_levels_id = $1 AND user_profiles_id = $2;`
	row := db.QueryRow(sqlstmt, membershipLevelsID, userProfileID)
	switch err := row.Scan(&id); err {
	case sql.ErrNoRows:
		log.Println("No rows were returned!")
	case nil:
		log.Println(id)
	default:
		panic(err)
	}

	// 已经是会员的情况下可以升级, 更改记录
	if id > 0 {
		sqlStatement1 := `UPDATE memberships SET membership_levels_id = $1 WHERE user_profiles_id = $2;`
		fmt.Sprintln(sqlStatement1, membershipLevelsID, userProfileID)
		_, err := db.Exec(sqlStatement1, membershipLevelsID, userProfileID)
		if err != nil {
			panic(err)
		}
	} else {
		// 开通会员创建记录
		sqlStatement2 := `INSERT INTO memberships (membership_levels_id, user_profiles_id) values($1, $2);`
		fmt.Sprintln(sqlStatement2, membershipLevelsID, userProfileID)
		_, err := db.Exec(sqlStatement2, membershipLevelsID, userProfileID)
		if err != nil {
			panic(err)
		}
	}

	return true
}

func (u UserProfile) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID          int64  `json:"id"`
		OpenID      string `json:"openid"`
		Nickname    string `json:"nickname"`
		Sex         int    `json:"sex"`
		City        string `json:"city"`
		Province    string `json:"province"`
		Country     string `json:"country"`
		Headimgurl  string `json:"headimgurl"`
		HasuraID    int64  `json:"hasura_id"`
		ParentID    int64  `json:"parent_id"`
		MobilePhone string `json:"mobile_phone"`
		UnionID     string `json:"unionid"`
	}{
		ID:          int64(u.ID),
		OpenID:      u.Openid.String,
		Nickname:    u.Nickname.String,
		Sex:         int(u.Sex.Int64),
		City:        u.City.String,
		Province:    u.Province.String,
		Country:     u.Country.String,
		Headimgurl:  u.HeadImageURL.String,
		HasuraID:    u.HasuraID.Int64,
		ParentID:    u.ParentID.Int64,
		UnionID:     u.Unionid.String,
		MobilePhone: u.MobilePhone.String,
	})
}
