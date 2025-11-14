// 全局配置
axios.defaults.headers.common['Content-Type'] = 'application/json';

// Toast容器
let toastContainer = null;

// 初始化Toast容器
function initToastContainer() {
    if (!toastContainer) {
        toastContainer = document.createElement('div');
        toastContainer.className = 'toast-container';
        document.body.appendChild(toastContainer);
    }
}

// 显示Toast通知
function showToast(message, type = 'info') {
    initToastContainer();
    
    const iconMap = {
        success: 'check-circle-fill',
        error: 'x-circle-fill',
        warning: 'exclamation-triangle-fill',
        info: 'info-circle-fill'
    };
    
    const bgMap = {
        success: 'bg-success',
        error: 'bg-danger',
        warning: 'bg-warning',
        info: 'bg-info'
    };
    
    const toastEl = document.createElement('div');
    toastEl.className = 'toast custom-toast align-items-center text-white border-0 ' + bgMap[type];
    toastEl.setAttribute('role', 'alert');
    toastEl.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">
                <i class="bi bi-${iconMap[type]} me-2"></i>${message}
            </div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    
    toastContainer.appendChild(toastEl);
    
    const toast = new bootstrap.Toast(toastEl, { delay: 3000 });
    toast.show();
    
    toastEl.addEventListener('hidden.bs.toast', () => {
        toastEl.remove();
    });
}

// 快捷方法
function showSuccess(message) {
    showToast(message, 'success');
}

function showError(message) {
    showToast(message, 'error');
}

function showWarning(message) {
    showToast(message, 'warning');
}

function showInfo(message) {
    showToast(message, 'info');
}

// 加载遮罩
let loadingOverlay = null;

function showLoading(message = '加载中...') {
    if (!loadingOverlay) {
        loadingOverlay = document.createElement('div');
        loadingOverlay.className = 'loading-overlay';
        loadingOverlay.innerHTML = `
            <div class="loading-spinner">
                <div class="spinner-border text-primary mb-3" role="status">
                    <span class="visually-hidden">Loading...</span>
                </div>
                <p class="mb-0">${message}</p>
            </div>
        `;
        document.body.appendChild(loadingOverlay);
    } else {
        loadingOverlay.querySelector('p').textContent = message;
        loadingOverlay.style.display = 'flex';
    }
}

function hideLoading() {
    if (loadingOverlay) {
        loadingOverlay.style.display = 'none';
    }
}

// 模板辅助函数
function sub(a, b) {
    return a - b;
}

function add(a, b) {
    return a + b;
}

// 日期格式化
function formatDate(dateString) {
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}`;
}

// 相对时间格式化（如：2小时前）
function formatRelativeTime(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    
    if (days > 7) {
        return formatDate(dateString);
    } else if (days > 0) {
        return `${days}天前`;
    } else if (hours > 0) {
        return `${hours}小时前`;
    } else if (minutes > 0) {
        return `${minutes}分钟前`;
    } else {
        return '刚刚';
    }
}

// 页面加载后格式化所有时间
document.addEventListener('DOMContentLoaded', function() {
    // 格式化所有带有data-time属性的元素
    document.querySelectorAll('[data-time]').forEach(el => {
        const time = el.getAttribute('data-time');
        if (el.hasAttribute('data-relative')) {
            el.textContent = formatRelativeTime(time);
        } else {
            el.textContent = formatDate(time);
        }
    });
});

// 页面加载完成后的初始化
document.addEventListener('DOMContentLoaded', function() {
    // 初始化所有tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
    
    // 初始化所有popovers
    const popoverTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="popover"]'));
    popoverTriggerList.map(function (popoverTriggerEl) {
        return new bootstrap.Popover(popoverTriggerEl);
    });
});

// 处理axios错误
axios.interceptors.response.use(
    response => response,
    error => {
        if (error.response) {
            if (error.response.status === 401) {
                showError('未登录或登录已过期，请重新登录');
                setTimeout(() => {
                    window.location.href = '/admin/login';
                }, 1500);
            } else if (error.response.status === 403) {
                showError('没有权限执行此操作');
            } else if (error.response.status === 500) {
                showError('服务器内部错误，请稍后重试');
            }
        } else if (error.request) {
            showError('网络请求失败，请检查网络连接');
        }
        return Promise.reject(error);
    }
);

