package main

import (
	"echo/router"
)

func main() {
	r := router.SetupRouter()
	r.Run(":8080")

}
