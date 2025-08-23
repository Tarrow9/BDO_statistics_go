package bdoapi

import (
	hfm "bdo_calc_go/pkg/huffmanunpack"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 요청 시 Payload 구조체
type ReqPayload interface {
	CategoryPayload | MainKeyPayload | MainSubKeyPayload
}
type CategoryPayload struct {
	KeyType      int `json:"keyType"`
	MainCategory int `json:"mainCategory"`
	SubCategory  int `json:"subCategory"`
}
type MainKeyPayload struct {
	KeyType int `json:"keyType"`
	MainKey int `json:"mainKey"`
}
type MainSubKeyPayload struct {
	KeyType int `json:"keyType"`
	MainKey int `json:"mainKey"`
	SubKey  int `json:"subKey"`
}

// 응답 시 받는 데이터 구조체
type RespObject interface {
	MarketListObject | MarketSubListObject
}
type MarketListObject struct {
	ItemID       int64
	CurrentStock int64
	TotalTrades  int64
	BasePrice    int64
}
type MarketSubListObject struct {
	ItemID int64
	// MinEnhance      int8
	// MaxEnhance      int8
	// BasePrice       int64
	CurrentStock int64
	TotalTrades  int64
	// MinPriceHardCap int64
	// MaxPriceHardCap int64
	LastTradePrice int64
	// LastTradeTime   time.Time
}

// 내부 연산 시 사용되는 구조체
type ItemGroup struct {
	ItemID   int
	ItemName string
}
type BiddingOrder struct {
	Price int64
	Sale  int64
	Buy   int64
}

// 지정값 하드코딩
var baseUrl = "https://trade.kr.playblackdesert.com/Trademarket/"
var PayloadMap = map[string]CategoryPayload{
	"ore":     {KeyType: 0, MainCategory: 25, SubCategory: 1},
	"plants":  {KeyType: 0, MainCategory: 25, SubCategory: 2},
	"seed":    {KeyType: 0, MainCategory: 25, SubCategory: 3},
	"leather": {KeyType: 0, MainCategory: 25, SubCategory: 4},
	"blood":   {KeyType: 0, MainCategory: 25, SubCategory: 5},
	"meat":    {KeyType: 0, MainCategory: 25, SubCategory: 6},
	"seafood": {KeyType: 0, MainCategory: 25, SubCategory: 7},
	"misc":    {KeyType: 0, MainCategory: 25, SubCategory: 8},

	"offensive_elixir":  {KeyType: 0, MainCategory: 35, SubCategory: 1},
	"defensive_elixir":  {KeyType: 0, MainCategory: 35, SubCategory: 2},
	"functional_elixir": {KeyType: 0, MainCategory: 35, SubCategory: 3},
	"food":              {KeyType: 0, MainCategory: 35, SubCategory: 4},
	"portion_elixir":    {KeyType: 0, MainCategory: 35, SubCategory: 5},
}

// 아이템 그룹 별 가장 싼 아이템 고르는 용도
var itemGroupMap = map[string][]ItemGroup{
	"deer": {
		{ItemID: 6201, ItemName: "사슴 피"},
		{ItemID: 6202, ItemName: "양 피"},
		{ItemID: 6206, ItemName: "소 피"},
		{ItemID: 6215, ItemName: "와라곤 피"},
		{ItemID: 6205, ItemName: "돼지 피"},
		{ItemID: 6227, ItemName: "라마 피"},
		{ItemID: 6228, ItemName: "염소 피"},
	},
	"wolf": {
		{ItemID: 6214, ItemName: "늑대 피"},
		{ItemID: 6204, ItemName: "코뿔소 피"},
		{ItemID: 6216, ItemName: "치타룡 피"},
		{ItemID: 6218, ItemName: "홍학 피"},
	},
	"fox": {
		{ItemID: 6203, ItemName: "여우 피"},
		{ItemID: 6210, ItemName: "너구리 피"},
		{ItemID: 6211, ItemName: "원숭이 피"},
		{ItemID: 6212, ItemName: "족제비 피"},
		{ItemID: 6224, ItemName: "전갈 피"},
		{ItemID: 6226, ItemName: "마못 피"},
	},
	"bear": {
		{ItemID: 6213, ItemName: "곰 피"},
		{ItemID: 6223, ItemName: "사자 피"},
		{ItemID: 6220, ItemName: "트롤 피"},
		{ItemID: 6221, ItemName: "오우거 피"},
		{ItemID: 6207, ItemName: "공룡 피"},
		{ItemID: 6225, ItemName: "야크 피"},
	},
	"lizard": {
		{ItemID: 6208, ItemName: "도마뱀 피"},
		{ItemID: 6209, ItemName: "웜 피"},
		{ItemID: 6219, ItemName: "박쥐 피"},
		{ItemID: 6217, ItemName: "쿠쿠새 피"},
		{ItemID: 6222, ItemName: "코브라 피"},
	},

	"meat": {
		{ItemID: 7913, ItemName: "늑대 고기"},
		{ItemID: 7961, ItemName: "토끼 고기"},
		{ItemID: 7925, ItemName: "가젤 고기"},
		{ItemID: 7901, ItemName: "사슴 고기"},
		{ItemID: 7960, ItemName: "강치 고기"},
		{ItemID: 7904, ItemName: "코뿔소 고기"},
		{ItemID: 7911, ItemName: "족제비 고기"},
		{ItemID: 7910, ItemName: "너구리 고기"},
		{ItemID: 7912, ItemName: "곰고기"},
		{ItemID: 7905, ItemName: "돼지고기"},
		{ItemID: 7957, ItemName: "염소 고기"},
		{ItemID: 7903, ItemName: "여우 고기"},
		{ItemID: 7906, ItemName: "소고기"},
		{ItemID: 7902, ItemName: "양고기"},
	},
	"grain":  {{ItemID: 7003, ItemName: "감자"}},
	"powder": {{ItemID: 7103, ItemName: "감자 가루"}},
	"dough":  {{ItemID: 7203, ItemName: "감자 반죽"}},
}

// /* 외부로 공개되는 인터페이스 */
// type BDOAPIController interface {
// 	SetReq(category string)
// }

// // 생성 함수
// func NewBDOAPIController() BDOAPIController {
// 	return &innerBDOAPIObject{
// 		BaseUrl:   "https://trade.kr.playblackdesert.com",
// 		Headers:   "",
// 		Payloads:  Payload{},
// 		ItemGroup: "",
// 	}
// }

/* object 컨트롤 함수(인터페이스에 적용) */
// type innerBDOAPIObject struct {
// 	BaseUrl   string
// 	Headers   string
// 	Payloads  Payload
// 	ItemGroup []int
// }

func doRequest[T ReqPayload](targetAPI string, payload T) (string, error) {
	targetUrl := baseUrl + targetAPI
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", targetUrl, bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "BlackDesert")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func doRequestUnpack[T ReqPayload](targetAPI string, payload T) (string, error) {
	targetUrl := baseUrl + targetAPI
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", targetUrl, bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "BlackDesert")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	unpackedData, err := hfm.UnpackBytes(data)
	if err != nil {
		fmt.Println("failed to unpack data:", err)
		return "", err
	}

	return unpackedData, nil
}

// func doParsing[T RespObject](record string, obj T) (T, error) {
// 	fs := strings.Split(record, "-")
// }

// func (c *innerBDOAPIObject) SetReq(category string) *http.Client {
func GetMarketList(category string) ([]MarketListObject, error) {
	marketListRawStr, err := doRequestUnpack("GetWorldMarketList", PayloadMap[category])
	if err != nil {
		return nil, fmt.Errorf("wrong request: [GetWorldMarketList] %s", category)
	}

	parts := strings.Split(marketListRawStr, "|")
	out := make([]MarketListObject, 0, len(parts))

	for idx, rec := range parts {
		if rec == "" {
			continue
		}
		fs := strings.SplitN(rec, "-", 4)
		if len(fs) != 4 {
			return nil, fmt.Errorf("[%s] id(%d): wrong format... [%s]", category, idx, rec)
		}
		itemID, err := strconv.ParseInt(fs[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: ItemID: %w", idx, err)
		}
		curr, err := strconv.ParseInt(fs[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: CurrentStock: %w", idx, err)
		}
		total, err := strconv.ParseInt(fs[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: TotalTrades: %w", idx, err)
		}
		price, err := strconv.ParseInt(fs[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: BasePrice: %w", idx, err)
		}
		out = append(out, MarketListObject{
			ItemID:       itemID,
			CurrentStock: curr,
			TotalTrades:  total,
			BasePrice:    price,
		})
	}
	return out, nil
}

// list는 강화단계별로 나뉘어져 있음
func GetMarketSubList(mainkey int) ([]MarketSubListObject, error) {
	marketSubListRawStr, err := doRequest("GetWorldMarketSubList", MainKeyPayload{KeyType: 0, MainKey: mainkey})
	if err != nil {
		return nil, fmt.Errorf("wrong request: [GetWorldMarketSubList] %d", mainkey)
	}

	var respMap map[string]interface{}
	if err := json.Unmarshal([]byte(marketSubListRawStr), &respMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: [GetWorldMarketSubList] %d", mainkey)
	}

	resultMsg := respMap["resultMsg"].(string)
	parts := strings.Split(resultMsg, "|")
	out := make([]MarketSubListObject, 0, len(parts))

	for idx, rec := range parts {
		if rec == "" {
			continue
		}
		fs := strings.SplitN(rec, "-", 10)
		if len(fs) != 10 {
			return nil, fmt.Errorf("[%d] id(%d): wrong format... [%s]", mainkey, idx, rec)
		}
		itemID, err := strconv.ParseInt(fs[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: ItemID: %w", idx, err)
		}
		curr, err := strconv.ParseInt(fs[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: CurrentStock: %w", idx, err)
		}
		total, err := strconv.ParseInt(fs[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: TotalTrades: %w", idx, err)
		}
		price, err := strconv.ParseInt(fs[8], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("record %d: LastTradePrice: %w", idx, err)
		}

		out = append(out, MarketSubListObject{
			ItemID:         itemID,
			CurrentStock:   curr,
			TotalTrades:    total,
			LastTradePrice: price,
		})
	}
	return out, nil
}

func GetBiddingInfoList(mainkey int, grade int) (int64, int64, error) {
	/* 계산기 내부에서 사용 */
	biddingInfoRawStr, err := doRequestUnpack("GetBiddingInfoList", MainSubKeyPayload{KeyType: 0, MainKey: mainkey, SubKey: grade})
	if err != nil {
		return -1, -1, fmt.Errorf("wrong request: [GetBiddingInfoList] %d, %d", mainkey, grade)
	}
	var orders []BiddingOrder

	// 문자열 파싱
	parts := strings.Split(biddingInfoRawStr, "|")
	for _, bid := range parts {
		if bid == "" {
			continue
		}
		parts := strings.Split(bid, "-")
		if len(parts) != 3 {
			continue
		}

		price, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return -1, -1, fmt.Errorf("record %d: Price: %w", mainkey, err)
		}
		sale, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return -1, -1, fmt.Errorf("record %d: Sale: %w", mainkey, err)
		}
		buy, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return -1, -1, fmt.Errorf("record %d: Buy: %w", mainkey, err)
		}
		orders = append(orders, BiddingOrder{Price: price, Sale: sale, Buy: buy})
	}

	// 최저 판매가 & 최고 매수가 찾기
	minSale := int64(math.MaxInt64)
	maxBuy := int64(0)

	for _, o := range orders {
		if o.Sale > 0 && o.Price < minSale {
			minSale = o.Price
		}
		if o.Buy > 0 && o.Price > maxBuy {
			maxBuy = o.Price
		}
	}

	if minSale == int64(math.MaxInt64) {
		minSale = 0 // 판매 대기 없음
	}
	return minSale, maxBuy, nil
}
