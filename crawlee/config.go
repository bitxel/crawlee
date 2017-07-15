package crawlee

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

var (
	gcfg *GlobalCfg
)

type GlobalCfg struct {
	MongoHost string               `yaml:"mongo_host"`
	Shopee    map[string]ShopeeCfg `yaml:"shopee"`
}

type ShopeeCfg struct {
	Host          string
	CategoryUrl   string `yaml:"category_url"`
	ItemListUrl   string `yaml:"item_list_url"`
	ItemInfoUrl   string `yaml:"item_info_url"`
	SleepInterval int
}

func InitConfig(filename string) error {
	bt, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	gcfg = &GlobalCfg{}
	err = yaml.Unmarshal(bt, gcfg)
	if err != nil {
		log.Printf("unmarshal cfg err:%s", err)
		return err
	}
	log.Printf("load config: %+v", gcfg)
	return nil
}
