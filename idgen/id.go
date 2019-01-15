package main

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

var (
	Step       int64
	SvrID      int64
	ListenIP   string
	ListenPort int
	db         *gorm.DB
	IDItems    map[int64]*IDItem = map[int64]*IDItem{}
	IDConfigs  []IDConfig        = []IDConfig{}
	IDStatusA                    = &IDStatus{}
	ErrorTime  time.Time
	ErrorCount int
)

type IDItem struct {
	Hashid int64 `gorm:"PRIMARY_KEY"`
	CurID  int64
	Maxid  int64
}

type IDConfig struct {
	Hashid    int64 `gorm:"PRIMARY_KEY"`
	Svrid     int64
	StartTime time.Time
	Cursvr    int64
}

type IDStatus struct {
	Svrid      int64 `gorm:"PRIMARY_KEY"`
	Status     int   `gorm:"-"`
	UpdateTime time.Time
	Ip         string
	Port       int
}

func GetId(uuid int64) int64 {
	hashid := uuid / 100000
	fmt.Println(hashid)
	item, ok := IDItems[hashid]
	if ok {
		if atomic.LoadInt64(&item.CurID) < atomic.LoadInt64(&item.Maxid) {
			cur := atomic.AddInt64(&item.CurID, 1)
			if cur <= atomic.LoadInt64(&item.Maxid) {
				return cur
			}
			atomic.AddInt64(&item.CurID, -1)
		}
		fmt.Println(*item)
		return -1
	}
	return -1
}

func TimerFunc() {
	IDStatusA.Ip = ListenIP
	IDStatusA.Port = ListenPort
	IDStatusA.Svrid = SvrID
	IDStatusA.UpdateTime = time.Now()
	var err error
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.New("core")
			fmt.Println("recover ")
		}

		if err != nil {
			if ErrorCount == 0 {
				ErrorTime = time.Now()
				ErrorCount++
				fmt.Println("get new error")
			} else if time.Now().Sub(ErrorTime) > time.Second*30 {
				fmt.Println("remove service")
				IDItems = make(map[int64]*IDItem)
			}
		} else {
			ErrorCount = 0
			fmt.Println("Timer success")
		}
	}()

	if err = tx.Error; err != nil {
		fmt.Println("Tx error ", err)
		return
	}

	if err = tx.Save(IDStatusA).Error; err != nil {
		fmt.Println("Save Status error ", err)
		tx.Rollback()
		return
	}

	if err = tx.Where("start_time < ?", time.Now()).Find(&IDConfigs).Error; err != nil {
		fmt.Println("Find IDConfigs error ", err)
		tx.Rollback()
		return
	}
	fmt.Println(IDConfigs)
	fmt.Println(SvrID)
	for _, cfg := range IDConfigs {
		item, ok := IDItems[cfg.Hashid]

		if cfg.Cursvr != SvrID && ok {
			//* 当前提供服务者不是自己 但是自己正在提供服务 停止服务
			delete(IDItems, cfg.Hashid)

		} else if cfg.Cursvr != SvrID && !ok {
			//* 当前提供服务者不是自己 自己也没提供服务 忽略
			continue
		} else if ok && item.CurID < item.Maxid-Step/10 {
			//* 当前服务提供者是自己 且正在提供服务 且curid还没达到maxid 忽略
			continue
		} else {
			//* 当前服务提供者是自己 没提供服务或者curid快耗尽了 重新申请id段
			if err = tx.Exec("insert into id_items set  hashid = ?,maxid =? ON DUPLICATE KEY UPDATE maxid = maxid + ?", cfg.Hashid, Step, Step).Error; err != nil {
				fmt.Println("INSERT IDItems error ", err)
				tx.Rollback()
				return
			}
			if item == nil {
				item = &IDItem{}
			}
			if err = tx.First(&item, cfg.Hashid).Error; err != nil {
				fmt.Println("tx First error: ", err)
				tx.Rollback()
				return
			}
			fmt.Println(*item)
			if ok {
				IDItems[cfg.Hashid].Maxid = item.Maxid
			} else {
				item.CurID = item.Maxid - Step
				IDItems[cfg.Hashid] = item
			}
		}

	}
	fmt.Println(IDItems)

	if err = tx.Commit().Error; err != nil {
		fmt.Println("Commit error ", err)
	}

}

func init() {

	var err error
	db, err = gorm.Open("mysql", "root:123456@tcp(10.3.247.59:3306)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			TimerFunc()
			time.Sleep(time.Second * 3)
		}
	}()

}
