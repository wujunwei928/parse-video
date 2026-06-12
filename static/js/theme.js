// 主题管理
function getSavedTheme() {
    return localStorage.getItem('selectedTheme') || 'original';
}

function toggleThemeSelector() {
    const selector = document.getElementById('themeSelector');
    const icon = document.getElementById('toggleIcon');
    selector.classList.toggle('collapsed');
    icon.textContent = selector.classList.contains('collapsed') ? '▶' : '▼';
}

function setTheme(themeName) {
    // 移除所有主题类
    document.body.className = '';

    // 添加新主题类
    if (themeName !== 'original') {
        document.body.classList.add(`theme-${themeName}`);
    }

    // 更新主题选择器状态
    document.querySelectorAll('.theme-option').forEach(option => {
        option.classList.remove('active');
    });
    document.querySelector(`[data-theme="${themeName}"]`).classList.add('active');

    // 保存主题选择
    localStorage.setItem('selectedTheme', themeName);

    // 显示主题切换提示
    showThemeNotification(themeName);
}

function showThemeNotification(themeName) {
    const themeNames = {
        'original': '经典MDUI',
        'rams': 'Dieter Rams 极简功能',
        'vignelli': 'Massimo Vignelli 现代网格',
        'kusama': 'Yayoi Kusama 波点艺术',
        'hadid': 'Zaha Hadid 流动几何',
        'starry': '梦幻星空'
    };

    const message = `已切换到 ${themeNames[themeName]} 主题`;

    // 创建临时提示元素
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        left: 50%;
        transform: translateX(-50%);
        background: rgba(0,0,0,0.8);
        color: white;
        padding: 12px 24px;
        border-radius: 25px;
        z-index: 10000;
        font-size: 14px;
        animation: slideDown 0.3s ease;
    `;
    notification.textContent = message;

    document.body.appendChild(notification);

    // 3秒后自动消失
    setTimeout(() => {
        notification.style.animation = 'slideUp 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}
// 添加动画样式
const style = document.createElement('style');
style.textContent = `
    @keyframes slideDown {
        from { opacity: 0; transform: translate(-50%, -20px); }
        to { opacity: 1; transform: translate(-50%, 0); }
    }

    @keyframes slideUp {
        from { opacity: 1; transform: translate(-50%, 0); }
        to { opacity: 0; transform: translate(-50%, -20px); }
    }
`;
document.head.appendChild(style);

// 小屏幕下自动折叠主题选择器
function collapseThemeSelectorOnSmallScreen() {
    if (window.innerWidth <= 600) {
        document.getElementById('themeSelector').classList.add('collapsed');
        document.getElementById('toggleIcon').textContent = '▶';
    }
}

// 页面加载时应用保存的主题
window.addEventListener('DOMContentLoaded', function() {
    setTheme(getSavedTheme());
    collapseThemeSelectorOnSmallScreen();
});

// 响应式处理
window.addEventListener('resize', collapseThemeSelectorOnSmallScreen);
