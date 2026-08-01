(() => {
  const grid = document.getElementById("demo-grid");
  const ui = QueueUI.createController({ showKind: true, limit: 10 });

  async function boot() {
    const res = await fetch("/api/demos");
    const data = await res.json();
    const demos = data.demos || [];

    QueueUI.renderRail(demos, "");

    grid.innerHTML = demos
      .map(
        (d, i) => `
      <a class="demo-card kind-${QueueUI.escapeHtml(d.kind)}" href="${QueueUI.escapeHtml(d.path)}" style="--delay:${0.08 + i * 0.05}s">
        <span class="demo-card-kind">${QueueUI.escapeHtml(d.kind)}</span>
        <h3>${QueueUI.escapeHtml(d.title)}</h3>
        <p>${QueueUI.escapeHtml(d.blurb)}</p>
        <span class="demo-card-cta">Open demo →</span>
      </a>`
      )
      .join("");

    ui.start();
  }

  boot().catch(() => {
    QueueUI.renderRail([], "");
    grid.innerHTML = `<p class="empty">Could not load demos.</p>`;
    QueueUI.setLive(false);
  });
})();
