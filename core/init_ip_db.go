package core

import (
	"fmt"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
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

func GetIpAddr(ip string) (addr string, err error) {
	region, err := searcher.Search(ip)
	if err != nil {
		return
	}
	return region, nil
}
