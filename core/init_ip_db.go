package core

import (
	"fmt"
	"strings"

	ipUtils "go-star/utils/ip"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/sirupsen/logrus"
)

var searcher *xdb.Searcher

func InitIPDB() {
	var dbPath = "init/ip2region.xdb"
	version := xdb.IPv4
	_searcher, err := xdb.NewWithFileOnly(version, dbPath)
	if err != nil {
		fmt.Printf("ip地址数据库加载失败: %s", err)
		return
	}

	searcher = _searcher
	fmt.Println("✅ IP地址库初始化成功")
}

func GetIpAddr(ip string) (addr string) {
	if ipUtils.HasLocalIPAddr(ip) {
		return "内网"
	}
	region, err := searcher.Search(ip)
	if err != nil {
		logrus.Warnf("错误的IP地址 %s", err)
		return "异常地址"
	}
	_addrList := strings.Split(region, "|")
	if len(_addrList) != 5 {
		logrus.Warnf("异常的ip地址 %s", ip)
		return "未知地址"
	}

	// 五个部分
	//中国|广东省|东莞市|移动|CN
	country := _addrList[0]
	province := _addrList[1]
	city := _addrList[2]

	if province != "0" && city != "0" {
		return fmt.Sprintf("%s·%s", province, city)
	}
	if country != "0" && province != "0" {
		return fmt.Sprintf("%s·%s", country, province)
	}
	if country != "0" {
		return country
	}
	return region
}
