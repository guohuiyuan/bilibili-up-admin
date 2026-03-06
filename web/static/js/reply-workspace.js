class ReplyWorkspaceModal {
    constructor(options) {
        this.channel = options.channel;
        this.modalId = options.modalId;
        this.apiPrefix = options.apiPrefix || "";
        this.onSent = options.onSent || (() => {});
        this.state = {
            targetId: 0,
            conversationId: 0,
            target: null,
            templates: [],
            examples: [],
            logs: [],
            selectedTemplateContent: "",
            sourceType: "manual",
            initialInstruction: ""
        };
    }

    async open(targetId, opts = {}) {
        this.state.targetId = targetId;
        this.state.conversationId = Number(opts.conversationId || 0);
        this.state.sourceType = opts.autoGenerate ? "ai" : "manual";
        this.state.initialInstruction = opts.instruction || "";
        Modal.show(this.modalId);
        this.renderLoading();
        try {
            await this.reloadWorkspaceData();
            this.render();
            if (opts.autoGenerate && !this.getEditorContent().trim()) {
                await this.generateReply();
            }
        } catch (error) {
            this.close();
            showToast(error.message, "error");
        }
    }

    async reloadWorkspaceData() {
        const query = new URLSearchParams({
            channel: this.channel,
            target_id: String(this.state.targetId)
        });
        if (this.state.conversationId > 0) {
            query.set("conversation_id", String(this.state.conversationId));
        }
        const data = await api(`${this.apiPrefix}/api/reply-workspace?${query.toString()}`);
        this.state.target = data.target || null;
        this.state.templates = data.templates || [];
        this.state.examples = data.examples || [];
        this.state.logs = data.logs || [];
    }

    close() {
        Modal.hide(this.modalId);
    }

    renderLoading() {
        this.text("reply-workspace-target", "加载中...");
        this.text("reply-workspace-source", "");
        this.setHTML("reply-workspace-logs", '<div class="text-sm text-gray-400">正在加载 AI 调用记录...</div>');
        this.editor().value = "";
    }

    render() {
        const target = this.state.target || {};
        this.text("reply-workspace-target", `${target.author_name || "用户"} / ${target.title || this.channel}`);
        this.text("reply-workspace-source", this.buildSourceText(target));
        this.editor().value = target.reply_content || "";
        this.input("reply-workspace-instruction").value = this.state.initialInstruction || "";
        this.state.selectedTemplateContent = "";
        this.renderLogs();
    }

    buildSourceText(target) {
        const parts = [];
        if (target.video_title) {
            parts.push(`视频标题：${target.video_title}`);
        }
        if (target.video_bvid) {
            parts.push(`BVID：${target.video_bvid}`);
        }
        if (target.video_desc) {
            parts.push(`视频简介：\n${target.video_desc}`);
        }
        if (target.input_content) {
            parts.push(`当前消息：\n${target.input_content}`);
        }
        return parts.join("\n\n");
    }

    renderTemplates() {
        const container = this.node("reply-workspace-templates");
        if (!container) {
            return;
        }
        if (!this.state.templates.length) {
            container.innerHTML = '<div class="text-sm text-gray-400">还没有可用模板。</div>';
            return;
        }
        container.innerHTML = this.state.templates.map(item => `
            <div class="rounded-2xl border border-gray-200 bg-white p-3">
                <div class="flex items-start justify-between gap-3">
                    <div>
                        <div class="text-sm font-medium text-gray-800">${this.escape(item.title)}</div>
                        <div class="mt-1 text-xs text-gray-400">${this.escape(item.scene || "通用模板")}</div>
                    </div>
                    <button type="button" onclick="window.__replyWorkspaceInstances['${this.channel}'].applyTemplate(${item.id})" class="rounded-full bg-pink-50 px-3 py-1 text-xs text-pink-600 hover:bg-pink-100 transition">套用</button>
                </div>
                <div class="mt-2 text-sm text-gray-600 whitespace-pre-wrap break-words">${this.escape(item.content)}</div>
            </div>
        `).join("");
    }

    renderExamples() {
        const container = this.node("reply-workspace-examples");
        if (!container) {
            return;
        }
        if (!this.state.examples.length) {
            container.innerHTML = '<div class="text-sm text-gray-400">发送时勾选保存示例后，这里会逐步积累高质量回复。</div>';
            return;
        }
        container.innerHTML = this.state.examples.map(item => `
            <div class="rounded-2xl border border-amber-100 bg-amber-50/60 p-3">
                <div class="text-sm font-medium text-gray-800">${this.escape(item.title)}</div>
                <div class="mt-2 text-xs text-gray-500">用户消息：${this.escape(item.user_input)}</div>
                <div class="mt-2 text-sm text-gray-700 whitespace-pre-wrap break-words">回复示例：${this.escape(item.reply_content)}</div>
            </div>
        `).join("");
    }

    renderLogs() {
        const container = this.node("reply-workspace-logs");
        if (!this.state.logs.length) {
            container.innerHTML = '<div class="rounded-2xl border border-dashed border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">当前对象还没有 AI 调用记录。</div>';
            return;
        }
        container.innerHTML = this.state.logs.map(item => {
            const logType = item.log_type || "reply";
            const typeLabel = logType === "summary" ? "摘要压缩" : "回复生成";
            const tokenLabel = `输入 ${item.prompt_tokens || 0} / 输出 ${item.output_tokens || 0} / 合计 ${item.total_tokens || 0}`;
            const durationLabel = `${item.duration || 0} ms`;
            const createdAt = this.formatDate(item.created_at);
            const statusClass = item.success ? "bg-emerald-50 text-emerald-700 border-emerald-200" : "bg-rose-50 text-rose-700 border-rose-200";
            return `
                <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="inline-flex rounded-full border px-2.5 py-1 text-xs ${statusClass}">${item.success ? "成功" : "失败"}</span>
                        <span class="inline-flex rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs text-slate-700">${this.escape(typeLabel)}</span>
                        <span class="text-xs text-slate-500">${this.escape(item.provider || "模型")} / ${this.escape(item.model || "-")}</span>
                        <span class="text-xs text-slate-400">${this.escape(createdAt)}</span>
                    </div>
                    <div class="mt-3 grid grid-cols-1 gap-2 text-xs text-slate-500">
                        <div>Token：${this.escape(tokenLabel)}</div>
                        <div>耗时：${this.escape(durationLabel)}</div>
                    </div>
                    <details class="mt-3 rounded-xl border border-slate-200 bg-slate-50 p-3">
                        <summary class="cursor-pointer text-sm font-medium text-slate-700">查看提示词与回复</summary>
                        <div class="mt-3 space-y-3 text-xs text-slate-600">
                            <div>
                                <div class="font-semibold text-slate-700">系统提示词</div>
                                <pre class="mt-1 whitespace-pre-wrap break-words rounded-xl bg-white p-3 border border-slate-200">${this.escape(item.system_prompt || "")}</pre>
                            </div>
                            <div>
                                <div class="font-semibold text-slate-700">请求消息</div>
                                <pre class="mt-1 whitespace-pre-wrap break-words rounded-xl bg-white p-3 border border-slate-200">${this.escape(this.prettyJSON(item.request_messages || ""))}</pre>
                            </div>
                            <div>
                                <div class="font-semibold text-slate-700">模型回复</div>
                                <pre class="mt-1 whitespace-pre-wrap break-words rounded-xl bg-white p-3 border border-slate-200">${this.escape(item.output_content || "")}</pre>
                            </div>
                        </div>
                    </details>
                </div>
            `;
        }).join("");
    }

    applyTemplate(templateId) {
        const template = this.state.templates.find(item => item.id === templateId);
        if (!template) return;
        this.state.selectedTemplateContent = template.content || "";
        const editor = this.editor();
        const current = editor.value.trim();
        editor.value = current ? `${current}\n${template.content}` : template.content;
        this.state.sourceType = "manual";
    }

    async generateReply() {
        const instruction = this.input("reply-workspace-instruction").value.trim();
        this.setBusy(true, "正在生成回复...");
        try {
            const data = await api(`${this.apiPrefix}/api/reply-workspace/draft/generate`, {
                method: "POST",
                body: {
                    channel: this.channel,
                    target_id: this.state.targetId,
                    conversation_id: this.state.conversationId,
                    template_content: this.state.selectedTemplateContent || "",
                    extra_instruction: instruction
                }
            });
            const content = (data.reply && data.reply.content) || (data.draft && data.draft.content) || data.content || "";
            this.editor().value = content;
            this.state.sourceType = "ai";
            await this.reloadWorkspaceData();
            this.renderLogs();
            showToast("已生成回复内容", "success");
        } finally {
            this.setBusy(false);
        }
    }

    async sendReply() {
        const content = this.getEditorContent();
        if (!content.trim()) {
            showToast("回复内容不能为空", "error");
            return;
        }
        this.setBusy(true, "正在发送回复...");
        try {
            await api(`${this.apiPrefix}/api/reply-workspace/draft/send`, {
                method: "POST",
                body: {
                    channel: this.channel,
                    target_id: this.state.targetId,
                    conversation_id: this.state.conversationId,
                    content,
                    source_type: this.state.sourceType || "manual",
                    template_content: this.state.selectedTemplateContent || "",
                    extra_instruction: this.input("reply-workspace-instruction").value.trim(),
                    save_as_example: false,
                    example_title: "",
                    example_notes: ""
                }
            });
            showToast("回复已发送", "success");
            this.close();
            this.onSent();
        } finally {
            this.setBusy(false);
        }
    }

    async saveAsTemplate() {
        const content = this.getEditorContent();
        if (!content.trim()) {
            showToast("模板内容不能为空", "error");
            return;
        }
        const title = window.prompt("请输入模板名称");
        if (!title) return;
        await api(`${this.apiPrefix}/api/reply-workspace/templates`, {
            method: "POST",
            body: {
                channel: this.channel,
                title,
                content,
                scene: this.input("reply-workspace-instruction").value.trim()
            }
        });
        const res = await api(`${this.apiPrefix}/api/reply-workspace/templates?channel=${encodeURIComponent(this.channel)}`);
        this.state.templates = res.templates || [];
        this.renderTemplates();
        showToast("模板已保存", "success");
    }

    setBusy(disabled, message = "") {
        [
            "reply-workspace-generate",
			"reply-workspace-send"
        ].forEach(id => {
            const node = this.node(id);
            if (!node) return;
            node.disabled = disabled;
            node.classList.toggle("opacity-60", disabled);
        });
        if (disabled && message) {
            showToast(message, "info");
        }
    }

    getEditorContent() {
        return this.editor().value || "";
    }

    prettyJSON(value) {
        try {
            return JSON.stringify(JSON.parse(value), null, 2);
        } catch (_) {
            return value;
        }
    }

    formatDate(value) {
        if (!value) return "";
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) return String(value);
        return date.toLocaleString("zh-CN");
    }

    editor() {
        return this.node("reply-workspace-draft");
    }

    input(id) {
        return this.node(id);
    }

    checkbox(id) {
        return this.node(id);
    }

        setHTML(id, value) {
                const node = this.node(id);
                if (node) {
                        node.innerHTML = value;
                }
        }

    text(id, value) {
		const node = this.node(id);
		if (node) {
			node.textContent = value;
		}
    }

    node(id) {
        return document.getElementById(id);
    }

    escape(value) {
        return String(value ?? "")
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");
    }
}

window.__replyWorkspaceInstances = window.__replyWorkspaceInstances || {};
window.ReplyWorkspaceModal = ReplyWorkspaceModal;
