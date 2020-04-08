package autorecharge

import (
	"fmt"
	"github.com/Unknwon/goconfig"
	. "github.com/coolwhite520/ffrecharge/protocol"
	"github.com/coolwhite520/ffrecharge/proxy"
	gogetter "github.com/hashicorp/go-getter"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"os"
	"strconv"
	"sync"
	"time"
)

/**
	自动模拟
 */
const (
	port = 9515
)

var mu sync.Mutex
var service *selenium.Service
var seleniumPath, homePage string
var cfg *goconfig.ConfigFile

func init() {
	var err error
	cfg, err = goconfig.LoadConfigFile("./config.ini")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "LoadConfigFile"}).Fatal(err.Error())
		return
	}
}

func StartServer() {
	var err error
	seleniumPath, err = cfg.GetValue("Chrome", "DriverPath")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "DriverPath"}).Fatal(err.Error())
		return
	}
	homePage, err = cfg.GetValue("Chrome", "HomePage")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "HomePage"}).Fatal(err.Error())
		return
	}
	opts := []selenium.ServiceOption{}
	//selenium.SetDebug(true)
	service, err = selenium.NewChromeDriverService(seleniumPath, port, opts...)
	if nil != err {
		log.WithFields(log.Fields{"funcName": "NewChromeDriverService", "port": port}).Fatal(err.Error())
		return
	}

	log.WithFields(log.Fields{}).Info("@@@@ChromeDriverService start success.")

}

func StopServer() {
	err := service.Stop()
	if err != nil {
		log.WithFields(log.Fields{"funcName": "StopServer"}).Fatal(err.Error())
	} else {
		log.WithFields(log.Fields{}).Info("@@@@ChromeDriverService stop success.")
	}
}

