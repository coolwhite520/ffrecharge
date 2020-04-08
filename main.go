package main

import (
	"github.com/coolwhite520/alipayserver/logcuthook"
	"github.com/coolwhite520/alipayserver/loghook"
	"github.com/coolwhite520/ffrecharge/autorecharge"
	"github.com/coolwhite520/ffrecharge/tools"
	gogetter "github.com/hashicorp/go-getter"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	app    *iris.Application
	webPageMap = make(map[string]*FFChAppTag)
)

func init() {
	logcuthook.ConfigLocalFilesystemLogger("./logs", "mylog", time.Hour*24*60, time.Hour*24)
	log.AddHook(loghook.NewContextHook())
	log.SetFormatter(&log.TextFormatter{TimestampFormat: "2006-01-02 15:04:05"})
	go listenSignal()
}

func main() {
	bgImgSrc:="https://captcha.guard.qcloud.com/cap_union_new_getcapbysig?asig=-Wn0Vv9rbWsUwHlBZa6G5rmxVu0EKWqefuIWuVnw6RlGI8JiHrC2AdFMl6I_SuLAwioXwCU02eZIEUtJafx-MiNaU2muEw-u&aid=1252323825&captype=&protocol=https&clientype=1&disturblevel=&apptype=&noheader=&color=FF8C00&showtype=popup&fb=1&theme=&lang=2052&ua=TW96aWxsYS81LjAgKGlQaG9uZTsgQ1BVIGlQaG9uZSBPUyAxM18yXzMgbGlrZSBNYWMgT1MgWCkgQXBwbGVXZWJLaXQvNjA1LjEuMTUgKEtIVE1MLCBsaWtlIEdlY2tvKSBWZXJzaW9uLzEzLjAuMyBNb2JpbGUvMTVFMTQ4IFNhZmFyaS82MDQuMQ==&sess=J1D2usTEIZm28TM6mdXe3Kxw6Dmy9-Cs_D4Af_8oBQXm7cDmc5ypZ3e84q3MgYrgNcm_3irBQKvSVSRWT891SinUSxzJfn9ypl7USQmwhFDrnefBipt_eB5V_Hbf54QBi6rSERzMftLI5x7JLBKNs6Vo7NWuPd9izkjN82XB0kEf4pQHoG58Zwa0bJ6GLB_h&fwidth=0&uid=&cap_cd=&rnd=160249&rand=0.9987429565311077&vsig=b01CPkM-YI0XlDvtv6URU4Eg6h8R0lC82gPNpegPAfkohmtQYqz2ub26zOjTtfk3rhtIw7Xp5SCJao-85EJJhPG4OpPhFEQ_zr2ujKhtvz5LjIVxgL9_E35uQ**&img_index=1"
	gogetter.GetAny("a.png", bgImgSrc)

	autorecharge.StartServer()
	defer autorecharge.StopServer()
	log.WithFields(log.Fields{"funcName": "Main", "json": "Begin"}).Info()
	app = iris.Default()
	app.Use(tools.LoggerMiddleware)
	app.Post("/push", push)
	err := app.Run(iris.Addr(":7000"))
	if err != nil {
		log.WithFields(log.Fields{}).Error(err)
	}
}
func listenSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-sigs:
		c := exec.Command("taskkill.exe", "/f", "/im", "chrome.exe")
		_ = c.Start()
		os.Exit(0)
	}
}
func push(ctx context.Context) {
	var task PushReqTag
	err := ctx.ReadJSON(&task)
	if err != nil {
		str, _ := MakeJson(HttpErrno{ Status: "999", Msg: "Parameters must be passed through JSON." })
		_, _ = ctx.WriteString(str)
		log.WithFields(log.Fields{"json": str}).Error(str)
		return
	} else {
		sign := tools.GetSign(task, tools.AppSecret)
		if sign != task.Sign {
			str, _ := MakeJson(HttpErrno{ Status: "10002", Msg: "签名错误", OrderNo: task.OrderNo })
			_, _ = ctx.WriteString(str)
			log.WithFields(log.Fields{"json": str}).Error(err.Error())
			return
		}
		OrderNo := task.OrderNo
		if webPageMap[OrderNo] != nil {
			jsonStr, _ := MakeJson(HttpErrno{ Status: "10013", Msg: "相同的订单号请求了多次", OrderNo: task.OrderNo})
			_, _ = ctx.WriteString(jsonStr)
			log.WithFields(log.Fields{}).Info(jsonStr)
			return
		}

		chTag := &FFChAppTag{
			ChAppUrl:   make(chan string),
			ChQuit:     make(chan string),
			ChSelfExit: make(chan string),
		}

		webPageMap[OrderNo] = chTag
		go autorecharge.Run(&task, chTag)
		appUrl := <- chTag.ChAppUrl
		resObj := PushResSuccessTag{ OrderNo: OrderNo, Status : "20000", Msg: "成功获取app支付链接", AppPayUrl: appUrl}
		sign = tools.GetSign(resObj, tools.AppSecret)
		resObj.Sign = sign
		jsonStr, _ := MakeJson(resObj)
		_, _ = ctx.WriteString(jsonStr)
		log.WithFields(log.Fields{}).Info(jsonStr)
		return
	}
}