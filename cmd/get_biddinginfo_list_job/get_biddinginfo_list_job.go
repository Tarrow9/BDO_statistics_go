package main

import (
	"bdo_calc_go/pkg/bdoapi"
	"fmt"
)

func main() {
	minSale, maxBuy, err := bdoapi.GetBiddingInfoList(15720, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("minSale: %d, maxBuy: %d\n", minSale, maxBuy)
}
