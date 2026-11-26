// MDUI inserts snackbar messages as HTML, so encode text before passing it in.
function showTextSnackbar(message, timeout = 4000) {
    const text = document.createElement("div");
    text.textContent = message;
    mdui.snackbar({ message: text.innerHTML, timeout });
}

function safeMediaURL(value) {
    if (typeof value !== "string") {
        return null;
    }
    try {
        const url = new URL(value);
        if ((url.protocol === "http:" || url.protocol === "https:") && !url.username && !url.password) {
            return url.href;
        }
    } catch {
        // The API may return empty or malformed URLs for optional media.
    }
    return null;
}

function isDouyinShareText(value) {
    const normalized = typeof value === "string" ? value.replace(/\\/g, "") : "";
    const links = normalized.match(/https?:\/\/[^\s]+/ig) || [];
    return links.some(link => {
        try {
            return ["v.douyin.com", "www.douyin.com", "www.iesdouyin.com"].includes(new URL(link).hostname.toLowerCase());
        } catch {
            return false;
        }
    });
}

function appendDownloadLink(container, url, label, filename) {
    if (!url) {
        return;
    }
    const link = document.createElement("a");
    link.className = "mdui-btn mdui-btn-raised";
    link.href = url;
    if (/^https?:\/\//i.test(url)) {
        link.target = "_blank";
        link.rel = "noopener";
        link.referrerPolicy = "no-referrer";
    }
    link.download = filename;
    const text = document.createElement("span");
    text.textContent = label;
    link.append(text);
    container.append(link);
    return link;
}

function renderParseResult(data, shareText = "") {
    const result = document.createDocumentFragment();
    const title = document.createElement("h4");
    title.textContent = typeof data.title === "string" ? data.title : "";
    result.append(title);
    appendDownloadLink(result, safeMediaURL(data.cover_url), "下载封面", "video");
    const directVideoURL = safeMediaURL(data.video_url);
    const videoDownloadURL = directVideoURL && isDouyinShareText(shareText)
        ? "/api/v1/douyin/download?url=" + encodeURIComponent(shareText)
        : directVideoURL;
    const videoFilename = ((typeof data.title === "string" && data.title.trim()) || "video")
        .replace(/[<>:"/\\|?*\u0000-\u001f]/g, "_") + ".mp4";
    appendDownloadLink(result, videoDownloadURL, "下载视频", videoFilename);
    appendDownloadLink(result, safeMediaURL(data.music_url), "下载音频", "audio");

    const gallery = Array.isArray(data.images) ? data.images.flatMap((item, index) => {
        const url = safeMediaURL(item?.url);
        const livePhotoURL = safeMediaURL(item?.live_photo_url);
        return url || livePhotoURL ? [{ url, livePhotoURL, number: index + 1 }] : [];
    }) : [];
    if (gallery.length > 0) {
        const header = document.createElement("div");
        header.style.cssText = "display: flex; justify-content: space-between; align-items: center; margin-bottom: 1em;";
        const label = document.createElement("h4");
        label.style.margin = "0";
        label.textContent = "图集";
        header.append(label);

        const images = gallery.filter(item => item.url);
        if (images.length > 0) {
            const download = document.createElement("button");
            download.type = "button";
            download.className = "mdui-btn mdui-color-theme-accent mdui-ripple";
            const downloadLabel = document.createElement("span");
            downloadLabel.textContent = "下载全部图片";
            download.append(downloadLabel);
            download.addEventListener("click", () => downloadAllImages(images, title.textContent));
            header.append(download);
        }
        result.append(document.createElement("hr"), header);

        for (const item of gallery) {
            const card = document.createElement("div");
            card.style.cssText = "display: inline-block; margin: 1em; text-align: center;";
            const links = document.createElement("div");
            links.style.marginTop = "0.5em";
            if (item.url) {
                const image = document.createElement("img");
                image.src = item.url;
                image.referrerPolicy = "no-referrer";
                image.style.cssText = "width: 160px; display: block; margin-bottom: 0.5em;";
                card.append(image);
                const imageLink = appendDownloadLink(links, item.url, "下载图片", "image_" + item.number + "." + getImageExtension(item.url));
                imageLink.style.cssText = "text-transform: none; margin-right: 0.5em;";
            }
            if (item.livePhotoURL) {
                const livePhotoLink = appendDownloadLink(links, item.livePhotoURL, "下载LivePhoto", "live_photo_" + item.number + ".mp4");
                livePhotoLink.style.textTransform = "none";
            }
            card.append(links);
            result.append(card);
        }
    }

    document.querySelector(".down").replaceChildren(result);
}

async function setValue() {
    const data = document.getElementById("url").value.trim();
    if (!data) {
        showTextSnackbar("需要解析的视频地址不能为空");
        return;
    }

    let response;
    try {
        const request = await fetch("/video/share/url/parse?url=" + encodeURIComponent(data));
        response = await request.json();
    } catch {
        showTextSnackbar("解析请求失败，请稍后重试");
        return;
    }

    if (response?.code === 200 && response.data) {
        renderParseResult(response.data, data);
        showTextSnackbar("解析成功");
    } else {
        const detail = typeof response?.msg === "string" ? response.msg : "";
        showTextSnackbar("解析失败，视频不存在或者链接不正确" + (detail ? "：" + detail : ""));
    }
}

function clearInput() {
    document.getElementById("url").value = "";
}
