package parser

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

const (
	douyinCookieEnv          = "PARSE_VIDEO_DOUYIN_COOKIE"
	douyinWebDetailEndpoint  = "https://www.douyin.com/aweme/v1/web/aweme/detail/"
	douyinTTWidEndpoint      = "https://ttwid.bytedance.com/ttwid/union/register/"
	douyinWebDetailUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36"
	douyinABogusBrowser      = "1536|742|1536|864|0|0|0|0|1536|864|1536|864|1536|742|24|24|MacIntel"
	douyinABogusAlphabet     = "Dkdpgh2ZmsQB80/MfvV36XI1R45-WUAlEixNLwoqYTOPuzKFjJnry79HbGcaStCe"
)

var douyinABogusInitialState = [8]uint32{
	1937774191,
	1226093241,
	388252375,
	3666478592,
	2842636476,
	372324522,
	3817729613,
	2969243214,
}

var douyinABogusUserAgentCode = [32]byte{
	76, 98, 15, 131, 97, 245, 224, 133,
	122, 199, 241, 166, 79, 34, 90, 191,
	128, 126, 122, 98, 66, 11, 14, 40,
	49, 110, 110, 173, 67, 96, 138, 252,
}

type douyinWebDetailFetcher struct {
	client        *resty.Client
	endpoint      string
	ttwidEndpoint string
	cookie        string
	now           func() time.Time
	random        func() int
}

func (d douYin) fetchNativeVideoDetail(client *resty.Client, videoID string) (gjson.Result, error) {
	fetcher := newDouyinWebDetailFetcher(client, os.Getenv(douyinCookieEnv))
	return fetcher.fetch(videoID)
}

func newDouyinWebDetailFetcher(client *resty.Client, cookie string) douyinWebDetailFetcher {
	if client == nil {
		client = newClient()
	}

	return douyinWebDetailFetcher{
		client:        client,
		endpoint:      douyinWebDetailEndpoint,
		ttwidEndpoint: douyinTTWidEndpoint,
		cookie:        strings.TrimSpace(cookie),
		now:           time.Now,
		random: func() int {
			return rand.Intn(10000)
		},
	}
}

func (f douyinWebDetailFetcher) fetch(videoID string) (gjson.Result, error) {
	if !isDouyinVideoID(videoID) {
		return gjson.Result{}, fmt.Errorf("invalid douyin video id: %q", videoID)
	}
	if f.client == nil {
		return gjson.Result{}, errors.New("douyin detail client is nil")
	}
	if f.endpoint == "" {
		return gjson.Result{}, errors.New("douyin detail endpoint is empty")
	}

	cookie := f.cookie
	if cookie == "" {
		var err error
		cookie, err = f.fetchAnonymousCookie()
		if err != nil {
			return gjson.Result{}, fmt.Errorf("get anonymous douyin cookie: %w", err)
		}
	}

	query := douyinWebDetailQuery(videoID)
	requestURL := f.endpoint + "?" + query + "&a_bogus=" + f.aBogus(query)
	request := f.client.R().
		SetHeaders(map[string]string{
			HttpHeaderUserAgent: douyinWebDetailUserAgent,
			HttpHeaderReferer:   "https://www.douyin.com/",
			"Accept-Language":   "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2",
		})
	if cookie != "" {
		request.SetHeader(HttpHeaderCookie, cookie)
	}

	response, err := request.Get(requestURL)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("request native douyin detail: %w", err)
	}
	if response == nil {
		return gjson.Result{}, errors.New("native douyin detail returned empty response")
	}
	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return gjson.Result{}, fmt.Errorf("native douyin detail returned status %d", response.StatusCode())
	}

	data := gjson.GetBytes(response.Body(), "aweme_detail")
	if !data.Exists() {
		return gjson.Result{}, fmt.Errorf(
			"native douyin detail did not return aweme_detail: %s",
			strings.TrimSpace(gjson.GetBytes(response.Body(), "status_msg").String()),
		)
	}
	if awemeID := data.Get("aweme_id").String(); awemeID != videoID {
		return gjson.Result{}, fmt.Errorf(
			"native douyin detail returned unexpected aweme id %q (status_code=%s, status_msg=%q)",
			awemeID,
			gjson.GetBytes(response.Body(), "status_code").String(),
			strings.TrimSpace(gjson.GetBytes(response.Body(), "status_msg").String()),
		)
	}

	return data, nil
}

