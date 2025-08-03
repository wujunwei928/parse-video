package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// 添加Cookie可以爬取更高清的视频，记得要把下面请求里的Cookie的注释也去掉
// const BiliCookie = `_uuid=; buvid_fp=; buvid4=; SESSDATA=; bili_jct=; DedeUserID=;`

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

type biliBili struct{}

func (b biliBili) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	bvid, err := b.getBvidFromURL(shareUrl)
	if err != nil {
		return nil, fmt.Errorf("无法提取BVID: %w", err)
	}
	viewAPIURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvid)
	viewRespBytes, err := b.sendBiliRequest(viewAPIURL)
	if err != nil {
		return nil, fmt.Errorf("请求视频信息API失败: %w", err)
	}
	var viewResp biliViewResponse
	if err := json.Unmarshal(viewRespBytes, &viewResp); err != nil {
	}
	if viewResp.Code != 0 || len(viewResp.Data.Pages) == 0 {
	}
	firstPageCID := viewResp.Data.Pages[0].Cid

	playAPIURL := fmt.Sprintf(
		"https://api.bilibili.com/x/player/playurl?otype=json&fnver=0&fnval=0&qn=80&bvid=%s&cid=%d&platform=html5",
		bvid, firstPageCID,
	)

	playRespBytes, err := b.sendBiliRequest(playAPIURL)
	if err != nil {
		return nil, fmt.Errorf("请求播放链接API失败: %w", err)
	}

	var playResp biliPlayURLResponse
	if err := json.Unmarshal(playRespBytes, &playResp); err != nil {
		return nil, fmt.Errorf("解析播放链接响应失败: %w", err)
	}
	if playResp.Code != 0 {
		return nil, fmt.Errorf("B站API返回错误: %s (code: %d)", playResp.Message, playResp.Code)
	}

	if len(playResp.Data.Durl) > 0 && playResp.Data.Durl[0].URL != "" {
		finalVideoURL := playResp.Data.Durl[0].URL

		videoInfo := &VideoParseInfo{
			Title:    viewResp.Data.Title,
			VideoUrl: finalVideoURL,
			CoverUrl: viewResp.Data.Pic,
			Images:   make([]ImgInfo, 0),
		}
		videoInfo.Author.Uid = fmt.Sprintf("%d", viewResp.Data.Owner.Mid)
		videoInfo.Author.Name = viewResp.Data.Owner.Name
		videoInfo.Author.Avatar = viewResp.Data.Owner.Face

		return videoInfo, nil
	}

	return nil, fmt.Errorf("无法获取该视频")
}

type biliViewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Bvid  string `json:"bvid"`
		Title string `json:"title"`
		Pic   string `json:"pic"` // 封面图
		Owner struct {
			Mid  int64  `json:"mid"` // 作者UID
			Name string `json:"name"`
			Face string `json:"face"` // 作者头像
		} `json:"owner"`
		Pages []struct {
			Cid int `json:"cid"`
		} `json:"pages"`
	} `json:"data"`
}

type biliPlayURLResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Durl []struct {
			URL string `json:"url"`
		} `json:"durl"`

		Dash struct {
			Video []struct {
				BaseURL   string `json:"baseUrl"`
				Bandwidth int    `json:"bandwidth"`
			} `json:"video"`
			Audio []struct {
				BaseURL   string `json:"baseUrl"`
				Bandwidth int    `json:"bandwidth"`
			} `json:"audio"`
		} `json:"dash"`
	} `json:"data"`
}

func (b biliBili) getBvidFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL格式无效")
	}

	if strings.Contains(parsedURL.Host, "b23.tv") {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Get(rawURL)
		if err != nil {
			return "", fmt.Errorf("请求b23.tv短链失败: %v", err)
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("无法从b23.tv获取重定向链接")
		}
		return b.getBvidFromURL(location)
	}

	if strings.Contains(parsedURL.Host, "bilibili.com") {
		path := strings.Trim(parsedURL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 && parts[0] == "video" {
			if strings.HasPrefix(parts[1], "BV") {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("不是有效的B站视频链接")
}

func (b biliBili) sendBiliRequest(apiURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	// 如需爬取更高清的视频请取消这里的注释
	// req.Header.Set("Cookie", BiliCookie)
	req.Header.Set("Referer", "https://www.bilibili.com/")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败, 状态码: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