func Run( task *PushReqTag, chObj *protocol.FFChAppTag) {
	defer func() {
		chObj.ChSelfExit <- task.OrderNo
	}()
	OpenNewPage(task, chObj);
}
/**
	创建一个新的页面
    paymentid: id
	ch ： 可写channel，负责传递qrcode
	chquit： 可读channel，负责读取外部通知退出消息
	chselftQuit : 可写channel，通知外部，自己退出
*/
func OpenNewPage( task *PushReqTag, chObj *FFChAppTag) {
	//链接本地的浏览器 chrome
	OrderNo := task.OrderNo
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	//禁止图片加载，加快渲染速度
	imagCaps := map[string]interface{}{
		// "profile.managed_default_content_settings.images": 2,
	}
	imagCaps = nil

	headless, err := cfg.GetValue("WebPage", "HeadLess")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "HeadLess"}).Fatal(err.Error())
		return
	}

	var args []string
	args = append(args, "--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36")
	args = append(args, fmt.Sprintf("--OrderNo=%v", OrderNo))
	if headless == "true" {
		args = append(args, "--headless")
	}

	useProxy, err := cfg.GetValue("WebPage", "UseProxy")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "UseProxy"}).Fatal(err.Error())
		return
	}

	if useProxy == "true" {
		proxyIp := proxy.GetProxyAddr()
		log.WithFields(log.Fields{"GetValue": "UseProxy", "proxy": proxyIp , "orderno": OrderNo}).Info()
		args = append(args, fmt.Sprintf("----proxy-server=%v", proxyIp))
	}

	iphone7Plus := &chrome.MobileEmulation{
		DeviceName:    "iPhone 7 Plus",
		DeviceMetrics: nil,
		UserAgent:     "",
	}

	chromeCaps := chrome.Capabilities{
		Prefs:           imagCaps,
		Path:            "",
		Args:            args,
		ExcludeSwitches: []string{"enable-automation", "enable-logging"},
		MobileEmulation: iphone7Plus,
	}
	fmt.Println(chromeCaps)
	//以上是设置浏览器参数
	caps.AddChrome(chromeCaps)
	// 调起chrome浏览器

	webPage, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		log.WithFields(log.Fields{"funcName": "NewRemote", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	//关闭一个webDriver会对应关闭一个chrome窗口
	//但是不会导致seleniumServer关闭
	defer func() {
		_ = webPage.Quit()
	}()

	_ = webPage.MaximizeWindow("")
	_ = webPage.ResizeWindow("", 800, 1200);

	err = webPage.Get(homePage)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "Get", "OrderNo": OrderNo}).Error(err.Error())
		go OpenNewPage(task, chObj)
		return
	}
	err = webPage.SetPageLoadTimeout(30 * time.Second)
	if err != nil {
		go OpenNewPage(task, chObj)
		return
	}
	inputPhoneNumber, err := webPage.FindElement(selenium.ByID, "number")
	if err!=nil {
		log.WithFields(log.Fields{"func": "FindElement", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	err = inputPhoneNumber.SendKeys(task.Account)
	if err != nil {
		log.WithFields(log.Fields{"func": "SendKeys", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	// 等待号码检测结果
	phoneType := ""
	err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error){
		phoneTypeEl, err := wd.FindElement(selenium.ByCSSSelector, "label.custominfos")
		if err != nil {
			fmt.Println(err.Error())
			return false, nil
		} else {
			phoneType, _ = phoneTypeEl.Text()
			if len(phoneType) > 0 {
				return true, nil
			}
			return false, nil
		}
	}, 1 * time.Minute, 1 * time.Second)
	fmt.Println(phoneType)
	if err != nil {
		log.WithFields(log.Fields{"func": "WaitWithTimeoutAndInterval", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	// 选择并点击充值金额
	iValue, _:= strconv.ParseInt(task.Amount, 10, 64)
	var amountSelector string
	var continueSelector string
	if task.ChanelCode == "LTCZ" {
		continueSelector = "body > div.wapToAppTipDiv > div > div > a.toPay"
		amountSelector = fmt.Sprintf("#cardlist > section > div > a[cardvalue=\"%d\"]",  iValue * 100)
	} else {
		continueSelector = "#body-class-random > div.otherChangeMask > div > div.tipBtn > span"
		amountSelector = fmt.Sprintf("section.amount-box.fixed.mobileCardListOther > div > a[cardvalue=\"%d\"]",  iValue * 100)
	}
	err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error){
		amountSelectEl, err := wd.FindElement(selenium.ByCSSSelector, amountSelector)
		if err != nil {
			fmt.Println(err.Error())
			return false, nil
		} else {
			content, _ := amountSelectEl.Text()
			if len(content) > 0 {
				_ = amountSelectEl.Click()
				return true, nil
			}
			return false, nil
		}
	}, 1 * time.Minute, 1 * time.Second)
	if err != nil {
		log.WithFields(log.Fields{"func": "WaitWithTimeoutAndInterval", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	// 点击弹出窗口的继续按钮
	continueBtn, err := webPage.FindElement(selenium.ByCSSSelector, continueSelector)
	if err != nil {
		log.WithFields(log.Fields{"func": "FindElement", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	_ = continueBtn.Click()
	// 等待支付方式页面弹出点击支付方式
	err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error){
		el, err := wd.FindElement(selenium.ByCSSSelector, "body > div.mask.confirmPay > section.btmPart")
		if err != nil {
			return false, nil
		} else {
			if bottom, err := el.CSSProperty("bottom"); err == nil && bottom == "0px" {
				return true, nil
			}
			return false, nil
		}
	}, 1 * time.Minute, 1 * time.Second)
	if err != nil {
		log.WithFields(log.Fields{"func": "WaitWithTimeoutAndInterval", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	payTypeSelector := fmt.Sprintf("body > div.mask.confirmPay > section > div.ulMinH > div.payTypeLists > div[channelcode=\"%v\"]", task.PayType)
	payTypeBtn, err := webPage.FindElement(selenium.ByCSSSelector, payTypeSelector)
	if err != nil {
		log.WithFields(log.Fields{"func": "FindElement", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	_ = payTypeBtn.Click()

	// 点击submit
	submitBtn, err := webPage.FindElement(selenium.ByCSSSelector, "body > div.mask.confirmPay > section > div.btnPd > button")
	if err != nil {
		log.WithFields(log.Fields{"func": "FindElement", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	_ = submitBtn.Click()
	// 切换frame并获取内部滑块的图片
	var iframe selenium.WebElement
	webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (b bool, err error) {
		iframe, err = wd.FindElement(selenium.ByCSSSelector, "iframe")
		if iframe != nil {
			err = wd.SwitchFrame(iframe)
			if err != nil {
				log.WithFields(log.Fields{"func": "SwitchFrame", "OrderNo": OrderNo}).Error(err.Error())
				return
			}
			return true, nil
		}
		return false, nil
	},  1 * time.Minute, 1 * time.Second)

	var bgImgSrc, sliderImgSrc string
	err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error){
		bgImgEl, _ := wd.FindElement(selenium.ByCSSSelector, "#bkBlock")
		sliderImgEl, _ := wd.FindElement(selenium.ByCSSSelector, "#slideBlock")
		if bgImgEl != nil && sliderImgEl != nil {
			bgImgSrc, _ = bgImgEl.GetAttribute("src")
			sliderImgSrc, _= sliderImgEl.GetAttribute("src")
			if bgImgSrc != "" && sliderImgSrc != "" {
				return true, nil
			}
		}
		return false, nil
	}, 1 * time.Minute, 1 * time.Second)
	if err != nil {
		log.WithFields(log.Fields{"func": "WaitWithTimeoutAndInterval", "OrderNo": OrderNo}).Error(err.Error())
		return
	}
	// 下载两张图片，并进行裁剪
	//gogetter.GetFile()
	gogetter.GetAny("a.png", bgImgSrc)
	gogetter.GetAny("b.png", sliderImgSrc)

	time.Sleep(5* time.Minute)
}


func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

