import requests
import json
from datetime import datetime, timedelta
from Services.unpack import unpack
from Services.RedisController import RedisController

''' 카테고리 분류
    1: 주무기
    5: 보조무기
    10: 각성무기
    15: 방어구
    20: 악세서리
    25: 재료
        1: 광석/보석
        2: 작물
        3: 씨앗/과일
        4: 가죽
        5: 피
        6: 고기
        7: 해산물
        8: 기타.
    30: 강화
    35: 소비아이템
        1. 공격비약
        2. 방어비약
        3. 기능비약
        4. 음식
        5. 물약
        6. 공성템
        7. 아이템 파츠?
        8. 기타.
    40: 생활도구
    45: 연금석
    50: 수정
    55: 펄 아이템
    60: 염색
    65: Mount?
    70: 배 관련
    75: 마차
    80: 가구
'''

''' 키 설명
    keyType: 0 고정
    mainKey: 아이템 ID
    subKey: 강화단계, 0~20, 없으면 0이 필요
    mainCategory: 메인 카테고리
    subCategory: 서브 카테고리
'''

class StatisticsCollector:
    """
    순서:
        1. 카테고리별로 base price, id 가져오기
            1-1. id의 total_trades를 이용해 거래량 산출 및 저장
        2. id의 sale_price / buy_price 가져와서 판매 / 구매 가격 산출 및 저장
    """
    def __init__(self):
        # self.redis = None
        # try:
        #     self.redis = RedisController()
        # except:
        #     print('Redis Failed.')
        #     self.redis = None
        
        self.__base_url = 'https://trade.kr.playblackdesert.com/Trademarket/'
        self.__headers = {
            "Content-Type": "application/json",
            "User-Agent": "BlackDesert"
        }
        self.__payloads = {
            "ore":      { "keyType": 0, "mainCategory": 25, "subCategory": 1 },
            "plants":   { "keyType": 0, "mainCategory": 25, "subCategory": 2 },
            "seed":     { "keyType": 0, "mainCategory": 25, "subCategory": 3 },
            "leather":  { "keyType": 0, "mainCategory": 25, "subCategory": 4 },
            "blood":    { "keyType": 0, "mainCategory": 25, "subCategory": 5 },
            "meat":     { "keyType": 0, "mainCategory": 25, "subCategory": 6 },
            "seafood":  { "keyType": 0, "mainCategory": 25, "subCategory": 7 },
            "misc":     { "keyType": 0, "mainCategory": 25, "subCategory": 8 },
            "offensive_elixir": { "keyType": 0, "mainCategory": 35, "subCategory": 1 },
            "defensive_elixir": { "keyType": 0, "mainCategory": 35, "subCategory": 2 },
            "functional_elixir":{ "keyType": 0, "mainCategory": 35, "subCategory": 3 },
            "food":             { "keyType": 0, "mainCategory": 35, "subCategory": 4 },
            "portion_elixir":   { "keyType": 0, "mainCategory": 35, "subCategory": 5 },
        }
        self.__item_group = {
            'deer': [6201, 6202, 6206, 6215, 6205, 6227, 6228], # 사슴 양 소 와라곤 돼지 라마 염소
            'wolf': [6214, 6204, 6216, 6218], # 늑대 코뿔소 치타룡 홍학
            'fox': [6203, 6210, 6211, 6212, 6224, 6226], # 여우 너구리 원숭이 족제비 전갈 마못
            'bear': [6213, 6223, 6220, 6221, 6207, 6225], # 곰 사자 트롤 오우거 공룡 야크
            'lizard': [6208, 6209, 6219, 6217, 6222], # 도마뱀 웜 박쥐 쿠쿠새 코브라

            'meat':[7913, 7961, 7925, 7901, 7960, 7904, 7911, 7910, 7912, 7905, 7957, 7903, 7906, 7902],
            'grain': [7003],
            'powder':[7103],
            'dough':[7203] 
        }
        self.__replace_dict = {
            6201:"사슴 피", 6202:"양 피", 6206:"소 피", 6215:"와라곤 피", 6205:"돼지 피", 6227:"라마 피", 6228:"염소 피",
            6214:"늑대 피", 6204:"코뿔소 피", 6216:"치타룡 피", 6218:"홍학 피",
            6203:"여우 피", 6210:"너구리 피", 6211:"원숭이 피", 6212:"족제비 피", 6224:"전갈 피", 6226:"마못 피",
            6213:"곰 피", 6223:"사자 피", 6220:"트롤 피", 6221:"오우거 피", 6207:"공룡 피", 6225:"야크 피",
            6208:"도마뱀 피", 6209:"웜 피", 6219:"박쥐 피", 6217:"쿠쿠새 피", 6222:"코브라 피",
            7913:"늑대 고기", 7961:"토끼 고기", 7925:"가젤 고기", 7901:"사슴 고기", 7960:"강치 고기", 7904:"코뿔소 고기", 7911:"족제비 고기", 7910:"너구리 고기", 7912:"곰 고기",7905:"돼지 고기",7957:"염소 고기",7903:"여우 고기",7906:"소 고기",7902:"양 고기",
            7003:"감자",
            7103:"감자 가루",
            7203:"감자 반죽",
        }


    def _status_code_not_200(self, status_code: int):
        if status_code != 200:
            print(f'API server is unavailable, status_code is: {status_code}')
            return True
        else:
            return False

    def _choose_cheapest_item(self, timestamp):
        # bid_sale 기준 가장 싼 매물

        # get redis
        # compare
        # set redis(new key)
        if not self.redis:
            return False
        ret_dict = {}
        for item_type in self.__item_group:
            ret_dict[item_type] = ('0000', 999999999) # { item_type : (price, itemid) }
            for id in self.__item_group[item_type]:
                item_info = self.redis.get_dict(str(id) + ':' + timestamp)
                cur_stock = int(item_info["current_stock"])
                bid_price = int(item_info["bid_sale_price"])
                # print(str(id), cur_stock, bid_price)

                if (cur_stock > 10000 and
                    bid_price < ret_dict[item_type][1]):

                    ret_dict[item_type] = (str(id), bid_price)

            # 조건에 맞지 않을 시 기본 피(사슴/늑대/여우/곰/도마뱀의 가격)
            if ret_dict[item_type] == ('0000', 999999999):
                common_item_id = self.__item_group[item_type][0]
                common_item_info = self.redis.get_dict(str(common_item_id) + ':' + timestamp)

                bid_price = int(common_item_info["bid_sale_price"])
                ret_dict[item_type] = (str(common_item_id), bid_price)
            # print()

        # print(ret_dict)
        for cheap_item in ret_dict:
            target_key = ret_dict[cheap_item][0] + ':' + timestamp # 6214:0319-1403
            target_dict = self.redis.get_dict(target_key)
            item_id = ret_dict[cheap_item][0]
            target_dict['item_name'] = self.__replace_dict[int(item_id)]

            cheap_key = 'cheap_' + cheap_item + ':' + timestamp # cheap_wolf:0319-1403
            print(cheap_key, target_dict)
            self.redis.set_dict(cheap_key, target_dict, expire_time=172800)


    # not use
    def get_hotlist(self):
        uri = self.__base_url + 'GetWorldMarketHotList'

    def _get_market_list(self, target_category: str) -> dict:
        '''
            input: (필수) keyType / (필수) mainCategory / subCategory(unpack을 위해선 필수)
            return: itemID - currentStock - totalTrades - basePrice |
        '''
        uri = self.__base_url + 'GetWorldMarketList'
        payload = self.__payloads[target_category]
        basic_market_dict = {}

        response = requests.request('POST', uri, json=payload, headers=self.__headers)
        if self._status_code_not_200(response.status_code):
            return False
        
        response = unpack(response.content)[:-1]
        datalist = response.split('|')
        for data in datalist:
            item_result = data.split('-')
            item_id = item_result[0]
            basic_market_dict[item_id] = {}
            basic_market_dict[item_id]['current_stock'] = item_result[1]
            basic_market_dict[item_id]['total_trades'] = item_result[2]
            basic_market_dict[item_id]['base_price'] = item_result[3]

        return basic_market_dict


    def _get_market_sublist(self, item_id):
        uri = self.__base_url + 'GetWorldMarketSubList'
        payload = {'keyType':0, 'mainKey':item_id}
        response = requests.request('POST', uri, json=payload, headers=self.__headers) # plane text
        res_list = json.loads(response.text)['resultMsg'][:-1].split('-')
        ret_dict = {}
        ret_dict['id'] = res_list[0]
        ret_dict['min_enhance'] = res_list[1]
        ret_dict['max_enhance'] = res_list[2]
        ret_dict['base_price'] = res_list[3]
        ret_dict['current_stock'] = res_list[4]
        ret_dict['total_trades'] = res_list[5]
        ret_dict['price_hardcap_min'] = res_list[6]
        ret_dict['price_hardcap_max'] = res_list[7]
        ret_dict['last_sale_price'] = res_list[8]
        ret_dict['last_sale_time'] = res_list[9]

        return ret_dict

    # not use?
    def _get_search_list(self):
        uri = self.__base_url + 'GetWorldMarketSearchList'

    def _get_bid_price(self, item_id: int, grade=0) -> tuple[str, str]:
        """
        자주 봐야 하는 녀석들 위주로만 temid 요청하기 (장비 등은 필요 없음)
        return: (판매대기 최하층 값, 구매대기 최상층 값)
        """
        uri = self.__base_url + 'GetBiddingInfoList'
        payload = {'keyType':0, 'mainKey':item_id, 'subKey': grade}
        response = requests.request('POST', uri, json=payload, headers=self.__headers)
        if self._status_code_not_200(response.status_code):
            return False
        
        response = unpack(response.content)[:-1]
        sale_dict = {}
        buy_dict = {}
        for bid in response.split('|'):
            price_sale_buy = bid.split('-')
            sale_dict[price_sale_buy[0]] = price_sale_buy[1]
            buy_dict[price_sale_buy[0]] = price_sale_buy[2]

        sale_price = max(sale_dict)
        price_list = sorted(sale_dict)
        for price in price_list:
            if sale_dict[price] != '0':
                sale_price = price
                break
        
        buy_price = min(buy_dict)
        price_list = sorted(buy_dict, reverse=True)
        for price in price_list:
            if buy_dict[price] != '0':
                buy_price = price
                break

        return sale_price, buy_price

    def _get_market_price_info(self):
        uri = self.__base_url + 'GetMarketPriceInfo'

    def _get_wait_list(self):
        uri = self.__base_url + 'GetWorldMarketWaitList'



    def set_last_trade_price(self, target_category, timestamp): # per 2~5 minute for statistics
        # id 1500개
        # 키 개수 = 2
        # 하루치 = 24
        # 2분단위 = x30: 2,160,000
        # 5분단위 = x12: 864,000
        # if not self.redis:
        #     return False
        
        subcategory_list_contents = self._get_market_list(target_category)
        
        for id in subcategory_list_contents:
            bidding_sale, bidding_buy = self._get_bid_price(id)
            sublist = self._get_market_sublist(id)

            item_info_dict = {
                "current_stock": sublist['current_stock'],
                "last_sale_price": sublist['last_sale_price'],
                "total_trades": sublist['total_trades'],
                "bid_sale_price": bidding_sale,
                "bid_buy_price": bidding_buy
            }
            print(id, item_info_dict)
            # self.redis.set_dict(str(id) + ':' + timestamp, item_info_dict, 172800) # 2days
    
    def set_cheapest_item(self, timestamp):
        self._choose_cheapest_item(timestamp)
        self.redis.set('last_setting_timestamp', timestamp, expire_time=172800)





def market_list(target_category: str) -> dict:
    base_url='https://trade.kr.playblackdesert.com/Trademarket/'
    uri = base_url + 'GetWorldMarketList'
    payload = payloads[target_category]
    headers = {'Content-Type': 'application/json', 'User-Agent': 'BlackDesert'}
    basic_market_dict = {}
    response = requests.request('POST', uri, json=payload, headers=headers)
    if response.status_code != 200:
        return False
    
    response = unpack(response.content)[:-1]
    datalist = response.split('|')
    for data in datalist:
        item_result = data.split('-')
        item_id = item_result[0]
        basic_market_dict[item_id] = {}
        basic_market_dict[item_id]['current_stock'] = item_result[1]
        basic_market_dict[item_id]['total_trades'] = item_result[2]
        basic_market_dict[item_id]['base_price'] = item_result[3]
    return basic_market_dict