package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

var (
	db *gorm.DB
)

func main() {
	var err error
	db, err = gorm.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	db.DB().SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	db.DB().SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	db.DB().SetConnMaxLifetime(time.Hour)

	for {
		TimerFunc()
		time.Sleep(time.Second * 3)
	}

}

type IDStatus struct {
	Svrid      int64 `gorm:"PRIMARY_KEY"`
	Status     int
	UpdateTime time.Time
	Ip         string
	Port       int
}

type IDConfig struct {
	Hashid    int64 `gorm:"PRIMARY_KEY"`
	Svrid     int64
	StartTime time.Time
	Cursvr    int64
}

func TimerFunc() {
	var err error
	tx := db.Begin()

	if err = tx.Error; err != nil {
		fmt.Println("Tx error ", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.New("core")
			fmt.Println("recover ")
		}

		if err != nil {
			tx.Rollback()
		}
	}()

	//* 查询新上线的
	sts := make([]IDStatus, 0)
	if err = tx.Where("update_time > ? and status = 0 ", time.Now().Add(time.Second*-30)).Find(&sts).Error; err != nil {
		fmt.Println("select online error ", err)
		return
	}

	fmt.Println("online ", sts)
	for _, st := range sts {
		if err = tx.Model(&st).Update("status", 1).Error; err != nil {
			fmt.Println("update  status error ", err)
			return
		}
		if err = tx.Table("id_configs").Where("svrid = ?", st.Svrid).Updates(map[string]interface{}{"cursvr": st.Svrid, "start_time": time.Now().Add(time.Second * 30)}).Error; err != nil {
			fmt.Println("update idconfigs error ", err)
			return
		}
	}

	sts = make([]IDStatus, 0)
	//* 查询已经掉线的
	if err = tx.Where("update_time < ? and status = 1 ", time.Now().Add(time.Second*-30)).Find(&sts).Error; err != nil {
		fmt.Println("select new error ", err)
		return
	}
	fmt.Println("offline status ", sts)

	for _, st := range sts {
		err = tx.Model(&st).Update("status", 0).Error
		if err != nil {
			fmt.Println("update  status error ", err)
			return
		}
	}
	for _, st := range sts {
		//* 选择一个当前当前在线负载最小的
		fmt.Println(st)

		//* 查询下线svrid负责的hashid
		configs := make([]IDConfig, 0)
		err = tx.Where("cursvr = ?", st.Svrid).Find(&configs).Error
		if err != nil {
			fmt.Println("get config error: ", err)
			return
		}

		for _, config := range configs {
			svrid := 0
			row := tx.Raw("SELECT svrid  FROM ( SELECT svrid FROM id_statuses WHERE status = 1) b LEFT JOIN (SELECT cursvr,COUNT(*) AS count FROM id_configs GROUP BY cursvr)a ON a.cursvr = b.svrid ORDER BY count limit 1").Row()
			if err != nil {
				fmt.Println("raw error: ", err)
				return
			}
			err = row.Scan(&svrid)
			if err == sql.ErrNoRows {
				//* 没有服务可用了
				fmt.Println("all server offline ")
				return
			} else if err != nil {
				fmt.Println("scan error  ", err)
				return
			}

			fmt.Println("svrid: ", svrid)
			if err = tx.Table("id_configs").Where("hashid = ?", config.Hashid).Updates(map[string]interface{}{"cursvr": svrid, "start_time": time.Now().Add(time.Second * 30)}).Error; err != nil {
				fmt.Println("update idconfigs error ", err)
				return
			}
		}

	}
	if err = tx.Commit().Error; err != nil {
		fmt.Println("commit error")
		return
	}

}
