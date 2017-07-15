package crawlee

func Start() {
	market := &Shopee{}
	market.Init(gcfg.MongoHost, "SG")
}
