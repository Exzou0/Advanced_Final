// ===== Config =====
const API_BASE = ""; // same origin (Go serves frontend + API together)

// ===== Helpers =====
const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => Array.from(document.querySelectorAll(sel));

function setActiveNav() {
    const path = location.pathname.split("/").pop() || "index.html";
    $$(".navlinks a.pill").forEach(a => {
        const href = a.getAttribute("href");
        a.classList.toggle("active", href === path);
    });
}

function toast(title, message) {
    let wrap = $(".toastWrap");
    if (!wrap) {
        wrap = document.createElement("div");
        wrap.className = "toastWrap";
        document.body.appendChild(wrap);
    }
    const el = document.createElement("div");
    el.className = "toast";
    el.innerHTML = `<p class="t">${escapeHtml(title)}</p><p class="m">${escapeHtml(message)}</p>`;
    wrap.appendChild(el);
    setTimeout(() => el.remove(), 3200);
}

function escapeHtml(str) {
    return String(str)
        .replaceAll("&","&amp;")
        .replaceAll("<","&lt;")
        .replaceAll(">","&gt;")
        .replaceAll('"',"&quot;")
        .replaceAll("'","&#039;");
}

async function api(path, { method="GET", body, headers } = {}) {
    const opts = { method, headers: { ...(headers || {}) } };
    if (body !== undefined) {
        opts.headers["Content-Type"] = "application/json";
        opts.body = JSON.stringify(body);
    }

    const res = await fetch(API_BASE + path, opts);
    const ct = res.headers.get("content-type") || "";
    let data = null;
    if (ct.includes("application/json")) {
        data = await res.json().catch(() => null);
    } else {
        data = await res.text().catch(() => null);
    }

    if (!res.ok) {
        const msg =
            (data && data.error) ? data.error :
                (typeof data === "string" && data.trim() ? data : `HTTP ${res.status}`);
        throw new Error(msg);
    }
    return data;
}

function openModal(title, contentHtml) {
    const modal = $("#modal");
    $("#modalTitle").textContent = title;
    $("#modalBody").innerHTML = contentHtml;
    modal.classList.add("open");
}
function closeModal() { $("#modal")?.classList.remove("open"); }

// ===== Page: Movies =====
async function loadMovies() {
    const tbody = $("#moviesBody");
    if (!tbody) return;

    tbody.innerHTML = `<tr><td colspan="6">Loading...</td></tr>`;
    try {
        const items = await api("/movies");
        if (!items || items.length === 0) {
            tbody.innerHTML = `<tr><td colspan="6">No movies yet. Create one 🙂</td></tr>`;
            return;
        }

        const q = ($("#search")?.value || "").toLowerCase().trim();
        const filtered = items.filter(m => {
            const text = `${m.id} ${m.title} ${m.genre} ${m.duration} ${m.price}`.toLowerCase();
            return !q || text.includes(q);
        });

        tbody.innerHTML = filtered.map(m => `
      <tr>
        <td>${m.id}</td>
        <td>${escapeHtml(m.title ?? "")}</td>
        <td><span class="badge warn">${escapeHtml(m.genre ?? "-")}</span></td>
        <td>${m.duration ?? "-"}</td>
        <td>${m.price ?? 0}</td>
        <td>
          <div class="row">
            <button class="btn" data-action="edit" data-id="${m.id}">Edit</button>
            <button class="btn danger" data-action="del" data-id="${m.id}">Delete</button>
          </div>
        </td>
      </tr>
    `).join("");
    } catch (e) {
        tbody.innerHTML = `<tr><td colspan="6">Error: ${escapeHtml(e.message)}</td></tr>`;
        toast("Movies", e.message);
    }
}

