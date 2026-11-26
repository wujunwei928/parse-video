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
    const result = document.createElement("div");
    result.className = "batch-result-content";
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

    return result;
}

const maxBatchInputs = 8;
let nextBatchRowId = 2;
let batchParsing = false;
let batchDownloadItems = [];

function getBatchInputRows() {
    return Array.from(document.querySelectorAll(".batch-input-row"));
}

function updateBatchInputLabels() {
    const rows = getBatchInputRows();
    rows.forEach((row, index) => {
        const number = index + 1;
        const label = row.querySelector(".batch-input-label");
        const input = row.querySelector(".batch-url-input");
        const remove = row.querySelector(".batch-remove-button");
        label.textContent = "视频 " + number;
        label.htmlFor = input.id;
        remove.setAttribute("aria-label", "移除视频 " + number);
        remove.disabled = batchParsing || rows.length === 1;
    });
    document.getElementById("batchInputCount").textContent = rows.length + " / " + maxBatchInputs;
    document.getElementById("addBatchInputButton").disabled = batchParsing || rows.length >= maxBatchInputs;
    document.getElementById("parseBatchButton").disabled = batchParsing;
    document.getElementById("clearBatchButton").disabled = batchParsing;
    document.querySelectorAll(".batch-url-input").forEach(input => { input.disabled = batchParsing; });
}

function addBatchInput() {
    const rows = getBatchInputRows();
    if (batchParsing || rows.length >= maxBatchInputs) {
        return;
    }
    const rowId = nextBatchRowId++;
    const row = document.createElement("div");
    row.className = "batch-input-row";
    row.dataset.rowId = String(rowId);

    const label = document.createElement("label");
    label.className = "batch-input-label";
    label.htmlFor = "batch-url-" + rowId;

    const input = document.createElement("textarea");
    input.className = "mdui-textfield-input batch-url-input";
    input.id = "batch-url-" + rowId;
    input.rows = 2;
    input.placeholder = "粘贴视频分享链接或完整分享文案";

    const remove = document.createElement("button");
    remove.className = "mdui-btn batch-remove-button";
    remove.type = "button";
    remove.textContent = "移除";
    remove.addEventListener("click", () => removeBatchInput(remove));

    row.append(label, input, remove);
    document.getElementById("batchInputs").append(row);
    updateBatchInputLabels();
    input.focus();
}

function removeBatchInput(button) {
    const rows = getBatchInputRows();
    if (batchParsing || rows.length <= 1) {
        return;
    }
    button.closest(".batch-input-row").remove();
    updateBatchInputLabels();
}

function createBatchResultCard(index) {
    const card = document.createElement("section");
    card.className = "batch-result-card is-pending";
    card.dataset.index = String(index);

    const heading = document.createElement("div");
    heading.className = "batch-result-heading";
    const title = document.createElement("h3");
    title.textContent = "视频 " + (index + 1);
    const status = document.createElement("span");
    status.className = "batch-result-state";
    status.textContent = "等待解析";
    heading.append(title, status);
    card.append(heading);
    return card;
}

function updateBatchResultCard(card, result, shareText) {
    const status = card.querySelector(".batch-result-state");
    card.classList.remove("is-pending", "is-success", "is-error");
    if (result?.data) {
        card.classList.add("is-success");
        status.textContent = "解析成功";
        card.append(renderParseResult(result.data, shareText));
        return true;
    }
    card.classList.add("is-error");
    status.textContent = "解析失败";
    const message = document.createElement("p");
    message.className = "batch-result-error";
    message.textContent = typeof result?.error === "string" ? result.error : "解析失败，请稍后重试";
    card.append(message);
    return false;
}

function updateBatchProgress(completed, total, successes) {
    const status = document.getElementById("batchStatus");
    status.textContent = "已完成 " + completed + " / " + total + " 项，成功 " + successes + " 项" + (completed < total ? "，其他项目仍在解析" : "");
}

function addBatchDownloadButton(items) {
    if (items.length === 0) {
        return;
    }
    const toolbar = document.createElement("div");
    toolbar.className = "batch-download-toolbar";
    const button = document.createElement("button");
    button.type = "button";
    button.className = "mdui-btn mdui-color-theme-accent mdui-ripple batch-download-button";
    button.textContent = "下载全部成功视频（" + items.length + "）";
    button.addEventListener("click", () => downloadBatchVideos(items));
    toolbar.append(button);
    document.getElementById("batchResults").prepend(toolbar);
}

