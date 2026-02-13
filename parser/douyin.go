package parser

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

type douYin struct{}

func (d douYin) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := fmt.Sprintf("https://www.iesdouyin.com/share/video/%s", videoId)

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1").
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	isNote := false
	resBody := res.Body()
	canonical, err := d.getCanonicalFromHTML(string(resBody))
	if err == nil && canonical != "" {
		//判断字符串中是否有 /note/ 字符
		if strings.Contains(canonical, "/note/") {
			isNote = true
		}
	}

	var jsonBytes []byte
	var data gjson.Result

	//获取图集
	if isNote {
		webId := "75" + d.generateFixedLengthNumericID(15)
		aBogus := d.randSeq(64)

		reqUrl = fmt.Sprintf("https://www.iesdouyin.com/web/api/v2/aweme/slidesinfo/?reflow_source=reflow_page&web_id=%s&device_id=%s&aweme_ids=%%5B%s%%5D&request_source=200&a_bogus=%s", webId, webId, videoId, aBogus)
		res, err = client.R().
			SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1").
			Get(reqUrl)
		if err != nil {
			return nil, err
		}

		jsonBytes = res.Body()
		data = gjson.GetBytes(jsonBytes, "aweme_details.0")
		if !data.Exists() {
			//fmt.Println(reqUrl, data)
			//设置为，好让下面判断
			isNote = false
		}
	}

	if !isNote {
		re := regexp.MustCompile(`window._ROUTER_DATA\s*=\s*(.*?)</script>`)
		findRes := re.FindSubmatch(resBody)
		if len(findRes) < 2 {
			return nil, errors.New("parse video json info from html fail")
		}

		jsonBytes = bytes.TrimSpace(findRes[1])
		data = gjson.GetBytes(jsonBytes, "loaderData.video_(id)/page.videoInfoRes.item_list.0")
	}

	if !data.Exists() {
		filterObj := gjson.GetBytes(
			jsonBytes,
			fmt.Sprintf(`loaderData.video_(id)/page.videoInfoRes.filter_list.#(aweme_id=="%s")`, videoId),
		)

		return nil, fmt.Errorf(
			"get video info fail: %s - %s",
			filterObj.Get("filter_reason"),
			filterObj.Get("detail_msg"),
		)
	}

	// 获取图集图片地址
	imagesObjArr := data.Get("images").Array()
	images := make([]ImgInfo, 0, len(imagesObjArr))
	for _, imageItem := range imagesObjArr {
		urlList := imageItem.Get("url_list").Array()
		// 优先获取非 .webp 格式的图片 url
		imageUrl := d.getNoWebpUrl(urlList)
		if len(imageUrl) > 0 {
			images = append(images, ImgInfo{
				Url:          imageUrl,
				LivePhotoUrl: imageItem.Get("video.play_addr.url_list.0").String(),
			})
		}
	}

	var videoUrl string
	if !isNote {
		// 获取视频播放地址
		videoUrl = data.Get("video.play_addr.url_list.0").String()
		videoUrl = strings.ReplaceAll(videoUrl, "playwm", "play")
		data.Get("video.play_addr.url_list").ForEach(func(key, value gjson.Result) bool {
			//fmt.Println(strings.ReplaceAll(value.String(), "playwm", "play"))
			return true
		})
	}

	// 获取音频地址（图集时，video.play_addr.uri 是音频地址；视频时不是音频）
	musicUrl := data.Get("video.play_addr.uri").String()

	// 如果图集地址不为空时，因为没有视频，上面抖音返回的视频地址无法访问，置空处理
	// 图集时，musicUrl 是音频地址；视频时，musicUrl 不是音频，置空
	if len(images) > 0 {
		videoUrl = ""
	} else {
		musicUrl = ""
	}

	urlList := data.Get("video.cover.url_list").Array()
	// 优先获取非 .webp 格式的图片 url
	coverUrl := d.getNoWebpUrl(urlList)

	videoInfo := &VideoParseInfo{
		Title:    data.Get("desc").String(),
		VideoUrl: videoUrl,
		MusicUrl: musicUrl,
		//CoverUrl: data.Get("video.cover.url_list.0").String(),
		CoverUrl: coverUrl,
		Images:   images,
	}
	videoInfo.Author.Uid = data.Get("author.sec_uid").String()
	videoInfo.Author.Name = data.Get("author.nickname").String()
	videoInfo.Author.Avatar = data.Get("author.avatar_thumb.url_list.0").String()

	// 视频地址非空时，获取302重定向之后的视频地址
	// 图集时，视频地址为空，不处理
	if len(videoInfo.VideoUrl) > 0 {
		d.getRedirectUrl(videoInfo)
	}

	if videoInfo.VideoUrl == "" && len(videoInfo.Images) == 0 {
		return nil, errors.New("没有作品")
	}

	return videoInfo, nil
}