function moviesWire() {
    if (!$("#moviesBody")) return;

    $("#refreshMovies")?.addEventListener("click", loadMovies);
    $("#search")?.addEventListener("input", loadMovies);

    $("#createMovie")?.addEventListener("click", () => {
        openModal("Create Movie", `
      <div class="grid">
        <div class="col-12 field">
          <label>Title</label>
          <input class="input" id="mTitle" placeholder="e.g. Interstellar" />
        </div>
        <div class="col-6 field">
          <label>Genre</label>
          <input class="input" id="mGenre" placeholder="Sci-Fi" />
        </div>
        <div class="col-3 field">
          <label>Duration (min)</label>
          <input class="input" id="mDuration" type="number" min="1" value="120"/>
        </div>
        <div class="col-3 field">
          <label>Price (KZT)</label>
          <input class="input" id="mPrice" type="number" min="0" value="2500"/>
        </div>
      </div>
      <div class="hr"></div>
      <div class="row">
        <button class="btn primary" id="mSave">Save</button>
        <div class="spacer"></div>
        <button class="btn" id="mCancel">Cancel</button>
      </div>
    `);

        $("#mCancel").onclick = closeModal;
        $("#mSave").onclick = async () => {
            try {
                const body = {
                    title: $("#mTitle").value,
                    genre: $("#mGenre").value,
                    duration: Number($("#mDuration").value),
                    price: Number($("#mPrice").value),
                };
                const created = await api("/movies", { method:"POST", body });
                toast("Created", `Movie #${created.id} saved`);
                closeModal();
                loadMovies();
            } catch(e) {
                toast("Create failed", e.message);
            }
        };
    });

    $("#moviesBody").addEventListener("click", async (ev) => {
        const btn = ev.target.closest("button");
        if (!btn) return;
        const action = btn.dataset.action;
        const id = Number(btn.dataset.id);

        if (action === "del") {
            openModal("Delete Movie", `
        <p class="small">Are you sure you want to delete movie <kbd>#${id}</kbd>?</p>
        <div class="hr"></div>
        <div class="row">
          <button class="btn danger" id="doDel">Delete</button>
          <div class="spacer"></div>
          <button class="btn" id="cancelDel">Cancel</button>
        </div>
      `);
            $("#cancelDel").onclick = closeModal;
            $("#doDel").onclick = async () => {
                try {
                    await api(`/movies/${id}`, { method:"DELETE" });
                    toast("Deleted", `Movie #${id} removed`);
                    closeModal();
                    loadMovies();
                } catch(e) {
                    toast("Delete failed", e.message);
                }
            };
        }

        if (action === "edit") {
            try {
                const m = await api(`/movies/${id}`);
                openModal(`Edit Movie #${id}`, `
          <div class="grid">
            <div class="col-12 field">
              <label>Title</label>
              <input class="input" id="eTitle" value="${escapeHtml(m.title ?? "")}" />
            </div>
            <div class="col-6 field">
              <label>Genre</label>
              <input class="input" id="eGenre" value="${escapeHtml(m.genre ?? "")}" />
            </div>
            <div class="col-3 field">
              <label>Duration (min)</label>
              <input class="input" id="eDuration" type="number" min="1" value="${m.duration ?? 1}"/>
            </div>
            <div class="col-3 field">
              <label>Price (KZT)</label>
              <input class="input" id="ePrice" type="number" min="0" value="${m.price ?? 0}"/>
            </div>
          </div>
          <div class="hr"></div>
          <div class="row">
            <button class="btn primary" id="eSave">Save changes</button>
            <div class="spacer"></div>
            <button class="btn" id="eCancel">Cancel</button>
          </div>
        `);
                $("#eCancel").onclick = closeModal;
                $("#eSave").onclick = async () => {
                    try {
                        const patch = {
                            title: $("#eTitle").value,
                            genre: $("#eGenre").value,
                            duration: Number($("#eDuration").value),
                            price: Number($("#ePrice").value),
                        };
                        await api(`/movies/${id}`, { method:"PATCH", body: patch });
                        toast("Updated", `Movie #${id} updated`);
                        closeModal();
                        loadMovies();
                    } catch(e) {
                        toast("Update failed", e.message);
                    }
                };
            } catch(e) {
                toast("Load failed", e.message);
            }
        }
    });
}

