package parser

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

const (
	douyinH5ItemInfoEndpoint = "https://www.iesdouyin.com/web/api/v2/aweme/iteminfo/"
	douyinH5ContextAttempts  = 3
)

var errDouyinH5ReflowContext = errors.New("invalid douyin h5 reflow context")

type douyinH5ReflowContext struct {
	token   string
	webID   string
	userCIP string
}

type douyinH5DetailFetcher struct {
	client   *resty.Client
	endpoint string
}

func (d douYin) fetchNativeH5VideoDetail(
	client *resty.Client,
	sharePageURL string,
	pageHTML []byte,
	cookies []*http.Cookie,
	videoID string,
) (gjson.Result, error) {
	fetcher := newDouyinH5DetailFetcher(client)
	return fetcher.fetchWithContextRetry(videoID, sharePageURL, pageHTML, cookies, douyinH5ContextAttempts)
}

func newDouyinH5DetailFetcher(client *resty.Client) douyinH5DetailFetcher {
	if client == nil {
		client = newClient()
	}
	return douyinH5DetailFetcher{
		client:   client,
		endpoint: douyinH5ItemInfoEndpoint,
	}
}

func (f douyinH5DetailFetcher) fetchWithContextRetry(
	videoID string,
	sharePageURL string,
	pageHTML []byte,
	cookies []*http.Cookie,
	maxAttempts int,
) (gjson.Result, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var contextErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		data, err := f.fetch(videoID, pageHTML, cookies)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, errDouyinH5ReflowContext) {
			return gjson.Result{}, err
		}
		contextErr = err
		if attempt+1 >= maxAttempts {
			break
		}

		response, err := f.client.R().
			SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
			Get(sharePageURL)
		if err != nil {
			return gjson.Result{}, fmt.Errorf("refresh douyin h5 share page after %v: %w", contextErr, err)
		}
		if response == nil {
			return gjson.Result{}, fmt.Errorf("refresh douyin h5 share page after %v: empty response", contextErr)
		}
		if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
			return gjson.Result{}, fmt.Errorf(
				"refresh douyin h5 share page after %v: status %d",
				contextErr,
				response.StatusCode(),
			)
		}
		pageHTML = response.Body()
		cookies = response.Cookies()
	}

	return gjson.Result{}, contextErr
}

func (f douyinH5DetailFetcher) fetch(videoID string, pageHTML []byte, cookies []*http.Cookie) (gjson.Result, error) {
	if !isDouyinVideoID(videoID) {
		return gjson.Result{}, fmt.Errorf("invalid douyin video id: %q", videoID)
	}
	if f.client == nil {
		return gjson.Result{}, errors.New("douyin h5 detail client is nil")
	}
	if f.endpoint == "" {
		return gjson.Result{}, errors.New("douyin h5 detail endpoint is empty")
	}

	context, err := parseDouyinH5ReflowContext(pageHTML)
	if err != nil {
		return gjson.Result{}, err
	}
	reflowID, err := encryptDouyinH5ReflowToken(context.token, context.webID)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("encrypt douyin h5 reflow token: %w", err)
	}

	query := map[string]string{
		"reflow_source":        "reflow_page",
		"web_id":               context.webID,
		"device_id":            context.webID,
		"use_new_select_scope": "0",
		"item_ids":             videoID,
		"reflow_id":            reflowID,
	}
	if context.userCIP != "" {
		query["user_cip"] = context.userCIP
	}

	response, err := f.client.R().
		SetHeaders(map[string]string{
			HttpHeaderUserAgent: DefaultUserAgent,
			HttpHeaderReferer:   fmt.Sprintf("https://www.iesdouyin.com/share/video/%s", videoID),
			"Accept":            "application/json, text/plain, */*",
		}).
		SetCookies(cookies).
		SetQueryParams(query).
		Get(f.endpoint)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("request native douyin h5 detail: %w", err)
	}
	if response == nil {
		return gjson.Result{}, errors.New("native douyin h5 detail returned empty response")
	}
	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return gjson.Result{}, fmt.Errorf("native douyin h5 detail returned status %d", response.StatusCode())
	}

	statusCode := gjson.GetBytes(response.Body(), "status_code")
	if !statusCode.Exists() || statusCode.Int() != 0 {
		return gjson.Result{}, fmt.Errorf(
			"native douyin h5 detail returned status_code=%s, status_msg=%q",
			statusCode.String(),
			strings.TrimSpace(gjson.GetBytes(response.Body(), "status_msg").String()),
		)
	}

	data := gjson.GetBytes(response.Body(), "item_list.0")
	if !data.Exists() {
		return gjson.Result{}, errors.New("native douyin h5 detail did not return item_list")
	}
	if awemeID := data.Get("aweme_id").String(); awemeID != videoID {
		return gjson.Result{}, fmt.Errorf("native douyin h5 detail returned unexpected aweme id %q", awemeID)
	}

	return data, nil
}

func parseDouyinH5ReflowContext(pageHTML []byte) (douyinH5ReflowContext, error) {
	document, err := html.Parse(bytes.NewReader(pageHTML))
	if err != nil {
		return douyinH5ReflowContext{}, fmt.Errorf("%w: parse share page: %v", errDouyinH5ReflowContext, err)
	}

	var context douyinH5ReflowContext
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch douyinHTMLAttribute(node, "id") {
			case "douyin_reflow_token":
				context.token = douyinHTMLAttribute(node, "xsstoken")
			case "douyin_reflow_webId":
				context.webID = douyinHTMLAttribute(node, "webId")
				context.userCIP = douyinHTMLAttribute(node, "usercip")
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

	if context.token == "" {
		return douyinH5ReflowContext{}, fmt.Errorf("%w: share page did not return xsstoken", errDouyinH5ReflowContext)
	}
	if context.webID == "" {
		return douyinH5ReflowContext{}, fmt.Errorf("%w: share page did not return webId", errDouyinH5ReflowContext)
	}
	if webIDLength := len([]byte(context.webID)); webIDLength < aes.BlockSize {
		return douyinH5ReflowContext{}, fmt.Errorf(
			"%w: share page returned an invalid webId length %d",
			errDouyinH5ReflowContext,
			webIDLength,
		)
	}

	return context, nil
}

func douyinHTMLAttribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return strings.TrimSpace(attribute.Val)
		}
	}
	return ""
}

func encryptDouyinH5ReflowToken(token, webID string) (string, error) {
	if token == "" {
		return "", errors.New("douyin h5 reflow token is empty")
	}
	webIDBytes := []byte(webID)
	if len(webIDBytes) < aes.BlockSize {
		return "", errors.New("douyin h5 webId must contain at least 16 bytes")
	}
	key := webIDBytes[:aes.BlockSize]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(token)
	paddingLength := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append(append([]byte(nil), plaintext...), bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)...)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, key).CryptBlocks(ciphertext, padded)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
