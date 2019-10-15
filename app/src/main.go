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
	"github.com/chanxuehong/wechat/mp/core"
	mpoauth2 "github.com/chanxuehong/wechat/mp/oauth2"
	"github.com/chanxuehong/wechat/oauth2"
	"github.com/gorilla/mux"

	"github.com/rs/cors"
)

var (
	// flagPort is the open port the application listens on
	flagPort = flag.String("port", "8080", "Port to listen on")
)

// WechatMPAuth Auth Provider
type WechatMPAuth struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	City       string `json:"city"`
	Province   string `json:"province"`
	Country    string `json:"country"`
	Headimgurl string `json:"headimgurl"`
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

	// 设置网页授权域名，微信需要验证授权域名 （公众号设置 -> 功能设置 -> 网页授权域名)
	wxMpVerifyURL     = os.Getenv("WX_MP_MPVERIFY_URL")
	wxMpVerifyContent = os.Getenv("WX_MP_MPVERIFY_CONTENT")
	// 网页授权域名  只能设置一个 （公众号设置 -> 功能设置)
	wxMpAuthDomain = os.Getenv("WX_MP_AUTH_DOMAIN")

	// 微信授权之后获取到用户的微信信息了，跳转到哪
	wxMpRedirectURL = os.Getenv("WX_MP_AUTH_REDIRECT_URL")

	// // JS接口安全域名 可设置多个  （公众号设置 -> 功能设置)
	// wxMpJSDomain = os.Getenv("WX_MP_JS_DOMAIN")
	// // 业务域名 可设置多个 （公众号设置 -> 功能设置)
	// wxMpBizDomain = os.Getenv("WX_MP_BIZ_DOMAIN")

	// 微信支付相关
	mchID  = os.Getenv("WX_PAY_MCHID")
	apiKey = os.Getenv("WX_PAY_APIKEY")

	cookieDomain = os.Getenv("WX_MP_COOKIE_DOMAIN")

	accessTokenServer core.AccessTokenServer = core.NewDefaultAccessTokenServer(wxAppID, wxAppSecret, nil)
	// wechatClient      *core.Client           = core.NewClient(accessTokenServer, nil)
)

var (
	// wxAppID           = "wxb7e6db75bccd7c53"                           // 公众号appID
	// wxAppSecret       = "67c34a2b5a088330c354ef7ce09ab06a"             // 填上自己的参数
	oauth2RedirectURI = fmt.Sprintf("http://%s/page2", wxMpAuthDomain) // 填上自己的参数
	oauth2Scope       = "snsapi_userinfo"                              // 填上自己的参数
)

var (
	sessionStorage                 = session.New(20*60, 60*60)
	oauth2Endpoint oauth2.Endpoint = mpoauth2.NewEndpoint(wxAppID, wxAppSecret)
)

// App app struct
type App struct {
	Router *mux.Router
	DB     *sql.DB
}

// Initialize intialize database, routes
func (a *App) Initialize(user, password, host, dbName string) {
	for _, e := range []string{
		"WX_MP_APPID", "WX_MP_APPSECRET",
		"WX_MP_MPVERIFY_URL", "WX_MP_MPVERIFY_CONTENT",
		"WX_PAY_MCHID", "WX_PAY_APIKEY",
		"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_DBNAME",
	} {
		if os.Getenv(e) == "" {
			log.Fatalf("请正确设置环境变量\n")
		}
	}

	// host = "localhost"
	// user = "postgres"
	// password = "postgres"
	// dbname := "mj"
	user = os.Getenv("POSTGRES_USER")
	password = os.Getenv("POSTGRES_PASSWORD")
	host = os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DBNAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// why parseTime=true
	// error: "sql: Scan error on column index 7: null: cannot scan type []uint8 into null.Time: [50 48 49 56 45 48 52 45 49 52 32 49 51 58 52 56 58 48 52]"
	// https://github.com/xo/xo/issues/19
	// connectionString := fmt.Sprintf("%s:%s@%s/%s?parseTime=true", user, password, host, dbName)
	log.Printf("connectionString is %s\n", psqlInfo)

	var err error
	a.DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	// defer a.DB.Close()

	err = a.DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully connected!")

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) initializeRoutes() {
	r := a.Router

	// 设置网页授权域名 微信需要验证 这个
	// 公众号设置 -> 功能设置 -> 网页授权域名
	// r.HandleFunc("/MP_verify_w7yalxZBScxCceA2.txt", mpVerifyHandler)
	r.HandleFunc(wxMpVerifyURL, mpVerifyHandler)

	// 开始请求网页授权
	r.HandleFunc("/page1", page1Handler)
	r.HandleFunc("/auth", page1Handler)

	// 由page1 跳转至此
	// /page2?code=081cbFCN1twrta131vCN1dFuCN1cbFCh&state=0d4910ba5704d6c37a911fd56af82abb
	// 获取到用户信息然后跳到回调URL
	r.HandleFunc("/page2", page2Handler())

	// 返回 中控服务器缓存的 accessToken
	r.HandleFunc("/access_token", accessTokenHandler)

	r.HandleFunc("/auth_callback", loginHandler(a.DB))
	r.HandleFunc("/bindPhone", bindPhoneHandler(a.DB))

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

	// r.HandleFunc("/v1/login", loginHandler(a.DB))
	// r.HandleFunc("/v1/signup", signUpHandler(a.DB))
}

func main() {
	a := App{}
	a.Initialize(os.Getenv("APP_DB_USER"), os.Getenv("APP_DB_PASSWORD"), os.Getenv("APP_DB_HOST"), os.Getenv("APP_DB_NAME"))

	log.Printf("listening on port %s", *flagPort)

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.
	// handler := cors.Default().Handler(mux)
	// log.Fatal(http.ListenAndServe(":"+*flagPort, a.Router))

	handler := cors.Default().Handler(a.Router)
	// https://github.com/rs/cors
	log.Fatal(http.ListenAndServe(":"+*flagPort, handler))
}

type AuthInfo struct {
	WechatMPAuth

	MobilePhone string `json:"mobile_phone"`
	UserID      string `json:"user_id"`
}