async function parseAll() {
    if (batchParsing) {
        return;
    }
    const items = getBatchInputRows()
        .map(row => row.querySelector(".batch-url-input").value.trim())
        .filter(Boolean);
    if (items.length === 0) {
        showTextSnackbar("请至少粘贴一个视频链接或分享文案");
        return;
    }
    batchDownloadItems = [];
    batchParsing = true;
    updateBatchInputLabels();

    const results = document.getElementById("batchResults");
    results.replaceChildren();
    const cards = items.map((item, index) => {
        const card = createBatchResultCard(index);
        const source = document.createElement("p");
        source.className = "batch-result-source";
        source.textContent = item;
        card.append(source);
        results.append(card);
        return card;
    });
    document.getElementById("batchStatus").textContent = "正在并发解析 " + items.length + " 项…";

    let completed = 0;
    let successes = 0;
    try {
        const response = await fetch("/api/v1/parse/batch", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ items })
        });
        if (!response.ok || !response.body) {
            const error = await response.json().catch(() => null);
            throw new Error(error?.error?.message || "批量解析请求失败，请稍后重试");
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let pending = "";
        while (true) {
            const { value, done } = await reader.read();
            pending += decoder.decode(value || new Uint8Array(), { stream: !done });
            const lines = pending.split("\n");
            pending = lines.pop() || "";
            for (const line of lines) {
                if (!line.trim()) {
                    continue;
                }
                const result = JSON.parse(line);
                const card = cards[result.index];
                if (!card || card.dataset.complete === "true") {
                    continue;
                }
                const success = updateBatchResultCard(card, result, items[result.index]);
                card.dataset.complete = "true";
                completed++;
                if (success) {
                    successes++;
                    batchDownloadItems.push(items[result.index]);
                }
                updateBatchProgress(completed, items.length, successes);
            }
            if (done) {
                break;
            }
        }
        if (pending.trim()) {
            const result = JSON.parse(pending);
            const card = cards[result.index];
            if (card && card.dataset.complete !== "true") {
                const success = updateBatchResultCard(card, result, items[result.index]);
                card.dataset.complete = "true";
                completed++;
                if (success) {
                    successes++;
                    batchDownloadItems.push(items[result.index]);
                }
            }
        }
        cards.forEach((card, index) => {
            if (card.dataset.complete !== "true") {
                updateBatchResultCard(card, { error: "解析服务未返回结果，请重试" }, items[index]);
                card.dataset.complete = "true";
                completed++;
            }
        });
        updateBatchProgress(completed, items.length, successes);
        addBatchDownloadButton(batchDownloadItems);
        showTextSnackbar("批量解析完成：成功 " + successes + " 项");
    } catch (error) {
        const message = error instanceof SyntaxError ? "解析服务返回了无效结果" : error.message;
        cards.forEach(card => {
            if (card.dataset.complete !== "true") {
                updateBatchResultCard(card, { error: message }, "");
                card.dataset.complete = "true";
            }
        });
        addBatchDownloadButton(batchDownloadItems);
        document.getElementById("batchStatus").textContent = message;
        showTextSnackbar(message);
    } finally {
        batchParsing = false;
        updateBatchInputLabels();
    }
}

function downloadBatchVideos(items) {
    if (!Array.isArray(items) || items.length === 0) {
        showTextSnackbar("没有可下载的视频");
        return;
    }
    const form = document.createElement("form");
    form.method = "POST";
    form.action = "/api/v1/download/batch";
    form.target = "_blank";
    form.hidden = true;
    for (const item of items) {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = "items";
        input.value = item;
        form.append(input);
    }
    document.body.append(form);
    form.submit();
    form.remove();
}

function clearBatchInputs() {
    if (batchParsing) {
        return;
    }
    getBatchInputRows().forEach((row, index) => {
        if (index > 0) {
            row.remove();
        } else {
            row.querySelector(".batch-url-input").value = "";
        }
    });
    document.getElementById("batchResults").replaceChildren();
    document.getElementById("batchStatus").textContent = "";
    batchDownloadItems = [];
    updateBatchInputLabels();
}

function initBatchInputs() {
    document.querySelectorAll(".batch-remove-button").forEach(button => {
        button.addEventListener("click", () => removeBatchInput(button));
    });
    updateBatchInputLabels();
}

initBatchInputs();