func (d douYin) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlRes, err := url.Parse(shareUrl)
	if err != nil {
		return nil, err
	}

	switch urlRes.Host {
	case "www.iesdouyin.com", "www.douyin.com":
		return d.parsePcShareUrl(shareUrl) // 解析电脑网页端链接
	case "v.douyin.com":
		return d.parseAppShareUrl(shareUrl) // 解析App分享链接
	}

	return nil, fmt.Errorf("douyin not support this host: %s", urlRes.Host)
}

func (d douYin) parseAppShareUrl(shareUrl string) (*VideoParseInfo, error) {
	// 适配App分享链接类型:
	// https://v.douyin.com/xxxxxx/

	client := resty.New()
	// disable redirects in the HTTP client, get params before redirects
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	// 非 resty.ErrAutoRedirectDisabled 错误时，返回错误
	if !errors.Is(err, resty.ErrAutoRedirectDisabled) {
		return nil, err
	}

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId, err := d.parseVideoIdFromPath(locationRes.Path)
	if err != nil {
		return nil, err
	}
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	// 西瓜视频解析方式不一样
	if strings.Contains(locationRes.Host, "ixigua.com") {
		return xiGua{}.parseVideoID(videoId)
	}

	return d.parseVideoID(videoId)
}

func (d douYin) parsePcShareUrl(shareUrl string) (*VideoParseInfo, error) {
	// 适配电脑网页端链接类型
	// https://www.iesdouyin.com/share/video/xxxxxx/
	// https://www.douyin.com/video/xxxxxx
	videoId, err := d.parseVideoIdFromPath(shareUrl)
	if err != nil {
		return nil, err
	}
	return d.parseVideoID(videoId)
}

func (d douYin) parseVideoIdFromPath(urlPath string) (string, error) {
	if len(urlPath) <= 0 {
		return "", errors.New("url path is empty")
	}

	urlPathParse, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	//判断网页精选页面的视频
	//https://www.douyin.com/jingxuan?modal_id=7555093909760789812
	videoId := urlPathParse.Query().Get("modal_id")

	if len(videoId) > 0 {
		return videoId, nil
	}

	//判断其他页面的视频
	//https://www.iesdouyin.com/share/video/7424432820954598707/?region=CN&mid=7424432976273869622&u_code=0
	urlPath = strings.Trim(urlPathParse.Path, "/")
	urlSplit := strings.Split(urlPath, "/")

	// 获取最后一个元素
	if len(urlSplit) > 0 {
		return urlSplit[len(urlSplit)-1], nil
	}

	return "", errors.New("parse video id from path fail")
}

func (d douYin) getRedirectUrl(videoInfo *VideoParseInfo) {
	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res2, _ := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(videoInfo.VideoUrl)
	locationRes, _ := res2.RawResponse.Location()
	if locationRes != nil {
		(*videoInfo).VideoUrl = locationRes.String()
	}
}

func (d douYin) randSeq(n int) string {
	letters := []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// 生成固定位数的随机数字（前导零）
func (d douYin) generateFixedLengthNumericID(length int) string {
	// 创建一个新的随机数生成器源
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	max2 := int64(1)
	for i := 0; i < length; i++ {
		max2 *= 10
	}

	randomNum := r.Int63n(max2)
	return fmt.Sprintf("%0*d", length, randomNum)
}

// 优先获取非 .webp 格式的图片 url
func (d douYin) getNoWebpUrl(urlList []gjson.Result) string {
	var imageUrl string
	// 手动遍历查找包含 .jpeg 或 .png 的 URL
	found := false
	for _, urllink := range urlList {
		urlStr := urllink.String()
		//if strings.Contains(urlStr, ".jpeg") || strings.Contains(urlStr, ".png") {
		if !strings.Contains(urlStr, ".webp") {
			imageUrl = urlStr
			found = true
			break
		}
	}

	// 如果没找到，使用第一项
	if !found && len(urlList) > 0 {
		imageUrl = urlList[0].String()
	}

	return imageUrl
}

// 从 HTML 字符串获取 canonical URL
func (d douYin) getCanonicalFromHTML(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	return d.findCanonical(doc), nil
}

// 递归查找 canonical link
func (d douYin) findCanonical(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "link" {
		var rel, href string
		for _, attr := range n.Attr {
			switch attr.Key {
			case "rel":
				rel = attr.Val
			case "href":
				href = attr.Val
			}
		}
		if rel == "canonical" && href != "" {
			return href
		}
	}

	// 递归遍历子节点
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := d.findCanonical(c); result != "" {
			return result
		}
	}

	return ""
}