func (f douyinWebDetailFetcher) fetchAnonymousCookie() (string, error) {
	if f.client == nil {
		return "", errors.New("douyin detail client is nil")
	}
	if f.ttwidEndpoint == "" {
		return "", errors.New("douyin ttwid endpoint is empty")
	}

	response, err := f.client.R().
		SetHeader(HttpHeaderContentType, "application/json").
		SetBody(map[string]any{
			"region":        "cn",
			"aid":           6383,
			"needFid":       false,
			"service":       "www.douyin.com",
			"migrate_info":  map[string]string{"ticket": "", "source": "node"},
			"cbUrlProtocol": "https",
			"union":         true,
		}).
		Post(f.ttwidEndpoint)
	if err != nil {
		return "", fmt.Errorf("register ttwid: %w", err)
	}
	if response == nil {
		return "", errors.New("ttwid registration returned empty response")
	}
	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ttwid registration returned status %d", response.StatusCode())
	}
	if statusCode := gjson.GetBytes(response.Body(), "status_code"); !statusCode.Exists() || statusCode.Int() != 0 {
		return "", fmt.Errorf(
			"ttwid registration failed: status_code=%s, message=%q",
			statusCode.String(),
			strings.TrimSpace(gjson.GetBytes(response.Body(), "message").String()),
		)
	}

	for _, cookie := range response.Cookies() {
		if cookie.Name == "ttwid" && strings.TrimSpace(cookie.Value) != "" {
			return (&http.Cookie{Name: cookie.Name, Value: cookie.Value}).String(), nil
		}
	}
	return "", errors.New("ttwid registration did not return the ttwid cookie")
}

func (f douyinWebDetailFetcher) aBogus(query string) string {
	now := f.now
	if now == nil {
		now = time.Now
	}
	randomNumber := f.random
	if randomNumber == nil {
		randomNumber = func() int { return rand.Intn(10000) }
	}

	started := now().UnixMilli()
	finished := started + int64(4+randomNumber()%5)
	return makeDouyinABogus(query, "GET", started, finished, randomNumber(), randomNumber(), randomNumber())
}

func douyinWebDetailQuery(videoID string) string {
	params := []struct {
		key   string
		value string
	}{
		{"device_platform", "webapp"},
		{"aid", "6383"},
		{"channel", "channel_pc_web"},
		{"pc_client_type", "1"},
		{"version_code", "290100"},
		{"version_name", "29.1.0"},
		{"cookie_enabled", "true"},
		{"screen_width", "1920"},
		{"screen_height", "1080"},
		{"browser_language", "zh-CN"},
		{"browser_platform", "Win32"},
		{"browser_name", "Chrome"},
		{"browser_version", "130.0.0.0"},
		{"browser_online", "true"},
		{"engine_name", "Blink"},
		{"engine_version", "130.0.0.0"},
		{"os_name", "Windows"},
		{"os_version", "10"},
		{"cpu_core_num", "12"},
		{"device_memory", "8"},
		{"platform", "PC"},
		{"downlink", "10"},
		{"effective_type", "4g"},
		{"from_user_page", "1"},
		{"locate_query", "false"},
		{"need_time_list", "1"},
		{"pc_libra_divert", "Windows"},
		{"publish_video_strategy_type", "2"},
		{"round_trip_time", "0"},
		{"show_live_replay_strategy", "1"},
		{"time_list_query", "0"},
		{"whale_cut_token", ""},
		{"update_version_code", "170400"},
		{"msToken", ""},
		{"aweme_id", videoID},
	}

	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, url.QueryEscape(param.key)+"="+url.QueryEscape(param.value))
	}
	return strings.Join(parts, "&")
}

func makeDouyinABogus(query, method string, started, finished int64, random1, random2, random3 int) string {
	paramsHash := douyinDoubleSM3(query + "cus")
	methodHash := douyinDoubleSM3(method + "cus")
	payload := []rune{
		44,
		rune((finished >> 24) & 255),
		0, 0, 0, 0,
		24,
		rune(paramsHash[21]),
		rune(methodHash[21]),
		0,
		rune(douyinABogusUserAgentCode[23]),
		rune((finished >> 16) & 255),
		0, 0, 0,
		1,
		0,
		239,
		rune(paramsHash[22]),
		rune(methodHash[22]),
		rune(douyinABogusUserAgentCode[24]),
		rune((finished >> 8) & 255),
		0, 0, 0, 0,
		rune(finished & 255),
		0, 0,
		14,
		rune((started >> 24) & 255),
		rune((started >> 16) & 255),
		0,
		rune((started >> 8) & 255),
		3,
		rune(finished >> 32),
		1,
		rune(started >> 32),
		1,
		rune(len(douyinABogusBrowser)),
		0, 0, 0,
	}

	checksum := 0
	for _, value := range payload {
		checksum ^= int(value)
	}
	payload = append(payload, []rune(douyinABogusBrowser)...)
	payload = append(payload, rune(checksum))

	prefix := append(douyinABogusRandomGroup(random1, 1, 2, 5, 45&170), douyinABogusRandomGroup(random2, 1, 0, 0, 0)...)
	prefix = append(prefix, douyinABogusRandomGroup(random3, 1, 0, 5, 0)...)
	ciphertext := douyinABogusRC4(payload)
	return douyinABogusEncode(append(prefix, ciphertext...))
}

