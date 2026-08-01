(() => {
  const kind = location.pathname.split("/").filter(Boolean).pop() || "";
  const titleEl = document.getElementById("demo-title");
  const blurbEl = document.getElementById("demo-blurb");
  const kickerEl = document.getElementById("demo-kicker");
  const fieldsEl = document.getElementById("dynamic-fields");
  const nameInput = document.getElementById("task-name");
  const form = document.getElementById("enqueue-form");
  const formHint = document.getElementById("form-hint");
  const flakyRow = document.getElementById("flaky-row");

  document.body.dataset.kind = kind;
  const ui = QueueUI.createController({ kind, limit: 12 });

  async function boot() {
    const [demosRes, demoRes] = await Promise.all([
      fetch("/api/demos"),
      fetch(`/api/demos/${encodeURIComponent(kind)}`),
    ]);
    if (!demoRes.ok) {
      titleEl.textContent = "Demo not found";
      return;
    }
    const demo = await demoRes.json();
    const all = (await demosRes.json()).demos || [];

    document.title = `${demo.title} — Calebsons`;
    kickerEl.textContent = `Demo · ${demo.kind}`;
    titleEl.textContent = demo.title;
    blurbEl.textContent = demo.blurb;
    nameInput.value = demo.example_name || "";
    nameInput.placeholder = demo.example_name || "job-name";

    if (demo.kind === "webhooks") flakyRow.hidden = false;

    fieldsEl.innerHTML = (demo.fields || [])
      .map(
        (f) => `
      <label>
        ${QueueUI.escapeHtml(f.label)}
        <input
          data-field="${QueueUI.escapeHtml(f.key)}"
          type="text"
          maxlength="200"
          placeholder="${QueueUI.escapeHtml(f.placeholder || "")}"
          ${f.required ? "required" : ""}
          value="${QueueUI.escapeHtml(f.placeholder || "")}"
        />
      </label>`
      )
      .join("");

    QueueUI.renderRail(all, kind);
    ui.start();
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    formHint.hidden = true;

    const payload = {};
    fieldsEl.querySelectorAll("[data-field]").forEach((input) => {
      payload[input.dataset.field] = input.value.trim();
    });
    if (kind === "webhooks" && document.getElementById("flaky-flag").checked) {
      payload.flaky = true;
    }

    const body = {
      kind,
      name: nameInput.value.trim(),
      payload: JSON.stringify(payload),
      max_attempts: Number(document.getElementById("task-attempts").value) || 3,
    };

    try {
      const res = await fetch("/api/tasks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "enqueue failed");
      ui.state.page = 1;
      ui.invalidate();
      await ui.refresh();
    } catch (err) {
      formHint.textContent = err.message;
      formHint.hidden = false;
    }
  });

  boot().catch((err) => {
    titleEl.textContent = "Failed to load demo";
    blurbEl.textContent = err.message;
    QueueUI.setLive(false);
  });
})();
