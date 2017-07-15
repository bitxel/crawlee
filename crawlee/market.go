package crawlee

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"time"
)

const (
	ItemPerPageLimit = 100
)

var (
	ErrCountryNotExist = errors.New("country not exist")
)

type Market interface {
	Init(string) error
	// GetCatgoryInfo
	GetCategory() error
	// GetItemList fetch newest item list from market
	GetItemList() error
	// GetItemInfo update item info of existing items in db every day
	GetItemInfo() error
}

type Shopee struct {
	cfg     *ShopeeCfg
	Host    string
	Country string
	db      *mgo.Database
}

type ShopeeCat struct {
	Main ShopeeCatDetail
	Sub  []ShopeeCatDetail
}

type ShopeeCatDetail struct {
	DisplayName string `bson:"display_name" json:"display_name"`
	CatId       int    `bson:"catid" json:"catid"`
	ParentId    int    `bson:"parent_category" json:"parent_category"`
}

type ShopeeItemList struct {
	Items []ShopeeItemMeta
}

type ShopeeItemMeta struct {
	Itemid int `bson:"itemid" json:"itemid"`
	Shopid int `bson:"shopid" json:"shopid"`
}

func (s *Shopee) Init(mgoaddr string, country string) (err error) {
	cfg, ok := gcfg.Shopee[country]
	if !ok {
		log.Printf("country %s not exist", country)
		return ErrCountryNotExist
	}
	s.cfg = &cfg
	s.Country = country
	sess, err := mgo.Dial(mgoaddr)
	if err != nil {
		log.Printf("dial mongodb err:%s", err)
		return err
	}
	s.db = sess.DB(fmt.Sprintf("shopee_%s", country))
	err = s.GetCategory()
	if err != nil {
		log.Printf("get category list err: %s", err)
		return err
	}
	log.Println("Begin to get itemlist")
	err = s.GetItemList()
	if err != nil {
		log.Printf("get item list err: %s", err)
		return
	}
	s.GetItemInfo()
	return nil
}

/*
	url: https://shopee.sg/api/v1/category_list/
	method: GET
*/
func (s *Shopee) GetCategory() (err error) {
	resp, err := GET(s.cfg.CategoryUrl)
	if err != nil {
		return err
	}
	cat := make([]ShopeeCat, 0)
	err = json.Unmarshal(resp, &cat)
	if err != nil {
		log.Printf("unmarshal err:%s", err)
		return err
	}
	log.Printf("cat list: %v", cat)

	c := s.db.C("category")
	for _, value := range cat {
		// Save main
		_, err = c.Upsert(bson.M{"catid": value.Main.CatId}, value.Main)
		if err != nil {
			log.Printf("upsert err: %s, origin:%+v", err, value)
			return
		}
		// Save sub
		for _, v := range value.Sub {
			_, err = c.Upsert(bson.M{"catid": v.CatId}, v)
			if err != nil {
				log.Printf("upsert err: %s, origin:%+v", err, v)
				return
			}
		}
	}
	return
}

/*
	url: https://shopee.sg/api/v1/search_items/?by=pop&order=desc&categoryids=6&newest=0&limit=50
	method: GET
*/
func (s *Shopee) GetItemList() (err error) {
	cats, err := s.loadCategory()
	if err != nil {
		return
	}
	for _, cat := range cats {
		offset := 0
		for {
			count, err := s.fetchItemListByPage(cat.CatId, offset, ItemPerPageLimit)
			if err != nil {
				log.Printf("fetch item list err: %s, catdi: %d, offset: %d", err, cat.CatId, offset)
				continue
			}
			if count < ItemPerPageLimit {
				log.Println("load count less than limit, maybe finished")
				break
			}
			offset += ItemPerPageLimit
		}
	}
	return
}

func (s *Shopee) loadCategory() (cats []ShopeeCatDetail, err error) {
	ccat := s.db.C("category")
	//err = ccat.Find(bson.M{"parent_category": 0}).All(&cats)
	err = ccat.Find(bson.M{"parent_category": bson.M{"$ne": 0}}).All(&cats)
	if err != nil {
		log.Printf("load cat err: %s", err)
		return
	}
	log.Printf("load %d cats", len(cats))
	return
}