func douyinABogusRandomGroup(value, extra1, extra2, extra3, extra4 int) []rune {
	low := value & 255
	high := (value >> 8) & 255
	return []rune{
		rune((low & 170) | extra1),
		rune((low & 85) | extra2),
		rune((high & 170) | extra3),
		rune((high & 85) | extra4),
	}
}

func douyinABogusRC4(plaintext []rune) []rune {
	state := make([]int, 256)
	for i := range state {
		state[i] = i
	}

	position := 0
	for i := range state {
		position = (position + state[i] + int('y')) % len(state)
		state[i], state[position] = state[position], state[i]
	}

	position = 0
	result := make([]rune, len(plaintext))
	for i, value := range plaintext {
		index := (i + 1) % len(state)
		position = (position + state[index]) % len(state)
		state[index], state[position] = state[position], state[index]
		result[i] = rune(state[(state[index]+state[position])%len(state)] ^ int(value))
	}
	return result
}

func douyinABogusEncode(values []rune) string {
	var result strings.Builder
	for index := 0; index < len(values); index += 3 {
		var number uint32
		number = uint32(values[index]) << 16
		if index+1 < len(values) {
			number |= uint32(values[index+1]) << 8
		}
		if index+2 < len(values) {
			number |= uint32(values[index+2])
		}

		result.WriteByte(douyinABogusAlphabet[(number>>18)&63])
		result.WriteByte(douyinABogusAlphabet[(number>>12)&63])
		if index+1 < len(values) {
			result.WriteByte(douyinABogusAlphabet[(number>>6)&63])
		}
		if index+2 < len(values) {
			result.WriteByte(douyinABogusAlphabet[number&63])
		}
	}

	for result.Len()%4 != 0 {
		result.WriteByte('=')
	}
	return result.String()
}

func douyinDoubleSM3(value string) [32]byte {
	first := douyinSM3([]byte(value))
	return douyinSM3(first[:])
}

func douyinSM3(message []byte) [32]byte {
	padded := append([]byte(nil), message...)
	bitLength := uint64(len(padded)) * 8
	padded = append(padded, 0x80)
	for len(padded)%64 != 56 {
		padded = append(padded, 0)
	}
	length := make([]byte, 8)
	binary.BigEndian.PutUint64(length, bitLength)
	padded = append(padded, length...)

	state := douyinABogusInitialState
	for offset := 0; offset < len(padded); offset += 64 {
		var words [68]uint32
		for i := 0; i < 16; i++ {
			words[i] = binary.BigEndian.Uint32(padded[offset+i*4 : offset+(i+1)*4])
		}
		for i := 16; i < len(words); i++ {
			words[i] = douyinSM3P1(words[i-16]^words[i-9]^bits.RotateLeft32(words[i-3], 15)) ^ bits.RotateLeft32(words[i-13], 7) ^ words[i-6]
		}

		var expanded [64]uint32
		for i := range expanded {
			expanded[i] = words[i] ^ words[i+4]
		}

		a, b, c, d, e, f, g, h := state[0], state[1], state[2], state[3], state[4], state[5], state[6], state[7]
		for i := 0; i < 64; i++ {
			constant := uint32(0x79CC4519)
			if i >= 16 {
				constant = 0x7A879D8A
			}
			ss1 := bits.RotateLeft32(bits.RotateLeft32(a, 12)+e+bits.RotateLeft32(constant, i), 7)
			ss2 := ss1 ^ bits.RotateLeft32(a, 12)

			ff := a ^ b ^ c
			gg := e ^ f ^ g
			if i >= 16 {
				ff = (a & b) | (a & c) | (b & c)
				gg = (e & f) | ((^e) & g)
			}

			tt1 := ff + d + ss2 + expanded[i]
			tt2 := gg + h + ss1 + words[i]
			d, c, b, a = c, bits.RotateLeft32(b, 9), a, tt1
			h, g, f, e = g, bits.RotateLeft32(f, 19), e, douyinSM3P0(tt2)
		}

		state[0] ^= a
		state[1] ^= b
		state[2] ^= c
		state[3] ^= d
		state[4] ^= e
		state[5] ^= f
		state[6] ^= g
		state[7] ^= h
	}

	var digest [32]byte
	for i, value := range state {
		binary.BigEndian.PutUint32(digest[i*4:], value)
	}
	return digest
}

func douyinSM3P0(value uint32) uint32 {
	return value ^ bits.RotateLeft32(value, 9) ^ bits.RotateLeft32(value, 17)
}

func douyinSM3P1(value uint32) uint32 {
	return value ^ bits.RotateLeft32(value, 15) ^ bits.RotateLeft32(value, 23)
}

func isDouyinVideoID(value string) bool {
	if len(value) < 10 || len(value) > 32 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
