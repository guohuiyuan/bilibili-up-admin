class ReplyWorkspaceModal {
    constructor(options) {
        this.channel = options.channel;
        this.modalId = options.modalId;
        this.apiPrefix = options.apiPrefix || "";
        this.onSent = options.onSent || (() => {});
        this.state = {
            targetId: 0,
            target: null,
            draft: null,
            templates: [],
            examples: [],
            logs: [],
            selectedTemplateContent: ""
        };
    }

    async open(targetId, opts = {}) {
        this.state.targetId = targetId;
        Modal.show(this.modalId);
        this.renderLoading();
        try {
            await this.reloadWorkspaceData();
            this.render();
            if (opts.autoGenerate && !this.getDraftContent().trim()) {
                await this.generateDraft();
            }
        } catch (error) {
            this.close();
            showToast(error.message, "error");
        }
    }

    async reloadWorkspaceData() {
        const data = await api(`${this.apiPrefix}/api/reply-workspace?channel=${encodeURIComponent(this.channel)}&target_id=${this.state.targetId}`);
        this.state.target = data.target || null;
        this.state.draft = data.draft || null;
        this.state.templates = data.templates || [];
        this.state.examples = data.examples || [];
        this.state.logs = data.logs || [];
    }

    close() {
        Modal.hide(this.modalId);
    }

    renderLoading() {
        this.text("reply-workspace-target", "Loading...");
        this.text("reply-workspace-source", "");
        this.node("reply-workspace-templates").innerHTML = '<div class="text-sm text-gray-400">Loading templates...</div>';
        this.node("reply-workspace-examples").innerHTML = '<div class="text-sm text-gray-400">Loading examples...</div>';
        this.node("reply-workspace-logs").innerHTML = '<div class="text-sm text-gray-400">Loading LLM logs...</div>';
        this.textarea().value = "";
    }

    render() {
        const target = this.state.target || {};
        this.text("reply-workspace-target", `${target.author_name || "User"} / ${target.title || this.channel}`);
        this.text("reply-workspace-source", this.buildSourceText(target));
        this.textarea().value = (this.state.draft && this.state.draft.content) || target.reply_content || "";
        this.input("reply-workspace-instruction").value = (this.state.draft && this.state.draft.extra_instruction) || "";
        this.state.selectedTemplateContent = (this.state.draft && this.state.draft.template_snapshot) || "";
        this.input("reply-workspace-example-title").value = `${target.author_name || "User"}-${this.channel === "message" ? "DM" : "Comment"} Example`;
        this.renderTemplates();
        this.renderExamples();
        this.renderLogs();
    }

    buildSourceText(target) {
        const parts = [];
        if (target.video_title) {
            parts.push(`Video: ${target.video_title}`);
        }
        if (target.video_bvid) {
            parts.push(`BVID: ${target.video_bvid}`);
        }
        if (target.video_desc) {
            parts.push(`Video description:\n${target.video_desc}`);
        }
        if (target.input_content) {
            parts.push(`Current message:\n${target.input_content}`);
        }
        return parts.join("\n\n");
    }

    renderTemplates() {
        const container = this.node("reply-workspace-templates");
        if (!this.state.templates.length) {
            container.innerHTML = '<div class="text-sm text-gray-400">No templates yet.</div>';
            return;
        }
        container.innerHTML = this.state.templates.map(item => `
            <div class="rounded-2xl border border-gray-200 bg-white p-3">
                <div class="flex items-start justify-between gap-3">
                    <div>
                        <div class="text-sm font-medium text-gray-800">${this.escape(item.title)}</div>
                        <div class="mt-1 text-xs text-gray-400">${this.escape(item.scene || "Template")}</div>
                    </div>
                    <button type="button" onclick="window.__replyWorkspaceInstances['${this.channel}'].applyTemplate(${item.id})" class="rounded-full bg-pink-50 px-3 py-1 text-xs text-pink-600 hover:bg-pink-100 transition">Use</button>
                </div>
                <div class="mt-2 text-sm text-gray-600 whitespace-pre-wrap break-words">${this.escape(item.content)}</div>
            </div>
        `).join("");
    }

    renderExamples() {
        const container = this.node("reply-workspace-examples");
        if (!this.state.examples.length) {
            container.innerHTML = '<div class="text-sm text-gray-400">Send a reply with example saving enabled to accumulate examples.</div>';
            return;
        }
        container.innerHTML = this.state.examples.map(item => `
            <div class="rounded-2xl border border-amber-100 bg-amber-50/60 p-3">
                <div class="text-sm font-medium text-gray-800">${this.escape(item.title)}</div>
                <div class="mt-2 text-xs text-gray-500">User: ${this.escape(item.user_input)}</div>
                <div class="mt-2 text-sm text-gray-700 whitespace-pre-wrap break-words">Reply: ${this.escape(item.reply_content)}</div>
            </div>
        `).join("");
    }

    renderLogs() {
        const container = this.node("reply-workspace-logs");
        if (!this.state.logs.length) {
            container.innerHTML = '<div class="rounded-2xl border border-dashed border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">No LLM calls in this conversation yet.</div>';
            return;
        }
        container.innerHTML = this.state.logs.map(item => {
            const logType = item.log_type || "draft";
            const typeLabel = logType === "summary" ? "Summary compression" : "Draft generation";
            const tokenLabel = `prompt ${item.prompt_tokens || 0} / output ${item.output_tokens || 0} / total ${item.total_tokens || 0}`;
            const durationLabel = `${item.duration || 0} ms`;
            const createdAt = this.formatDate(item.created_at);
            const statusClass = item.success ? "bg-emerald-50 text-emerald-700 border-emerald-200" : "bg-rose-50 text-rose-700 border-rose-200";
            return `
                <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
                    <div class="flex flex-wrap items-center gap-2">
                        <span class="inline-flex rounded-full border px-2.5 py-1 text-xs ${statusClass}">${item.success ? "Success" : "Failed"}</span>
                        <span class="inline-flex rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs text-slate-700">${this.escape(typeLabel)}</span>
                        <span class="text-xs text-slate-500">${this.escape(item.provider || "LLM")} / ${this.escape(item.model || "-")}</span>
                        <span class="text-xs text-slate-400">${this.escape(createdAt)}</span>
                    </div>
                    <div class="mt-3 grid grid-cols-1 gap-2 text-xs text-slate-500">
                        <div>Tokens: ${this.escape(tokenLabel)}</div>
                        <div>Latency: ${this.escape(durationLabel)}</div>
                    </div>
                    <details class="mt-3 rounded-xl border border-slate-200 bg-slate-50 p-3">
                        <summary class="cursor-pointer text-sm font-medium text-slate-700">Prompt and response</summary>
                        <div class="mt-3 space-y-3 text-xs text-slate-600">
                            <div>
                                <div class="font-semibold text-slate-700">System prompt</div>
                                <pre class="mt-1 whitespace-pre-wrap break-words rounded-xl bg-white p-3 border border-slate-200">${this.escape(item.system_prompt || "")}</pre>
                            </div>
                            <div>
                                <div class="font-semibold text-slate-700">Request messages</div>
                                <pre class="mt-1 whitespace-pre-wrap break-words rounded-xl bg-white p-3 border border-slate-200">${this.escape(this.prettyJSON(item.request_messages || ""))}</pre>
                            </div>
                            <div>
                                <div class="font-semibold text-slate-700">Response</div>
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
        const textarea = this.textarea();
        const current = textarea.value.trim();
        textarea.value = current ? `${current}\n${template.content}` : template.content;
    }

    async generateDraft() {
        const instruction = this.input("reply-workspace-instruction").value.trim();
        this.setBusy(true, "Generating draft...");
        try {
            const data = await api(`${this.apiPrefix}/api/reply-workspace/draft/generate`, {
                method: "POST",
                body: {
                    channel: this.channel,
                    target_id: this.state.targetId,
                    template_content: this.state.selectedTemplateContent || "",
                    extra_instruction: instruction
                }
            });
            this.state.draft = data.draft || null;
            await this.reloadWorkspaceData();
            this.render();
            showToast("Draft generated", "success");
        } finally {
            this.setBusy(false);
        }
    }

    async saveDraft() {
        this.setBusy(true, "Saving draft...");
        try {
            const data = await api(`${this.apiPrefix}/api/reply-workspace/draft/save`, {
                method: "POST",
                body: {
                    channel: this.channel,
                    target_id: this.state.targetId,
                    content: this.getDraftContent(),
                    source_type: this.state.draft?.source_type || "manual",
                    template_content: this.state.selectedTemplateContent || "",
                    extra_instruction: this.input("reply-workspace-instruction").value.trim()
                }
            });
            this.state.draft = data.draft || null;
            showToast("Draft saved", "success");
        } finally {
            this.setBusy(false);
        }
    }

    async sendDraft() {
        const content = this.getDraftContent();
        if (!content.trim()) {
            showToast("Draft content is empty", "error");
            return;
        }
        this.setBusy(true, "Sending reply...");
        try {
            await api(`${this.apiPrefix}/api/reply-workspace/draft/send`, {
                method: "POST",
                body: {
                    channel: this.channel,
                    target_id: this.state.targetId,
                    content,
                    source_type: this.state.draft?.source_type || "manual",
                    template_content: this.state.selectedTemplateContent || "",
                    extra_instruction: this.input("reply-workspace-instruction").value.trim(),
                    save_as_example: this.checkbox("reply-workspace-save-example").checked,
                    example_title: this.input("reply-workspace-example-title").value.trim(),
                    example_notes: this.input("reply-workspace-example-notes").value.trim()
                }
            });
            showToast("Reply sent", "success");
            this.close();
            this.onSent();
        } finally {
            this.setBusy(false);
        }
    }

    async saveAsTemplate() {
        const content = this.getDraftContent();
        if (!content.trim()) {
            showToast("Template content is empty", "error");
            return;
        }
        const title = window.prompt("Template title");
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
        showToast("Template saved", "success");
    }

    setBusy(disabled, message = "") {
        [
            "reply-workspace-generate",
            "reply-workspace-save",
            "reply-workspace-send",
            "reply-workspace-save-template"
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

    getDraftContent() {
        return this.textarea().value || "";
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

    textarea() {
        return this.node("reply-workspace-draft");
    }

    input(id) {
        return this.node(id);
    }

    checkbox(id) {
        return this.node(id);
    }

    text(id, value) {
        this.node(id).textContent = value;
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
