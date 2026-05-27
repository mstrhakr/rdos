const SETTINGS_TABS = [
  {
    id: "connection",
    label: "RDP",
    title: "Remote connection",
    subtitle: "Server, login, and launch defaults.",
    cards: [
      {
        title: "Connection basics",
        fields: [
          { key: "server", label: "Server", type: "text", placeholder: "rdp.example.local" },
          { key: "domain", label: "Domain", type: "text", placeholder: "domain" },
          { key: "param", label: "Parameter", type: "text", placeholder: "optional launch parameter" },
          { key: "config_url", label: "Config URL", type: "text", placeholder: "https://..." },
        ],
      },
      {
        title: "Support and login",
        fields: [
          { key: "adminpass", label: "Admin password", type: "password", placeholder: "admin password" },
          { key: "helpdesk", label: "Helpdesk", type: "text", placeholder: "the helpdesk" },
          { key: "login_timeout", label: "Login timeout", type: "number", placeholder: "600" },
        ],
      },
    ],
  },
  {
    id: "network",
    label: "Network",
    title: "Wired and wireless",
    subtitle: "DHCP, static IPv4, and WiFi connection control.",
    cards: [
      {
        title: "Wired settings",
        fields: [
          {
            key: "network_mode",
            label: "Mode",
            type: "select",
            options: [
              ["dhcp", "Automatic (DHCP)"],
              ["static", "Static IPv4"],
            ],
          },
          { key: "network_interface", label: "Interface", type: "text", placeholder: "eth0 or enp..." },
          { key: "static_address", label: "Address", type: "text", placeholder: "192.168.1.10" },
          { key: "static_prefix", label: "Prefix", type: "number", placeholder: "24" },
          { key: "static_gateway", label: "Gateway", type: "text", placeholder: "192.168.1.1" },
          { key: "static_dns", label: "DNS", type: "text", placeholder: "1.1.1.1 8.8.8.8" },
        ],
      },
      {
        title: "WiFi",
        custom: "wifi",
      },
    ],
  },
  {
    id: "device",
    label: "Device",
    title: "Audio, display, and background",
    subtitle: "The everyday appliance settings.",
    cards: [
      {
        title: "Device tuning",
        fields: [
          { key: "volume", label: "Volume", type: "number", placeholder: "100" },
          { key: "microphone", label: "Microphone", type: "number", placeholder: "100" },
          { key: "brightness", label: "Brightness", type: "number", placeholder: "50" },
          { key: "screen_timeout", label: "Screen timeout", type: "number", placeholder: "600" },
          { key: "keylayout", label: "Keyboard layout", type: "text", placeholder: "us" },
          {
            key: "exit_type",
            label: "Exit action",
            type: "select",
            options: [
              ["Shutdown", "Shutdown"],
              ["Sleep", "Sleep"],
              ["Restart", "Restart"],
              ["Exit", "Exit"],
            ],
          },
          {
            key: "wallpaper_mode",
            label: "Wallpaper mode",
            type: "select",
            options: [
              ["fit", "Fit"],
              ["max", "Fill"],
              ["stretch", "Stretch"],
              ["center", "Center"],
              ["tile", "Tile"],
            ],
          },
        ],
      },
      {
        title: "System summary",
        fields: [
          { key: "auto_update_enabled", label: "Auto update", type: "checkbox" },
          { key: "maintenance_window", label: "Maintenance window", type: "text", placeholder: "02:00" },
          {
            key: "ota_channel",
            label: "OTA channel",
            type: "select",
            options: [
              ["stable", "Stable"],
              ["beta", "Beta"],
              ["nightly", "Nightly"],
            ],
          },
        ],
      },
    ],
  },
  {
    id: "status",
    label: "Status",
    title: "Overlay and indicators",
    subtitle: "Status items that show on screen.",
    cards: [
      {
        title: "Overlay controls",
        fields: [
          { key: "battery_low_threshold", label: "Low battery threshold", type: "number", placeholder: "20" },
          { key: "status_overlay_enabled", label: "Overlay enabled", type: "checkbox" },
          { key: "status_show_datetime", label: "Show date and time", type: "checkbox" },
          { key: "status_show_battery", label: "Show battery", type: "checkbox" },
          { key: "status_show_wifi", label: "Show WiFi", type: "checkbox" },
          { key: "status_show_ip", label: "Show IP", type: "checkbox" },
          { key: "status_show_wireguard", label: "Show WireGuard", type: "checkbox" },
          { key: "wireguard_enabled", label: "WireGuard enabled", type: "checkbox" },
        ],
      },
      {
        title: "Operator note",
        fields: [],
        note: "These values mirror the legacy YAD settings so the Go UI stays compatible with existing deployments.",
      },
    ],
  },
  {
    id: "support",
    label: "Support",
    title: "Help and recovery",
    subtitle: "Admin contact and maintenance details.",
    cards: [
      {
        title: "Support contact",
        fields: [
          { key: "helpdesk", label: "Helpdesk text", type: "text", placeholder: "the helpdesk" },
          { key: "adminpass", label: "Admin password", type: "password", placeholder: "admin password" },
          { key: "config_url", label: "Provisioning URL", type: "text", placeholder: "https://..." },
        ],
      },
      {
        title: "System details",
        fields: [],
        note: "Use the A/B boot option at startup to switch between the web UI and the legacy shell UI.",
      },
    ],
  },
];

