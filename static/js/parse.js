// 原有的解析功能
async function setValue() {
    var data = document.getElementById("url").value;
    let regex = /http[s]?:\/\/[\w.-]+[\w\/-]*[\w.-:]*\??[\w=&:\-\+\%.]*[/]*/;
    var v = data.match(regex)[0];

    // 异步请求，避免同步 XHR 阻塞主线程导致 UI 冻结
    var resp = await fetch("/video/share/url/parse?url=" + encodeURIComponent(v));
    var jsonObj = await resp.json();

    if (jsonObj.code == 200) {
        mdui.snackbar({
            message: '解析成功'
        });

        let successHtml = '<h4>' + jsonObj.data.title + ' </h4>';
        successHtml += '<a class="mdui-btn mdui-btn-raised" href="' + jsonObj.data.cover_url + '" target="_blank" download="video" referrerpolicy="no-referrer"><span>下载封面</span></a>';

        // 如果video_url不为空, 则显示下载视频按钮
        if (jsonObj.data.video_url != "") {
            successHtml += '<a class="mdui-btn mdui-btn-raised" href="' + jsonObj.data.video_url + '" target="_blank" download="video" referrerpolicy="no-referrer"><span>下载视频</span></a>';
        }

        // 如果music_url不为空, 则显示下载音频按钮
        if (jsonObj.data.music_url && jsonObj.data.music_url != "") {
            successHtml += '<a class="mdui-btn mdui-btn-raised" href="' + jsonObj.data.music_url + '" target="_blank" download="audio" referrerpolicy="no-referrer"><span>下载音频</span></a>';
        }

        // 如果 jsonObj.data.images 是数组， 并且长度大于0, 则img展示图片
        if (jsonObj.data.images && jsonObj.data.images.length > 0) {
            successHtml += "<hr/>";
            successHtml += '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1em;">';
            successHtml += '<h4 style="margin: 0;">图集</h4>';

            // 将图片数据存储到全局变量,避免在HTML属性中传递大量数据
            const downloadDataId = 'download_data_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
            window[downloadDataId] = {
                images: jsonObj.data.images,
                title: jsonObj.data.title || ''
            };

            // 使用data属性存储ID,点击时读取全局变量
            successHtml += '<button class="mdui-btn mdui-color-theme-accent mdui-ripple" ' +
                          'data-download-id="' + downloadDataId + '" ' +
                          'onclick="handleDownloadClick(this)"><span>下载全部图片</span></button>';

            successHtml += '</div>';

            jsonObj.data.images.forEach(function (item, index) {
                successHtml += '<div style="display: inline-block; margin: 1em; text-align: center;">';
                successHtml += '<img src="' + item.url + '" style="width: 160px; display: block; margin-bottom: 0.5em;"/>';
                successHtml += '<div style="margin-top: 0.5em;">';
                successHtml += '<a class="mdui-btn mdui-btn-raised" href="' + item.url + '" target="_blank" download="image_' + (index + 1) + '.' + getImageExtension(item.url) + '" referrerpolicy="no-referrer" style="text-transform: none; margin-right: 0.5em;"><span>下载图片</span></a>';
                // 如果 item.live_photo_url 不为空， 显示下载按钮
                if (item.live_photo_url) {
                    successHtml += '<a class="mdui-btn mdui-btn-raised" href="' + item.live_photo_url + '" target="_blank" download="live_photo_' + (index + 1) + '.mp4" referrerpolicy="no-referrer" style="text-transform: none;"><span>下载LivePhoto</span></a>';
                }
                successHtml += '</div>';
                successHtml += '</div>';
            });
        }

        document.querySelector(".down").innerHTML = successHtml;
    } else {
        mdui.snackbar({
            message: "解析失败,视频不存在或者链接不正确:<br/>" + jsonObj.msg
        });
    }
}

// 新增清空输入框内容的函数
function clearInput() {
    document.getElementById("url").value = "";
}
