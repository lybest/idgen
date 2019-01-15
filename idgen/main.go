package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {

	id := flag.Int64("svrid", 1, "svrid")
	step := flag.Int64("step", 1000, "step")
	flag.Parse()

	SvrID = *id
	Step = *step

	r := gin.New()

	r.Use(gin.Logger())

	r.Use(gin.Recovery())

	r.GET("/id", func(c *gin.Context) {
		uuid := c.Query("uuid")
		u, _ := strconv.ParseInt(uuid, 10, 0)
		fmt.Println(u)
		c.String(200, "%v", GetId(u))
	})
	r.Run()
}
