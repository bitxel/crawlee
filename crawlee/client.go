package crawlee

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var (
	ErrHttpStatusCode = errors.New("response http status not 200")
	httpClient        *http.Client
	maxRetries        = 0
	config            = &pconfig{
		RefreshUserAgent: false,
		UserAgent:        "Mozilla/5.0 AppleWebKit/529.86 (KHTML, like Gecko) Chrome/54.1.2280.43 Safari/533.36",
	}
)

type pconfig struct {
	RefreshUserAgent bool
	UserAgent        string
}

func (conf *pconfig) renewUserAgent() {
	conf.UserAgent = fmt.Sprintf("Mozilla/5.0 AppleWebKit/%d.%d (KHTML, like Gecko) Chrome/%d.1.%d.%d Safari/%d.%d",
		529+rand.Intn(10), 80+rand.Intn(10), 54+rand.Intn(10), 2200+rand.Intn(100), 40+rand.Intn(10),
		533+rand.Intn(10), 35+rand.Intn(10),
	)
}

func (conf *pconfig) GetUserAgent() string {
	if conf.RefreshUserAgent {
		conf.renewUserAgent()
	}
	return conf.UserAgent
}

func POST(url string, body io.Reader) (data []byte, err error) {
	return do("POST", url, body, nil)
}

func POSTX(url string, body io.Reader, header http.Header) (data []byte, err error) {
	return do("POST", url, body, header)
}

func GET(url string) ([]byte, error) {
	return do("GET", url, nil, nil)
}

func GETX(url string, header http.Header) (data []byte, err error) {
	return do("GET", url, nil, header)
}

func do(t string, url string, body io.Reader, header http.Header) (data []byte, err error) {
	var req *http.Request
	var resp *http.Response

	req, err = http.NewRequest(t, url, body)
	if err != nil {
		return
	}

	if header != nil {
		req.Header = header
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", config.GetUserAgent())
	}

	if req.Header.Get("Referer") == "" {
		idx := strings.LastIndex(url, "/")
		if idx >= 8 {
			req.Header.Set("Referer", url[:idx])
		}
	}

	tried := maxRetries
	for {
		resp, err = httpClient.Do(req)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "Client.Timeout") {
			tried = tried - 1
			if tried < 0 {
				break
			}
			continue
		} else {
			break
		}
	}
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("responese status %d not 200", resp.StatusCode))
	}
	data, err = ioutil.ReadAll(resp.Body)
	return
}

func SetTimeoutRetry(count int) {
	maxRetries = count
}

func SetHttpClient(client *http.Client) {
	httpClient = client
	config.renewUserAgent()
}

func init() {
	httpClient = &http.Client{Timeout: 10000 * time.Millisecond}
	tr := &http.Transport{
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 100,
	}
	httpClient.Transport = tr
}
