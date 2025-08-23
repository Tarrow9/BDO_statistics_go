package main

import (
	"bdo_calc_go/pkg/bdoapi"
	"fmt"
)

func main() {
	list, err := bdoapi.GetMarketSubList(15720)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("parsed %d items\n", len(list))
	fmt.Printf("%+v\n", list[0]) // 첫 아이템 출력 예시
}
