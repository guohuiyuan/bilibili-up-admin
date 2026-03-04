async function api(url, options = {}) {
    const config = {
        headers: {
            "Content-Type": "application/json",
            ...(options.headers || {})
        },
        ...options
    };

    if (config.body && typeof config.body === "object") {
        config.body = JSON.stringify(config.body);
    }

    const response = await fetch(url, config);
    const contentType = response.headers.get("content-type") || "";
    const payload = contentType.includes("application/json") ? await response.json() : await response.text();

    if (!response.ok) {
        const message = typeof payload === "string" ? payload : payload.error || payload.message || `HTTP ${response.status}`;
        throw new Error(message);
    }

    return payload;
}

function showToast(message, type = "info") {
    const toast = document.createElement("div");
    const color = type === "success"
        ? "bg-green-600"
        : type === "error"
            ? "bg-red-600"
            : type === "warning"
                ? "bg-amber-500"
                : "bg-slate-800";

    toast.className = `fixed top-4 right-4 z-50 rounded-xl px-5 py-3 text-sm text-white shadow-lg ${color}`;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 3000);
}

function formatTime(timestamp) {
    if (!timestamp) return "";
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleString();
}

function formatNumber(num) {
    const value = Number(num || 0);
    if (value >= 100000000) return `${(value / 100000000).toFixed(1)}e8`;
    if (value >= 10000) return `${(value / 10000).toFixed(1)}w`;
    return String(value);
}

function formatDuration(seconds) {
    const total = Number(seconds || 0);
    const mins = Math.floor(total / 60);
    const secs = total % 60;
    return `${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
}

async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast("Copied", "success");
        return true;
    } catch {
        showToast("Copy failed", "error");
        return false;
    }
}

function debounce(fn, wait) {
    let timer = null;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => fn(...args), wait);
    };
}

function throttle(fn, wait) {
    let inFlight = false;
    return (...args) => {
        if (inFlight) return;
        inFlight = true;
        fn(...args);
        setTimeout(() => {
            inFlight = false;
        }, wait);
    };
}

const Modal = {
    show(id) {
        const node = document.getElementById(id);
        if (!node) return;
        node.classList.remove("hidden");
        document.body.style.overflow = "hidden";
    },
    hide(id) {
        const node = document.getElementById(id);
        if (!node) return;
        node.classList.add("hidden");
        document.body.style.overflow = "";
    }
};

const Storage = {
    get(key, fallback = null) {
        try {
            const raw = localStorage.getItem(key);
            return raw ? JSON.parse(raw) : fallback;
        } catch {
            return fallback;
        }
    },
    set(key, value) {
        localStorage.setItem(key, JSON.stringify(value));
    },
    remove(key) {
        localStorage.removeItem(key);
    }
};

const PageState = {
    get(key, fallback = "") {
        return new URL(window.location.href).searchParams.get(key) || fallback;
    },
    set(key, value) {
        const url = new URL(window.location.href);
        if (value === "" || value == null) {
            url.searchParams.delete(key);
        } else {
            url.searchParams.set(key, value);
        }
        window.history.replaceState({}, "", url);
    }
};

document.addEventListener("DOMContentLoaded", () => {
    const path = window.location.pathname;
    document.querySelectorAll(".sidebar-link").forEach(link => {
        if (link.getAttribute("href") === path) {
            link.classList.add("active");
        }
    });
});

window.api = api;
window.showToast = showToast;
window.formatTime = formatTime;
window.formatNumber = formatNumber;
window.formatDuration = formatDuration;
window.copyToClipboard = copyToClipboard;
window.debounce = debounce;
window.throttle = throttle;
window.Modal = Modal;
window.Storage = Storage;
window.PageState = PageState;
