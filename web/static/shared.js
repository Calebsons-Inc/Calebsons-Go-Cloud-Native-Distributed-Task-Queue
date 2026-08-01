window.QueueUI = (() => {
  const meterDefs = [
    { key: "queued", label: "Queued" },
    { key: "running", label: "Running" },
    { key: "completed", label: "Done" },
    { key: "failed", label: "Failed" },
    { key: "dead", label: "Dead" },
    { key: "workers", label: "Workers" },
  ];

  function escapeHtml(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");
  }

  function formatTime(iso) {
    if (!iso) return "—";
    return new Date(iso).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  function taskSignature(t) {
    return [t.id, t.status, t.attempts, t.last_error || "", t.result || "", t.updated_at || ""].join("|");
  }

  function setLive(ok) {
    const badge = document.getElementById("live-badge");
    const label = document.getElementById("live-label");
    if (!badge || !label) return;
    const text = ok ? "live" : "offline";
    if (label.textContent !== text) label.textContent = text;
    badge.classList.toggle("is-online", ok);
    badge.classList.toggle("is-offline", !ok);
  }

  function mountMeters(root) {
    if (!root) return;
    root.innerHTML = meterDefs
      .map(
        (m) => `
      <article class="meter" data-key="${m.key}">
        <p class="meter-label">${m.label}</p>
        <p class="meter-value" data-meter="${m.key}">0</p>
      </article>`
      )
      .join("");
  }

  function createController(options = {}) {
    const state = {
      kind: options.kind || "",
      status: "",
      page: 1,
      limit: options.limit || 12,
      total: 0,
      tasksSig: "",
      statsSig: "",
      pageSig: "",
      timer: null,
      showKind: Boolean(options.showKind),
    };

    const listEl = document.getElementById("task-list");
    const emptyEl = document.getElementById("empty-state");
    const pageLabel = document.getElementById("page-label");
    const prevBtn = document.getElementById("prev-page");
    const nextBtn = document.getElementById("next-page");

    mountMeters(document.getElementById("meters"));

    document.querySelectorAll(".filter").forEach((btn) => {
      btn.addEventListener("click", () => {
        document.querySelectorAll(".filter").forEach((b) => {
          b.classList.remove("is-active");
          b.setAttribute("aria-selected", "false");
        });
        btn.classList.add("is-active");
        btn.setAttribute("aria-selected", "true");
        state.status = btn.dataset.status || "";
        state.page = 1;
        state.tasksSig = "";
        refresh();
      });
    });

    prevBtn?.addEventListener("click", () => {
      if (state.page > 1) {
        state.page -= 1;
        state.tasksSig = "";
        refresh();
      }
    });

    nextBtn?.addEventListener("click", () => {
      const pages = Math.max(1, Math.ceil(state.total / state.limit));
      if (state.page < pages) {
        state.page += 1;
        state.tasksSig = "";
        refresh();
      }
    });

    listEl?.addEventListener("click", async (e) => {
      const btn = e.target.closest("[data-cancel]");
      if (!btn) return;
      try {
        const res = await fetch(`/api/tasks/${encodeURIComponent(btn.dataset.cancel)}/cancel`, {
          method: "POST",
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "cancel failed");
        state.tasksSig = "";
        await refresh();
      } catch (err) {
        alert(err.message);
      }
    });

    function updateMeters(stats) {
      const sig = meterDefs.map((m) => `${m.key}:${stats[m.key] ?? 0}`).join(",");
      if (sig === state.statsSig) return;
      state.statsSig = sig;
      meterDefs.forEach((m) => {
        const el = document.querySelector(`[data-meter="${m.key}"]`);
        const next = String(stats[m.key] ?? 0);
        if (el && el.textContent !== next) el.textContent = next;
      });
    }

    function renderTasks(tasks) {
      if (!listEl || !emptyEl) return;
      const sig = tasks.map(taskSignature).join("\n");
      if (sig === state.tasksSig) return;
      state.tasksSig = sig;

      if (!tasks.length) {
        listEl.replaceChildren();
        emptyEl.hidden = false;
        return;
      }
      emptyEl.hidden = true;

      const existing = new Map(
        [...listEl.querySelectorAll(".task")].map((el) => [el.dataset.id, el])
      );
      const nextIds = new Set(tasks.map((t) => t.id));
      const frag = document.createDocumentFragment();

      for (const t of tasks) {
        const prev = existing.get(t.id);
        const canCancel = t.status === "queued" || t.status === "running" || t.status === "failed";
        let el = prev;
        if (!el) {
          el = document.createElement("li");
          el.className = "task is-new";
          el.dataset.id = t.id;
          el.addEventListener("animationend", () => el.classList.remove("is-new"), { once: true });
        } else if (prev.dataset.sig === taskSignature(t)) {
          frag.appendChild(el);
          continue;
        }

        el.dataset.sig = taskSignature(t);
        const kindChip = state.showKind
          ? `<span class="kind-chip kind-${escapeHtml(t.kind)}">${escapeHtml(t.kind)}</span>`
          : "";
        el.innerHTML = `
          <div class="task-main">
            <p class="task-name">${escapeHtml(t.name)} ${kindChip}</p>
            <p class="task-meta">${escapeHtml(t.id)} · try ${t.attempts}/${t.max_attempts} · ${formatTime(t.created_at)}</p>
            ${t.result ? `<p class="task-result">${escapeHtml(t.result)}</p>` : ""}
            ${t.last_error ? `<p class="task-error">${escapeHtml(t.last_error)}</p>` : ""}
          </div>
          <div class="task-side">
            <span class="status status-${t.status}">${t.status}</span>
            ${canCancel ? `<button type="button" class="cancel-btn" data-cancel="${t.id}">Cancel</button>` : ""}
          </div>`;
        frag.appendChild(el);
      }

      for (const [id, el] of existing) {
        if (!nextIds.has(id)) el.remove();
      }
      listEl.replaceChildren(frag);
    }

    async function refresh() {
      try {
        const params = new URLSearchParams({
          page: String(state.page),
          limit: String(state.limit),
        });
        if (state.status) params.set("status", state.status);
        if (state.kind) params.set("kind", state.kind);

        const statsURL = state.kind ? `/api/stats?kind=${encodeURIComponent(state.kind)}` : "/api/stats";
        const [statsRes, tasksRes] = await Promise.all([
          fetch(statsURL),
          fetch(`/api/tasks?${params}`),
        ]);
        if (!statsRes.ok || !tasksRes.ok) throw new Error("bad response");

        const stats = await statsRes.json();
        const list = await tasksRes.json();
        updateMeters(stats);
        state.total = list.total;
        renderTasks(list.tasks || []);

        const pages = Math.max(1, Math.ceil(state.total / state.limit));
        const pageSig = `${state.page}|${pages}|${state.total}`;
        if (pageLabel && pageSig !== state.pageSig) {
          state.pageSig = pageSig;
          pageLabel.textContent = `Page ${state.page} of ${pages} · ${state.total} tasks`;
          if (prevBtn) prevBtn.disabled = state.page <= 1;
          if (nextBtn) nextBtn.disabled = state.page >= pages;
        }
        setLive(true);
        options.onStats?.(stats);
        return stats;
      } catch {
        setLive(false);
        return null;
      }
    }

    function invalidate() {
      state.tasksSig = "";
      state.statsSig = "";
    }

    function start() {
      refresh();
      state.timer = setInterval(refresh, 1500);
    }

    return { refresh, invalidate, start, state, escapeHtml };
  }

  function renderRail(demos, activeKind) {
    const railNav = document.getElementById("rail-nav");
    if (!railNav) return;
    const overviewActive = !activeKind ? "is-active" : "";
    const links = [
      `<a class="rail-link ${overviewActive}" href="/">Overview</a>`,
      ...demos.map((d) => {
        const active = d.kind === activeKind ? "is-active" : "";
        return `<a class="rail-link ${active}" href="${escapeHtml(d.path)}">${escapeHtml(d.title)}</a>`;
      }),
    ];
    railNav.innerHTML = links.join("");
  }

  return { createController, escapeHtml, meterDefs, setLive, renderRail };
})();
