package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/chanxuehong/session"
	mpoauth2 "github.com/chanxuehong/wechat/mp/oauth2"
	"github.com/chanxuehong/wechat/oauth2"
	"github.com/gorilla/mux"
)

var (
	// flagPort is the open port the application listens on
	flagPort = flag.String("port", "8080", "Port to listen on")
)

// WechatMPAuth Auth Provider
type WechatMPAuth struct {
	Provider string `json:"provider"`
	Data     struct {
		Openid     string `json:"openid"`
		Unionid    string `json:"unionid"`
		Nickname   string `json:"nickname"`
		Sex        int    `json:"sex"`
		City       string `json:"city"`
		Province   string `json:"province"`
		Country    string `json:"country"`
		Headimgurl string `json:"headimgurl"`
	} `json:"data"`
}

type signSignatureRequest struct {
	nonceStr  string
	timestamp string
	url       string
}

//go:generate xo pgsql://@localhost/hasuradb -o models --template-path templates

// const (
// // mchID  = "1518844551"                       // 微信支付商户ID
// // apiKey = "haY7TtuAoLszKsLwhAqPioNvYha53dfa" // 微信支付商户APIKEY
// )

var (
	wxAppID     = os.Getenv("WX_MP_APPID")
	wxAppSecret = os.Getenv("WX_MP_APPSECRET")

	wxMpRedirectURL = os.Getenv("WX_MP_AUTH_REDIRECT_URL")

	// 网页授权域名
	wxMpAuthDomain = os.Getenv("WX_MP_AUTH_DOMAIN")
	// JS接口安全域名
	wxMpJSDomain = os.Getenv("WX_MP_JS_DOMAIN")
	// 业务域名
	wxMpBizDomain = os.Getenv("WX_MP_BIZ_DOMAIN")

	wxMpVerifyURL     = os.Getenv("WX_MP_MPVERIFY_URL")
	wxMpVerifyContent = os.Getenv("WX_MP_MPVERIFY_CONTENT")

	mchID  = os.Getenv("WX_PAY_MCHID")
	apiKey = os.Getenv("WX_PAY_APIKEY")
)

var (
	// wxAppID           = "wxb7e6db75bccd7c53"                           // 公众号appID
	// wxAppSecret       = "67c34a2b5a088330c354ef7ce09ab06a"             // 填上自己的参数
	oauth2RedirectURI = fmt.Sprintf("https://%s/page2", wxMpAuthDomain) // 填上自己的参数
	oauth2Scope       = "snsapi_userinfo"                               // 填上自己的参数
)

var (
	sessionStorage                 = session.New(20*60, 60*60)
	oauth2Endpoint oauth2.Endpoint = mpoauth2.NewEndpoint(wxAppID, wxAppSecret)
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

// Initialize intialize database, routes
func (a *App) Initialize(user, password, host, dbName string) {
	for _, e := range []string{"WX_MP_APPID", "WX_MP_APPSECRET", "WX_MP_MPVERIFY_URL", "WX_MP_MPVERIFY_CONTENT", "WX_PAY_MCHID", "WX_PAY_APIKEY"} {
		if os.Getenv(e) == "" {
			log.Fatalf("请正确设置环境变量 \n可以通过hasura secret update来设置 \n 然后在k9s deployment.yaml 通过secretKeyRef读取secret到Docker容器里\n")
		}
	}

	// // host = "localhost"
	// // user = "postgres"
	// // password = "postgres"
	// // dbname := "mj"
	// port := 5432
	// user = os.Getenv("POSTGRES_USER")
	// password = os.Getenv("POSTGRES_PASSWORD")
	// host = "postgres.hasura"
	// dbname := "hasuradb"

	// psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
	// 	"password=%s dbname=%s sslmode=disable",
	// 	host, port, user, password, dbname)

	// // why parseTime=true
	// // error: "sql: Scan error on column index 7: null: cannot scan type []uint8 into null.Time: [50 48 49 56 45 48 52 45 49 52 32 49 51 58 52 56 58 48 52]"
	// // https://github.com/xo/xo/issues/19
	// // connectionString := fmt.Sprintf("%s:%s@%s/%s?parseTime=true", user, password, host, dbName)
	// log.Printf("connectionString is %s\n", psqlInfo)

	// var err error
	// a.DB, err = sql.Open("postgres", psqlInfo)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // defer a.DB.Close()

	// err = a.DB.Ping()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("Successfully connected!")

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) initializeRoutes() {
	r := a.Router

	// 开始请求网页授权
	r.HandleFunc("/page1", page1Handler)

	r.HandleFunc("/auth", page1Handler)

	// 获取用户信息然后回调
	r.HandleFunc("/page2", page2Handler())

	// 生成调用wx jssdk需要的签名等
	r.HandleFunc("/jssdk_signature", jssdkSignatureHandler)

	// wx.ChooseWxPay 前调用统一下单
	// 返回wx.chooseWxPay 需要的参数中的prepay_id
	r.HandleFunc("/unifiedorder", unifiedOrderHandler)
	// 返回wx.chooseWxPay 需要的参数
	r.HandleFunc("/jssdk_paysign", jsSdkPaySignHandler)
	// wx.ChooseWxPay成功之后的回调
	r.HandleFunc("/pay_notify", paymentNotifyHandler)

	// // wx.ChooseWxPay 的success callback 最好查询一下
	// r.HandleFunc("/order_query", orderQueryHandler(a.DB))

	// r.HandleFunc("/join", referralHandler(a.DB))

	// 公众号设置 -> 功能设置 -> 设置JS接口安全域名 和 网页授权域名 需要的验证
	// r.HandleFunc("/MP_verify_w7yalxZBScxCceA2.txt", mpVerifyHandler)
	r.HandleFunc(wxMpVerifyURL, mpVerifyHandler)

	// r.HandleFunc("/v1/login", loginHandler(a.DB))
	// r.HandleFunc("/v1/signup", signUpHandler(a.DB))
}

func main() {
	a := App{}
	a.Initialize(os.Getenv("APP_DB_USER"), os.Getenv("APP_DB_PASSWORD"), os.Getenv("APP_DB_HOST"), os.Getenv("APP_DB_NAME"))

	log.Printf("listening on port %s", *flagPort)
	log.Fatal(http.ListenAndServe(":"+*flagPort, a.Router))
}
