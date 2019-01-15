package main

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func TestTimerFunc(t *testing.T) {
	db, err := gorm.Open("mysql", "web:web123!@#@tcp(10.3.247.59:3306)/test?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO id_configs SET hashid = ?,svrid= ?,start_time=now(),cursvr=-1 ", i, i+1)
	}
}
