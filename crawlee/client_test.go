package crawlee

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHttpGet(t *testing.T) {
	resp, err := GET("https://www.google.com")
	assert.Equal(t, err, nil)
	assert.Contains(t, string(resp), "html")
	_, err = GET("https://www.google.com/asdfasdf")
	assert.Equal(t, err, ErrHttpStatusCode)
}

func TestHttpPost(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("{\"shop_ids\":[12168168,10477,11441031]}")
	header := make(http.Header)
	header.Set("Referer", "https://shopee.sg/")
	header.Set("x-csrftoken", "OeEMDEn0j07E2wDok1lkKX3dCKGuxSi")
	header.Set("Cookie", "csrftoken=OeEMDEn0j07E2wDok1lkKX3dCKGuxSi; REC_T_ID=18bb56c6-67b5-11e6-a32b-d4ae52b94876; SPC_T_ID=\"NC6JtDKNuFxPQW+RUwEsrEt7qUSXKjbQ4UQbwAlLzUUQBIkJuoPbDg3zmtEmEUtNqYA9A3hm5YDBi0GhoqghCfojat7bJeJJINRMqvt41uA=\"; SPC_T_IV=\"YSo6+ObUwOHrKq4WFm5+cw==\"; SPC_T_F=1; django_language=en; _atrk_siteuid=1Yst3vGhS1WbWRG4; sessionid=qk6mqa0i4zm1edulky0cn4xm2dtwta2xfkbwjatjj2b")
	resp, err := POSTX("http://shopee.sg/api/v1/shops/", &buf, header)
	assert.Equal(t, err, nil)
	assert.Contains(t, string(resp), "username")

	_, err = POST("https://www.google.com/", &buf)
	assert.Equal(t, err, ErrHttpStatusCode)
}
