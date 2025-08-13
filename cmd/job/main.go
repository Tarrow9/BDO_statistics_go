package main

import (
	"bdo_calc_go/pkg/bdoapi"
	"fmt"
)

func main() {
	test, err := bdoapi.GetMarketList("ore")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(test)
}