func (s *Shopee) fetchItemListByPage(catid, offset, limit int) (count int, err error) {
	resp, err := GET(fmt.Sprintf(s.cfg.ItemListUrl, catid, offset, limit))
	if err != nil {
		return
	}
	itemlist := &ShopeeItemList{}
	err = json.Unmarshal(resp, itemlist)
	if err != nil {
		return
	}
	count = len(itemlist.Items)
	log.Printf("fetch item list: catid: %d offset: %d limit: %d result: %d", catid, offset, limit, count)
	err = s.fetchItemInfoBatch(itemlist.Items)
	return
}

func (s *Shopee) fetchItemInfoBatch(meta []ShopeeItemMeta) (err error) {
	header := make(http.Header)
	header.Set("Referer", "https://shopee.sg/")
	header.Set("x-csrftoken", "OeEMDEn0j07E2wDok1lkKX3dCKGuxSi")
	header.Set("Cookie", "csrftoken=OeEMDEn0j07E2wDok1lkKX3dCKGuxSi; REC_T_ID=18bb56c6-67b5-11e6-a32b-d4ae52b94876; SPC_T_ID=\"NC6JtDKNuFxPQW+RUwEsrEt7qUSXKjbQ4UQbwAlLzUUQBIkJuoPbDg3zmtEmEUtNqYA9A3hm5YDBi0GhoqghCfojat7bJeJJINRMqvt41uA=\"; SPC_T_IV=\"YSo6+ObUwOHrKq4WFm5+cw==\"; SPC_T_F=1; django_language=en; _atrk_siteuid=1Yst3vGhS1WbWRG4; sessionid=qk6mqa0i4zm1edulky0cn4xm2dtwta2xfkbwjatjj2b")
	bt, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	bt = append([]byte("{\"item_shop_ids\":"), bt...)
	bt = append(bt, []byte("}")...)
	buf := bytes.NewBuffer(bt)
	resp, err := POSTX(s.cfg.ItemInfoUrl, buf, header)
	if err != nil {
		return
	}
	//log.Println(string(resp))
	itemInfo := make([]map[string]interface{}, 0)
	if err = json.Unmarshal(resp, &itemInfo); err != nil {
		return
	}
	//log.Printf("%+v", itemInfo)
	for _, item := range itemInfo {
		s.saveItemInfo(int(item["itemid"].(float64)), item)
	}
	return
}

func (s *Shopee) saveItemInfo(itemid int, item map[string]interface{}) (err error) {
	c := s.db.C("item")
	dt := Date()
	bs := bson.M(item)
	bs["_date"] = dt
	if _, err = c.Upsert(bson.M{"itemid": itemid}, bson.M{"$set": bson.M{"itemid": itemid, "mtime": time.Now().Unix()}}); err != nil {
		return
	}
	count, err := c.Find(bson.M{"itemid": itemid, "history._date": dt}).Count()
	if err != nil || count > 0 {
		return
	}
	err = c.Update(bson.M{"itemid": itemid}, bson.M{"$push": bson.M{"history": bs}})
	if err != nil {
		log.Printf("push to item err: %s", err)
	}
	return
}

/*
	url: https://shopee.sg/api/v1/items/
	method: POST
	payload: """
		{"item_shop_ids":[
			{"itemid":11380339,"adsid":0,"shopid":85844,"campaignid":0},
			{"itemid":20696111,"adsid":0,"shopid":1844382,"campaignid":0},
			{"itemid":10480762,"adsid":0,"shopid":177248,"campaignid":0},
			{"itemid":12410819,"adsid":0,"shopid":1969167,"campaignid":0}]}
		"""
*/
func (s *Shopee) GetItemInfo() (err error) {
	return
}

func Date() string {
	return time.Now().Format("2006-01-02")
}