const appState = {
  config: {},
  network: null,
  health: null,
  session: null,
  status: null,
  wifiNetworks: [],
  activeTab: "connection",
};

const settingsFieldKeys = SETTINGS_TABS.flatMap((tab) =>
  tab.cards.flatMap((card) => (card.fields || []).map((field) => field.key)),
);

function api(path, method = "GET", body) {
  return fetch(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Request failed: ${res.status}`);
    }
    const contentType = res.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      return res.json();
    }
    return res.text();
  });
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function valueForKey(key) {
  return appState.config?.[key] ?? "";
}

function isChecked(value) {
  return ["1", "true", "yes", "on"].includes(String(value).trim().toLowerCase());
}

function updateText(id, value) {
  const element = document.getElementById(id);
  if (element) {
    element.textContent = value;
  }
}

function renderPills() {
  const pills = [
    ["Service", appState.health?.status || "unknown"],
    ["Boot", appState.health?.bootMode || appState.status?.bootMode || "unknown"],
    ["Time", appState.status?.time ? new Date(appState.status.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "--:--"],
    ["WiFi", appState.status?.wifi || "n/a"],
    ["IP", appState.status?.ip || "n/a"],
  ];
  const target = document.getElementById("statusPills");
  if (!target) {
    return;
  }
  target.innerHTML = pills
    .map(
      ([label, value]) => `
        <div class="pill">
          <span class="pill-subtle">${escapeHtml(label)}</span>
          <strong>${escapeHtml(value)}</strong>
        </div>
      `,
    )
    .join("");
}

function renderMetrics() {
  const metrics = [
    ["Hostname", appState.status?.hostname || "unknown"],
    ["Server", appState.status?.server || valueForKey("server") || "not configured"],
    ["Network", appState.status?.connection || valueForKey("network_mode") || "dhcp"],
    ["Wallpaper", appState.status?.wallpaper || valueForKey("wallpaper_mode") || "max"],
    ["Helpdesk", appState.status?.helpdesk || valueForKey("helpdesk") || "the helpdesk"],
    ["WireGuard", appState.status?.wireguard || "n/a"],
  ];
  const target = document.getElementById("statusMetrics");
  if (!target) {
    return;
  }
  target.innerHTML = metrics
    .map(
      ([label, value]) => `
        <div class="metric">
          <div class="label">${escapeHtml(label)}</div>
          <div class="value">${escapeHtml(value)}</div>
        </div>
      `,
    )
    .join("");
}

function renderWifiRows(containerId) {
  const target = document.getElementById(containerId);
  if (!target) {
    return;
  }

  if (appState.wifiNetworks.length === 0) {
    target.innerHTML = '<div class="status-card">No WiFi networks scanned yet.</div>';
    return;
  }

  target.innerHTML = appState.wifiNetworks
    .map(
      (network) => `
        <div class="wifi-row">
          <div>
            <div class="wifi-name">${escapeHtml(network.ssid)}</div>
            <div class="wifi-meta">${escapeHtml(network.security || "Open")} • ${escapeHtml(network.signal || "")}</div>
          </div>
          <span class="wifi-meta">${escapeHtml(network.interface || "")}</span>
          <span class="wifi-meta">${escapeHtml(network.security || "Open")}</span>
          <button type="button" data-wifi-ssid="${escapeHtml(network.ssid)}" data-wifi-security="${escapeHtml(network.security || "")}">Connect</button>
        </div>
      `,
    )
    .join("");

  target.querySelectorAll("button[data-wifi-ssid]").forEach((button) => {
    button.addEventListener("click", () => {
      const ssid = button.getAttribute("data-wifi-ssid") || "";
      const security = button.getAttribute("data-wifi-security") || "";
      openSettings("network");
      const ssidField = document.getElementById("wifiSSID");
      const passwordField = document.getElementById("wifiPassword");
      if (ssidField) {
        ssidField.value = ssid;
      }
      if (passwordField && security.toLowerCase() === "open") {
        passwordField.value = "";
      }
      updateText("settingsNote", `Prepared WiFi connection for ${ssid}.`);
    });
  });
}

function buildField(field) {
  const currentValue = valueForKey(field.key);
  if (field.type === "checkbox") {
    return `
      <div class="switch-row">
        <label>
          <input type="checkbox" data-config-key="${field.key}" ${isChecked(currentValue) ? "checked" : ""} />
          <span>${escapeHtml(field.label)}</span>
        </label>
      </div>
    `;
  }

  if (field.type === "select") {
    const options = field.options
      .map(([value, label]) => `<option value="${escapeHtml(value)}" ${String(currentValue) === value ? "selected" : ""}>${escapeHtml(label)}</option>`)
      .join("");
    return `
      <label>
        ${escapeHtml(field.label)}
        <select data-config-key="${field.key}">${options}</select>
      </label>
    `;
  }

  const inputType = field.type === "password" ? "password" : field.type === "number" ? "number" : "text";
  return `
    <label>
      ${escapeHtml(field.label)}
      <input
        type="${inputType}"
        data-config-key="${field.key}"
        value="${escapeHtml(currentValue)}"
        placeholder="${escapeHtml(field.placeholder || "")}" />
    </label>
  `;
}

function buildCard(card) {
  const fieldsMarkup = (card.fields || []).length
    ? `<div class="form-grid">${card.fields.map((field) => buildField(field)).join("")}</div>`
    : "";
  const noteMarkup = card.note ? `<div class="status-card">${escapeHtml(card.note)}</div>` : "";
  return `
    <section class="tab-card">
      <h3>${escapeHtml(card.title)}</h3>
      ${fieldsMarkup}
      ${noteMarkup}
    </section>
  `;
}

function renderSettingsPanels() {
  const tabsTarget = document.getElementById("settingsTabs");
  const panelsTarget = document.getElementById("settingsPanels");
  if (!tabsTarget || !panelsTarget) {
    return;
  }

  tabsTarget.innerHTML = SETTINGS_TABS.map(
    (tab) => `
      <button
        type="button"
        class="tab-button"
        data-tab-id="${tab.id}"
        role="tab"
        aria-selected="${tab.id === appState.activeTab ? "true" : "false"}"
      >
        ${escapeHtml(tab.label)}
      </button>
    `,
  ).join("");

  panelsTarget.innerHTML = SETTINGS_TABS.map((tab) => {
    const cardsMarkup = tab.cards.filter((card) => !card.custom).map((card) => buildCard(card)).join("");
    const intro = `<div class="status-card">${escapeHtml(tab.subtitle)}</div>`;
    const wifiBlock = tab.id === "network"
      ? `
        <section class="tab-card wifi-connect">
          <h3>WiFi connection</h3>
          <div class="form-grid">
            <label>Interface<input id="wifiInterface" type="text" placeholder="wlan0" value="${escapeHtml(appState.status?.wifiInterface || valueForKey("network_interface"))}" /></label>
            <label>SSID<input id="wifiSSID" type="text" placeholder="WiFi network" /></label>
            <label>Password<input id="wifiPassword" type="password" placeholder="password" /></label>
            <label class="switch-row"><input id="wifiHidden" type="checkbox" /> <span>Hidden network</span></label>
          </div>
          <div class="actions tight">
            <button type="button" id="scanWifiSettings" class="secondary">Scan WiFi</button>
            <button type="button" id="applyWifi">Connect WiFi</button>
          </div>
          <div id="settingsWifiList" class="wifi-list"></div>
        </section>
      `
      : "";

    return `
      <section class="tab-panel ${tab.id === appState.activeTab ? "active" : ""}" data-panel-id="${tab.id}" role="tabpanel">
        <div class="status-card">${escapeHtml(tab.title)}</div>
        <div class="tab-panel-grid">
          <div class="tab-column">${intro}${cardsMarkup}</div>
          <div class="tab-column">${wifiBlock}</div>
        </div>
      </section>
    `;
  }).join("");

  tabsTarget.querySelectorAll("[data-tab-id]").forEach((button) => {
    button.addEventListener("click", () => setActiveTab(button.getAttribute("data-tab-id") || "connection"));
  });

  attachSettingsActions();
  attachNetworkActions();
  syncConfigFields();
  syncNetworkFields();
  renderWifiRows("settingsWifiList");
}

function attachSettingsActions() {
  const scanButton = document.getElementById("scanWifiSettings");
  if (scanButton && !scanButton.dataset.bound) {
    scanButton.dataset.bound = "1";
    scanButton.addEventListener("click", () => scanWifi());
  }

  const applyButton = document.getElementById("applyWifi");
  if (applyButton && !applyButton.dataset.bound) {
    applyButton.dataset.bound = "1";
    applyButton.addEventListener("click", () => connectWifi());
  }
}

function attachNetworkActions() {
  const applyNetwork = document.getElementById("applyNetworkSettings");
  if (applyNetwork && !applyNetwork.dataset.bound) {
    applyNetwork.dataset.bound = "1";
    applyNetwork.addEventListener("click", () => saveNetwork());
  }

  const refreshNetworkButton = document.getElementById("refreshNetworkSettings");
  if (refreshNetworkButton && !refreshNetworkButton.dataset.bound) {
    refreshNetworkButton.dataset.bound = "1";
    refreshNetworkButton.addEventListener("click", () => refreshNetwork());
  }
}

function setActiveTab(tabId) {
  appState.activeTab = tabId;
  document.querySelectorAll(".tab-button").forEach((button) => {
    const selected = button.getAttribute("data-tab-id") === tabId;
    button.setAttribute("aria-selected", selected ? "true" : "false");
  });
  document.querySelectorAll(".tab-panel").forEach((panel) => {
    const active = panel.getAttribute("data-panel-id") === tabId;
    panel.classList.toggle("active", active);
  });
  renderSettingsPanels();
}

function syncConfigFields() {
  document.querySelectorAll("[data-config-key]").forEach((field) => {
    const key = field.getAttribute("data-config-key");
    if (!key) {
      return;
    }
    const value = valueForKey(key);
    if (field.type === "checkbox") {
      field.checked = isChecked(value);
    } else if (field.tagName === "SELECT") {
      field.value = value || field.options[0]?.value || "";
    } else {
      field.value = value || "";
    }
  });
}

function syncNetworkFields() {
  if (!appState.network) {
    return;
  }
  const fields = [
    ["networkMode", appState.network.mode || "dhcp"],
    ["networkInterface", appState.network.interface || ""],
    ["networkAddress", appState.network.address || ""],
    ["networkPrefix", appState.network.prefix || ""],
    ["networkGateway", appState.network.gateway || ""],
    ["networkDNS", appState.network.dns || ""],
  ];
  fields.forEach(([id, value]) => {
    const element = document.getElementById(id);
    if (element) {
      element.value = value;
    }
  });
}

function buildNetworkForm() {
  return `
    <section class="tab-card">
      <h3>Wired network</h3>
      <div class="form-grid">
        <label>Mode
          <select id="networkMode">
            <option value="dhcp">Automatic (DHCP)</option>
            <option value="static">Static IPv4</option>
          </select>
        </label>
        <label>Interface<input id="networkInterface" type="text" placeholder="eth0" /></label>
        <label>Address<input id="networkAddress" type="text" placeholder="192.168.1.10" /></label>
        <label>Prefix<input id="networkPrefix" type="number" placeholder="24" /></label>
        <label>Gateway<input id="networkGateway" type="text" placeholder="192.168.1.1" /></label>
        <label>DNS<input id="networkDNS" type="text" placeholder="1.1.1.1 8.8.8.8" /></label>
      </div>
      <div class="actions tight">
        <button type="button" id="refreshNetworkSettings" class="secondary">Reload</button>
        <button type="button" id="applyNetworkSettings">Apply wired settings</button>
      </div>
    </section>
  `;
}

function buildMainNetworkRows() {
  renderWifiRows("wifiList");
}

function bindMainActions() {
  document.getElementById("connect")?.addEventListener("click", connectSession);
  document.getElementById("disconnect")?.addEventListener("click", disconnectSession);
  document.getElementById("refreshAll")?.addEventListener("click", refreshAll);
  document.getElementById("openSettings")?.addEventListener("click", () => openSettings("connection"));
  document.getElementById("openNetworkTab")?.addEventListener("click", () => openSettings("network"));
  document.getElementById("scanWifi")?.addEventListener("click", () => scanWifi());
  document.getElementById("loadNetwork")?.addEventListener("click", refreshNetwork);
  document.getElementById("saveNetwork")?.addEventListener("click", saveNetwork);
}

function bindModalActions() {
  document.getElementById("closeSettings")?.addEventListener("click", closeSettings);
  document.getElementById("reloadSettings")?.addEventListener("click", async () => {
    await refreshConfig();
    await refreshNetwork();
    renderSettingsPanels();
  });
  document.getElementById("saveSettings")?.addEventListener("click", saveSettings);
}

function openSettings(tabId = "connection") {
  appState.activeTab = tabId;
  const modal = document.getElementById("settingsModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "false");
  renderSettingsPanels();
  syncConfigFields();
  syncNetworkFields();
  updateText("settingsNote", "Changes save to tcconfig.");
}

function closeSettings() {
  const modal = document.getElementById("settingsModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "true");
}

async function refreshHealth() {
  try {
    appState.health = await api("/api/v1/health");
    updateText("health", `${appState.health.status} | ${appState.health.service} ${appState.health.version}`);
  } catch (err) {
    appState.health = null;
    updateText("health", `backend offline: ${err.message}`);
  }
  renderPills();
}

async function refreshSession() {
  try {
    appState.session = await api("/api/v1/session");
    updateText("sessionState", JSON.stringify(appState.session, null, 2));
  } catch (err) {
    updateText("sessionState", `session error: ${err.message}`);
  }
}

async function refreshConfig() {
  try {
    const payload = await api("/api/v1/config");
    appState.config = payload.values || {};
    renderSettingsPanels();
    updateText("configState", `${Object.keys(appState.config).length} configuration values loaded`);
  } catch (err) {
    updateText("configState", `config error: ${err.message}`);
  }
  renderMetrics();
}

async function refreshNetwork() {
  try {
    appState.network = await api("/api/v1/network");
    syncNetworkFields();
    updateText("networkState", JSON.stringify(appState.network, null, 2));
  } catch (err) {
    updateText("networkState", `network error: ${err.message}`);
  }
}

async function refreshStatus() {
  try {
    appState.status = await api("/api/v1/status");
    renderPills();
    renderMetrics();
  } catch (err) {
    updateText("configState", `status error: ${err.message}`);
  }
}

async function refreshWifi() {
  try {
    const payload = await api("/api/v1/wifi/scan");
    appState.wifiNetworks = payload.networks || [];
    buildMainNetworkRows();
    renderWifiRows("settingsWifiList");
  } catch (err) {
    appState.wifiNetworks = [];
    const message = `wifi scan error: ${err.message}`;
    updateText("wifiList", message);
    updateText("settingsWifiList", message);
  }
}

async function refreshAll() {
  await Promise.all([refreshHealth(), refreshSession(), refreshConfig(), refreshNetwork(), refreshStatus(), refreshWifi()]);
}

async function connectSession() {
  const payload = {
    server: document.getElementById("server")?.value.trim(),
    username: document.getElementById("username")?.value.trim(),
    password: document.getElementById("password")?.value,
    domain: document.getElementById("domain")?.value.trim(),
    certPolicy: "tofu",
  };

  try {
    await api("/api/v1/session/connect", "POST", payload);
    await refreshSession();
    updateText("settingsNote", `Connecting to ${payload.server || "the server"}.`);
  } catch (err) {
    updateText("sessionState", `connect error: ${err.message}`);
  }
}

async function disconnectSession() {
  try {
    await api("/api/v1/session/disconnect", "POST", {});
    await refreshSession();
  } catch (err) {
    updateText("sessionState", `disconnect error: ${err.message}`);
  }
}

function collectConfigValues() {
  const values = {};
  document.querySelectorAll("[data-config-key]").forEach((field) => {
    const key = field.getAttribute("data-config-key");
    if (!key || !settingsFieldKeys.includes(key)) {
      return;
    }
    if (field.type === "checkbox") {
      values[key] = field.checked ? "true" : "false";
    } else {
      values[key] = field.value.trim();
    }
  });
  return values;
}

async function saveSettings() {
  try {
    const values = collectConfigValues();
    const payload = await api("/api/v1/config", "POST", { values });
    appState.config = payload.values || values;
    renderSettingsPanels();
    await refreshStatus();
    updateText("settingsNote", "Settings saved.");
  } catch (err) {
    updateText("settingsNote", `save error: ${err.message}`);
  }
}

async function saveNetwork() {
  const payload = {
    mode: document.getElementById("networkMode")?.value || "dhcp",
    interface: document.getElementById("networkInterface")?.value.trim() || "",
    address: document.getElementById("networkAddress")?.value.trim() || "",
    prefix: document.getElementById("networkPrefix")?.value.trim() || "",
    gateway: document.getElementById("networkGateway")?.value.trim() || "",
    dns: document.getElementById("networkDNS")?.value.trim() || "",
  };

  try {
    const saved = await api("/api/v1/network", "POST", payload);
    appState.network = saved;
    syncNetworkFields();
    updateText("networkState", JSON.stringify(saved, null, 2));
    updateText("settingsNote", "Wired network saved.");
  } catch (err) {
    updateText("networkState", `network save error: ${err.message}`);
  }
}

async function scanWifi() {
  try {
    const interfaceValue = document.getElementById("wifiInterface")?.value.trim() || "";
    const query = interfaceValue ? `?interface=${encodeURIComponent(interfaceValue)}` : "";
    const payload = await api(`/api/v1/wifi/scan${query}`);
    appState.wifiNetworks = payload.networks || [];
    buildMainNetworkRows();
    renderWifiRows("settingsWifiList");
    updateText("settingsNote", `Found ${appState.wifiNetworks.length} WiFi network(s).`);
  } catch (err) {
    appState.wifiNetworks = [];
    const message = `wifi scan error: ${err.message}`;
    updateText("wifiList", message);
    updateText("settingsWifiList", message);
  }
}

async function connectWifi() {
  const payload = {
    interface: document.getElementById("wifiInterface")?.value.trim() || "",
    ssid: document.getElementById("wifiSSID")?.value.trim() || "",
    password: document.getElementById("wifiPassword")?.value || "",
    hidden: Boolean(document.getElementById("wifiHidden")?.checked),
  };

  try {
    await api("/api/v1/wifi/connect", "POST", payload);
    updateText("settingsNote", `Connecting to ${payload.ssid || "WiFi"}.`);
    await refreshNetwork();
    await refreshStatus();
  } catch (err) {
    updateText("settingsNote", `wifi connect error: ${err.message}`);
  }
}

function wireGlobalShortcuts() {
  window.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeSettings();
    }
  });
}

function initStaticPanels() {
  const networkPanel = document.querySelector(".network-panel .network-layout");
  if (networkPanel) {
    networkPanel.innerHTML = `
      <div class="network-block">
        <h3>Wireless</h3>
        <div class="actions tight">
          <button id="scanWifi" class="secondary" type="button">Scan</button>
          <button id="openNetworkTab" class="secondary" type="button">Open settings</button>
        </div>
        <div id="wifiList" class="wifi-list"></div>
      </div>

      <div class="network-block">
        <h3>Wired</h3>
        <div class="status-card compact" id="networkState">network not loaded</div>
        <div class="actions tight">
          <button id="loadNetwork" class="secondary" type="button">Reload</button>
          <button id="saveNetwork" type="button">Apply</button>
        </div>
      </div>
    `;
  }

  const modalPanels = document.getElementById("settingsPanels");
  if (modalPanels) {
    modalPanels.innerHTML = [
      `
        <section class="tab-panel active" data-panel-id="connection" role="tabpanel">
          <div class="status-card">Remote desktop launch defaults.</div>
          <div class="tab-panel-grid">
            <div class="tab-column">
              ${SETTINGS_TABS[0].cards.map((card) => buildCard(card)).join("")}
            </div>
            <div class="tab-column">
              <section class="tab-card">
                <h3>Operator note</h3>
                <div class="status-card">This screen stays locked to the browser viewport so it behaves like a device panel, not a website.</div>
              </section>
            </div>
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="network" role="tabpanel">
          <div class="status-card">Wired and wireless connection management.</div>
          <div class="tab-panel-grid">
            <div class="tab-column">
              ${buildNetworkForm()}
            </div>
            <div class="tab-column">
              <section class="tab-card wifi-connect">
                <h3>WiFi connection</h3>
                <div class="form-grid">
                  <label>Interface<input id="wifiInterface" type="text" placeholder="wlan0" value="${escapeHtml(appState.status?.wifiInterface || valueForKey("network_interface"))}" /></label>
                  <label>SSID<input id="wifiSSID" type="text" placeholder="WiFi network" /></label>
                  <label>Password<input id="wifiPassword" type="password" placeholder="password" /></label>
                  <label class="switch-row"><input id="wifiHidden" type="checkbox" /> <span>Hidden network</span></label>
                </div>
                <div class="actions tight">
                  <button type="button" id="scanWifiSettings" class="secondary">Scan WiFi</button>
                  <button type="button" id="applyWifi">Connect WiFi</button>
                </div>
                <div id="settingsWifiList" class="wifi-list"></div>
              </section>
            </div>
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="device" role="tabpanel">
          <div class="status-card">Audio, brightness, keyboard layout, and background mode.</div>
          <div class="tab-panel-grid">
            ${SETTINGS_TABS[2].cards.map((card) => buildCard(card)).join("")}
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="status" role="tabpanel">
          <div class="status-card">Overlay toggles and indicators used by the desktop shell.</div>
          <div class="tab-panel-grid">
            ${SETTINGS_TABS[3].cards.map((card) => buildCard(card)).join("")}
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="support" role="tabpanel">
          <div class="status-card">Recovery, helpdesk, and deployment details.</div>
          <div class="tab-panel-grid">
            ${SETTINGS_TABS[4].cards.map((card) => buildCard(card)).join("")}
          </div>
        </section>
      `,
    ].join("");
  }

  document.getElementById("scanWifi")?.addEventListener("click", () => scanWifi());
  document.getElementById("openNetworkTab")?.addEventListener("click", () => openSettings("network"));
  document.getElementById("loadNetwork")?.addEventListener("click", refreshNetwork);
  document.getElementById("saveNetwork")?.addEventListener("click", saveNetwork);
  attachSettingsActions();
}

window.addEventListener("DOMContentLoaded", async () => {
  initStaticPanels();
  bindMainActions();
  bindModalActions();
  wireGlobalShortcuts();

  await refreshAll();
  await scanWifi();
  renderSettingsPanels();
  setInterval(refreshSession, 5000);
  setInterval(refreshStatus, 7000);
  setInterval(refreshHealth, 15000);
});