// ===== Page: Book Ticket =====
function bookWire() {
    if (!$("#bookForm")) return;

    // +/- buttons
    document.addEventListener("click", (e) => {
        const b = e.target.closest("button[data-step]");
        if (!b) return;

        const step = b.dataset.step; // "seat" or "user"
        const delta = Number(b.dataset.delta || 0);

        const input = step === "seat" ? $("#seatId") : $("#userId");
        const cur = Number(input.value || 1);
        const next = Math.max(1, cur + delta);
        input.value = String(next);
    });

    // random buttons
    $("#seatRandom")?.addEventListener("click", () => {
        $("#seatId").value = String(Math.floor(Math.random() * 80) + 1); // 1..80
    });

    $("#userRandom")?.addEventListener("click", () => {
        $("#userId").value = String(Math.floor(Math.random() * 50) + 1); // 1..50
    });

    // fill example
    $("#fillExample")?.addEventListener("click", () => {
        $("#sessionId").value = "1";
        $("#seatId").value = "15";
        $("#userId").value = "7";
    });
    
}

// ===== Page: Ticket =====
async function ticketLoad() {
    const box = $("#ticketBox");
    if (!box) return;

    const params = new URLSearchParams(location.search);
    const id = Number(params.get("id") || 0);

    if (!id) {
        box.innerHTML = `<p class="small">No ticket id. Try: <kbd>ticket.html?id=1</kbd></p>`;
        return;
    }

    $("#ticketIdLabel").textContent = `#${id}`;
    box.innerHTML = `<p class="small">Loading ticket...</p>`;

    try {
        const t = await api(`/ticket?id=${id}`);
        box.innerHTML = `
      <div class="grid">
        <div class="col-6 card soft">
          <h2>Ticket ${escapeHtml(String(t.id))}</h2>
          <p>Status: <span class="badge ok">${escapeHtml(t.status ?? "UNKNOWN")}</span></p>
          <div class="hr"></div>
          <p class="small">Session: <kbd>${escapeHtml(String(t.session_id ?? "-"))}</kbd></p>
          <p class="small">Seat: <kbd>${escapeHtml(String(t.seat_id ?? "-"))}</kbd></p>
          <p class="small">User: <kbd>${escapeHtml(String(t.user_id ?? "-"))}</kbd></p>
          <p class="small">Price: <kbd>${escapeHtml(String(t.price ?? 0))} KZT</kbd></p>
        </div>
        <div class="col-6 card soft">
          <h2>Raw JSON</h2>
          <pre style="white-space:pre-wrap; margin:0; font-size:12px; color:rgba(234,240,255,.85)">${escapeHtml(JSON.stringify(t, null, 2))}</pre>
        </div>
      </div>
    `;
    } catch(e) {
        box.innerHTML = `<p class="small">Error: ${escapeHtml(e.message)}</p>`;
        toast("Ticket", e.message);
    }
}

function modalWire() {
    $("#modalClose")?.addEventListener("click", closeModal);
    $("#modal")?.addEventListener("click", (e) => {
        if (e.target.id === "modal") closeModal();
    });
    document.addEventListener("keydown", (e) => {
        if (e.key === "Escape") closeModal();
    });
}

// ===== Init =====
document.addEventListener("DOMContentLoaded", () => {
    setActiveNav();
    modalWire();
    moviesWire();
    loadMovies();
    bookWire();
    ticketLoad();

    // Health ping (optional)
    const health = $("#health");
    if (health) {
        (async () => {
            try {
                await fetch(API_BASE + "/");
                health.textContent = "API: Online";
                health.classList.add("badge","ok");
            } catch {
                health.textContent = "API: Offline";
                health.classList.add("badge");
            }
        })();
    }
});
