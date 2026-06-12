// 处理下载按钮点击事件
function handleDownloadClick(button) {
    try {
        const downloadDataId = button.getAttribute('data-download-id');
        if (!downloadDataId) {
            mdui.snackbar({
                message: '下载数据ID不存在,请刷新页面重试'
            });
            return;
        }

        const downloadData = window[downloadDataId];
        if (!downloadData || !downloadData.images) {
            mdui.snackbar({
                message: '下载数据不存在,请刷新页面重试'
            });
            return;
        }

        // 调用批量下载函数
        downloadAllImages(downloadData.images, downloadData.title);
    } catch (error) {
        console.error('处理下载点击失败:', error);
        mdui.snackbar({
            message: '启动下载失败:' + error.message
        });
    }
}

// 批量下载图片函数
async function downloadAllImages(images, title) {
    try {
        // 检查必要的库是否加载
        const requiredLibs = [
            { name: 'JSZip', label: 'JSZip库' },
            { name: 'saveAs', label: 'FileSaver.js库' }
        ];
        for (const lib of requiredLibs) {
            if (typeof window[lib.name] === 'undefined') {
                mdui.snackbar({
                    message: lib.label + '未加载,请检查网络连接或刷新页面重试',
                    timeout: 5000
                });
                return;
            }
        }

        if (!images || images.length === 0) {
            mdui.snackbar({
                message: '没有可下载的图片'
            });
            return;
        }

        // 显示下载进度提示
        const progressMsg = document.createElement('div');
        progressMsg.innerHTML = `<div class="loading">正在准备下载 ${images.length} 张图片</div>`;
        document.body.appendChild(progressMsg);

        // 创建JSZip实例
        const zip = new JSZip();
        const imgFolder = zip.folder("images");

        // 下载所有图片
        const downloadPromises = images.map(async (item, index) => {
            try {
                // Referer/User-Agent 属浏览器禁止设置的 header，会被静默丢弃，故不传；
                // 跨域图片下载的防盗链绕过靠 <a referrerpolicy="no-referrer">，无法在 fetch 端实现。
                const response = await fetch(item.url);

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const blob = await response.blob();
                const fileName = `image_${String(index + 1).padStart(3, '0')}.${getImageExtension(item.url)}`;
                imgFolder.file(fileName, blob);

                return { success: true, index: index + 1 };
            } catch (error) {
                console.error(`下载第 ${index + 1} 张图片失败:`, error);
                return { success: false, index: index + 1, error: error.message };
            }
        });

        // 等待所有图片下载完成
        const results = await Promise.all(downloadPromises);

        // 移除进度提示
        progressMsg.remove();

        // 统计下载结果
        const successCount = results.filter(r => r.success).length;
        const failedCount = results.length - successCount;

        if (successCount === 0) {
            mdui.snackbar({
                message: '所有图片下载失败，请检查网络连接'
            });
            return;
        }

        // 生成ZIP文件
        const zipBlob = await zip.generateAsync({
            type: "blob",
            compression: "DEFLATE",
            compressionOptions: { level: 6 }
        });

        // 生成文件名
        const safeTitle = title.replace(/[^\w\s-]/g, '').trim() || 'images';
        const fileName = `${safeTitle}_${new Date().getTime()}.zip`;

        // 下载ZIP文件
        saveAs(zipBlob, fileName);

        // 显示下载结果
        let message = `成功下载 ${successCount} 张图片`;
        if (failedCount > 0) {
            message += `，${failedCount} 张失败`;
        }
        message += '，已打包为ZIP文件';

        mdui.snackbar({
            message: message,
            timeout: 5000
        });

    } catch (error) {
        console.error('批量下载失败:', error);
        mdui.snackbar({
            message: '批量下载失败：' + error.message,
            timeout: 5000
        });
    }
}

// 获取图片文件扩展名
function getImageExtension(url) {
    try {
        const urlObj = new URL(url);
        const pathname = urlObj.pathname.toLowerCase();

        if (pathname.endsWith('.jpg') || pathname.endsWith('.jpeg')) {
            return 'jpg';
        } else if (pathname.endsWith('.png')) {
            return 'png';
        } else if (pathname.endsWith('.gif')) {
            return 'gif';
        } else if (pathname.endsWith('.webp')) {
            return 'webp';
        } else {
            // 默认返回jpg
            return 'jpg';
        }
    } catch (error) {
        return 'jpg';
    }
}
