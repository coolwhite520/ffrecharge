package proxy

import (
"encoding/json"
"fmt"
"io/ioutil"
"net/http"
"net/url"
"time"
)
type NodeIP struct {
	Ip string `json:"ip"`
	Port int `json:"port"`
	Outip string `json:"outip"`
}

type ProxyList struct {
	Code int `json:"code"`
	Data []NodeIP `json:"data"`
	Msg string `json:"msg"`
	Success bool `json:"success"`
}

type XunNode struct {
	Ip string `json:"ip"`
	Port string `json:"port"`
}

type XunList struct {
	ErrorCode string `json:"ERRORCODE"`
	Result []XunNode `json:"RESULT"`
}


const zhimadailiApi = "http://http.tiqu.alicdns.com/getip3?num=50&type=2&pro=&city=0&yys=0&port=1&time=1&ts=0&ys=0&cs=0&lb=1&sb=0&pb=4&mr=1&regions=&gm=4"
const xdailiApi = "http://api.xdaili.cn/xdaili-api//greatRecharge/getGreatIp?spiderId=88641bc468414190be69177fbca4091f&orderno=YZ2020451506JQFRoT&returnType=2&count=10"
//const remoteUrl =  "http://upay.10010.com/npfwap/npfMobWap/bankcharge/index.html"
const remoteUrl =  "https://www.baidu.com"
/*
验证代理ip是否可用
通过传入一个代理ip，然后使用它去访问一个url看看是否访问成功，以此为依据进行判断当前代理ip是否有效。
参数：proxy_addr 要验证的ip
返回：ip 验证通过的ip、status 状态（200表示成功）
*/
func proxyThorn(proxy_addr string) (ip string,status int) {
	//访问查看ip的一个网址

	proxy, err := url.Parse(proxy_addr)
	netTransport := &http.Transport{
		Proxy:http.ProxyURL(proxy),
		MaxIdleConnsPerHost: 10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: netTransport,
	}
	res, err := httpClient.Get(remoteUrl)
	if err != nil {
		//fmt.Println("错误信息：",err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		//log.Println(err)
		return
	}
	//c, _ := ioutil.ReadAll(res.Body)
	//fmt.Println(string(c))
	return proxy_addr, res.StatusCode
}
/*
批量从redis中读取代理ip
*/
//func get_to_redis_ips(redis_host string,set_name string) (list [] map[string]string)  {
//	//连接redis
//	conn,err := redis.Dial("tcp",redis_host)
//	if err != nil {
//		fmt.Println("connect redis error :",err)
//		return
//	}
//	defer conn.Close()
//
//	//获取test_set集合中的所有数据，并循环遍历打印
//	list_set, err := redis.Values(conn.Do("SMEMBERS", set_name))
//	if err != nil {
//		fmt.Println("获取test_set集合中的值失败:", err)
//	}
//	//定义返回的列表数据类型
//	var ly_list [] map[string]string
//	for _, v := range list_set {
//		if str, ok := v.([]uint8); ok {
//			//json str 转map
//			var ip_data=make(map[string]string)
//			if err := json.Unmarshal([]byte(string(str)), &ip_data); err == nil {
//				//依次将转换好的数据最加到列表中
//				ly_list=append(ly_list,ip_data)
//			}
//		}
//	}
//	return ly_list
//}

func verificationIP(ip_info string, ch chan string) {
	var ip,status = proxyThorn(ip_info)
	//判断是否有返回ip，并且请求状态为200
	if status==200 && ip!=""{
		ch <- ip
		//fmt.Println(ip_info+" 请求 "+remoteUrl+" 返回ip:【"+ip+"】-【检测结果：可用】")
	}else {
		//fmt.Println(ip_info+" 请求 " +remoteUrl+ "返回ip:【"+ip+"】-【检测结果：不可用】")
	}
}

/**
批量获取proxyip
*/
func getXunDailiProxylist() (list []string) {
	httpUrl := xdailiApi
	netTransport := &http.Transport{
		MaxIdleConnsPerHost: 10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: netTransport,
	}
	res, err := httpClient.Get(httpUrl)
	if err != nil {
		//fmt.Println("错误信息：",err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return
	}
	c, _ := ioutil.ReadAll(res.Body)
	var a XunList
	if err = json.Unmarshal(c, &a); err != nil {
		fmt.Printf("Unmarshal err, %v\n", err)
		return
	}
	for _, item := range a.Result {
		str := fmt.Sprintf("%s:%s", item.Ip, item.Port)
		list = append(list, str)
	}
	return list
}
func getZhiMaproxylist() (list []string) {
	httpUrl := zhimadailiApi
	netTransport := &http.Transport{
		MaxIdleConnsPerHost: 10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: netTransport,
	}
	res, err := httpClient.Get(httpUrl)
	if err != nil {
		//fmt.Println("错误信息：",err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return
	}
	c, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(c))
	var a ProxyList
	if err = json.Unmarshal(c, &a); err != nil {
		fmt.Printf("Unmarshal err, %v\n", err)
		return
	}
	for _, item := range a.Data {
		str := fmt.Sprintf("%s:%d", item.Ip, item.Port)
		list = append(list, str)
	}
	return list
}

func GetProxyAddr() string {
	ch := make(chan string)
	go func() {
		time.Sleep(time.Second * time.Duration(10))
		ch <- ""
	}()
	ipList := getZhiMaproxylist()
	//ipList := getXunDailiProxylist()
	if len(ipList) > 0 {
		for _, ip := range ipList {
			go verificationIP(ip, ch)
		}
		ip := <-ch
		return ip
	}
	return ""
}

