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
          {
            key: "cert_policy",
            label: "Certificate policy",
            type: "select",
            options: [
              ["tofu", "Trust on first use (save key)"],
              ["ignore", "Ignore certificate checks (insecure)"],
            ],
          },
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
    ],
  },
  {
    id: "updates",
    label: "Updates",
    title: "OTA management",
    subtitle: "Track the active slot, update channel, and rollback readiness.",
    cards: [
      {
        title: "Update policy",
        fields: [
          { key: "auto_check_enabled", label: "Auto check for updates", type: "checkbox" },
          {
            key: "auto_check_schedule",
            label: "Auto check schedule",
            type: "select",
            options: [
              ["hourly", "Hourly"],
              ["daily", "Daily"],
              ["weekly", "Weekly"],
            ],
          },
          { key: "auto_update_enabled", label: "Auto update", type: "checkbox" },
          {
            key: "auto_update_schedule",
            label: "Auto update schedule",
            type: "select",
            options: [
              ["hourly", "Hourly"],
              ["daily", "Daily"],
              ["weekly", "Weekly"],
            ],
          },
          { key: "maintenance_window", label: "Maintenance window", type: "text", placeholder: "02:00" },
          {
            key: "ota_channel",
            label: "OTA channel",
            type: "select",
            options: [
              ["stable", "Stable"],
              ["beta", "Beta"],
            ],
          },
          { key: "update_pin_semver", label: "Semver pin", type: "text", placeholder: "0 or 1.2 or 1.2.3" },
          { key: "update_pin_prefix", label: "Tag prefix pin", type: "text", placeholder: "v0 or v1.2" },
        ],
      },
      {
        title: "Operator note",
        fields: [],
        note: "RDOS updates stay platform-level and A/B slot based; the web UI is only a control surface for that flow.",
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
    id: "terminal",
    label: "Terminal",
    title: "Recovery terminal",
    subtitle: "Run shell commands when direct console access is unavailable.",
    cards: [],
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
  networkInterfaces: {
    interfaces: [],
    wireless: [],
    hasWireless: false,
    defaultInterface: "",
    defaultWireless: "",
    details: [],
  },
  health: null,
  ota: null,
  otaReleases: [],
  otaCatalog: [],
  otaCheck: null,
  otaUsbImages: [],
  otaUsbEvent: null,
  selectedOtaTag: "",
  session: null,
  status: null,
  wifiNetworks: [],
  wireguardUSBConfigs: [],
  terminal: {
    running: false,
    ready: false,
    url: "http://127.0.0.1:7681/",
    message: "Console is stopped.",
  },
  certPrompt: {
    open: false,
    lastSignature: "",
    dismissedSignature: "",
  },
  otaConfirm: {
    targetTag: "",
  },
  wifiConnectDraft: {
    interface: "",
    ssid: "",
    security: "",
    hidden: false,
  },
  activeTab: "connection",
};

const settingsFieldKeys = SETTINGS_TABS.flatMap((tab) =>
  tab.cards.flatMap((card) => (card.fields || []).map((field) => field.key)),
);

const NETWORK_CONFIG_KEYS = new Set([
  "network_mode",
  "network_interface",
  "static_address",
  "static_prefix",
  "static_gateway",
  "static_dns",
]);

const OTA_POLICY_KEYS = new Set([
  "maintenance_window",
  "ota_channel",
  "auto_check_enabled",
  "auto_check_schedule",
  "auto_update_enabled",
  "auto_update_schedule",
  "update_pin_semver",
  "update_pin_prefix",
]);

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

function normalizeCertPolicy(value) {
  const normalized = String(value || "").trim().toLowerCase();
  if (normalized === "ignore") {
    return "ignore";
  }
  return "tofu";
}

function certificateIssueFromSnapshot(snapshot) {
  if (!snapshot || snapshot.state !== "error") {
    return null;
  }
  if (normalizeCertPolicy(valueForKey("cert_policy")) === "ignore") {
    return null;
  }
  const details = [snapshot.message || "", snapshot.lastOutput || ""].filter(Boolean).join("\n").trim();
  if (!details) {
    return null;
  }

  const lower = details.toLowerCase();
  const certAccepted = [
    "no certificate stored, automatically accepting",
    "automatically accepting",
  ].some((token) => lower.includes(token));

  if (certAccepted) {
    return null;
  }

  const hasCertSignal = [
    "host key verification failed",
    "certificate verification failure",
    "self-signed certificate",
    "add correct host key",
    "the fingerprint for the host key",
    "host key for",
    "certificate has changed",
  ].some((token) => lower.includes(token));

  if (!hasCertSignal) {
    return null;
  }

  return {
    details,
    signature: `${snapshot.exitCode || ""}|${snapshot.message || ""}|${snapshot.lastOutput || ""}`,
  };
}

function openCertTrustModal(issue) {
  const modal = document.getElementById("certTrustModal");
  if (!modal) {
    return;
  }
  appState.certPrompt.open = true;
  modal.setAttribute("aria-hidden", "false");
  updateText("certTrustDetails", issue.details || "Certificate/security verification failed.");
}

function closeCertTrustModal() {
  const modal = document.getElementById("certTrustModal");
  if (!modal) {
    return;
  }
  appState.certPrompt.open = false;
  appState.certPrompt.dismissedSignature = appState.certPrompt.lastSignature;
  modal.setAttribute("aria-hidden", "true");
}

function openOTAConfirmModal(tag) {
  const modal = document.getElementById("otaConfirmModal");
  if (!modal) {
    return;
  }
  appState.otaConfirm.targetTag = String(tag || "").trim();
  const target = appState.otaConfirm.targetTag || "(none)";
  updateText("otaConfirmDetails", `Target release: ${target}`);
  modal.setAttribute("aria-hidden", "false");
}

function closeOTAConfirmModal() {
  const modal = document.getElementById("otaConfirmModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "true");
}

async function persistCertPolicy(policy) {
  const normalized = normalizeCertPolicy(policy);
  const payload = await api("/api/v1/config", "POST", { values: { cert_policy: normalized } });
  appState.config = payload.values || { ...appState.config, cert_policy: normalized };
  renderSettingsPanels();
  return normalized;
}

function maybePromptCertificateTrust(snapshot) {
  const issue = certificateIssueFromSnapshot(snapshot);
  if (!issue) {
    return;
  }
  if (appState.certPrompt.dismissedSignature === issue.signature) {
    return;
  }
  if (appState.certPrompt.open && appState.certPrompt.lastSignature === issue.signature) {
    return;
  }
  appState.certPrompt.lastSignature = issue.signature;
  openCertTrustModal(issue);
}

function renderCornerClock() {
  const target = document.getElementById("cornerClock");
  if (!target) {
    return;
  }
  const source = appState.status?.time ? new Date(appState.status.time) : new Date();
  target.textContent = source.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function networkInterfaceOptionsMarkup() {
  const all = appState.networkInterfaces.interfaces || [];
  const selected = appState.network?.interface || appState.networkInterfaces.defaultInterface || "";
  const options = ['<option value="">Automatic</option>'];
  all.forEach((iface) => {
    const isSelected = iface === selected ? " selected" : "";
    options.push(`<option value="${escapeHtml(iface)}"${isSelected}>${escapeHtml(iface)}</option>`);
  });
  return options.join("");
}

function wifiInterfaceOptionsMarkup() {
  const wireless = appState.networkInterfaces.wireless || [];
  const selected = appState.networkInterfaces.defaultWireless || appState.status?.wifiInterface || "";
  if (wireless.length === 0) {
    return '<option value="">No wireless adapter detected</option>';
  }
  return wireless
    .map((iface) => `<option value="${escapeHtml(iface)}"${iface === selected ? " selected" : ""}>${escapeHtml(iface)}</option>`)
    .join("");
}

function isModalOpen(id) {
  const modal = document.getElementById(id);
  return Boolean(modal && modal.getAttribute("aria-hidden") === "false");
}

function syncInterfaceFields() {
  const networkInterface = document.getElementById("networkInterface");
  if (networkInterface) {
    networkInterface.innerHTML = networkInterfaceOptionsMarkup();
    const selected = appState.network?.interface || appState.networkInterfaces.defaultInterface || "";
    networkInterface.value = selected;
  }

  const wifiInterface = document.getElementById("wifiInterface");
  if (wifiInterface) {
    wifiInterface.innerHTML = wifiInterfaceOptionsMarkup();
    wifiInterface.disabled = !appState.networkInterfaces.hasWireless;
    const selectedWifi = appState.networkInterfaces.defaultWireless || wifiInterface.value || "";
    wifiInterface.value = selectedWifi;
  }

  const wifiScanInterface = document.getElementById("wifiScanInterface");
  if (wifiScanInterface) {
    wifiScanInterface.innerHTML = wifiInterfaceOptionsMarkup();
    wifiScanInterface.disabled = !appState.networkInterfaces.hasWireless;
    const selectedWifi = appState.networkInterfaces.defaultWireless || wifiScanInterface.value || "";
    wifiScanInterface.value = selectedWifi;
  }

  const wifiConnectInterface = document.getElementById("wifiConnectInterface");
  if (wifiConnectInterface) {
    wifiConnectInterface.innerHTML = wifiInterfaceOptionsMarkup();
    wifiConnectInterface.disabled = !appState.networkInterfaces.hasWireless;
    const selectedWifi = appState.wifiConnectDraft.interface || appState.networkInterfaces.defaultWireless || wifiConnectInterface.value || "";
    wifiConnectInterface.value = selectedWifi;
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
      openWifiConnectModal({ ssid, security });
    });
  });
}

function openWifiScanModal() {
  const modal = document.getElementById("wifiScanModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "false");
  syncInterfaceFields();
  renderWifiRows("modalWifiList");
}

function closeWifiScanModal() {
  const modal = document.getElementById("wifiScanModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "true");
}

function openWifiConnectModal(network = {}) {
  const modal = document.getElementById("wifiConnectModal");
  if (!modal) {
    return;
  }

  appState.wifiConnectDraft = {
    interface: appState.networkInterfaces.defaultWireless || appState.status?.wifiInterface || "",
    ssid: network.ssid || "",
    security: network.security || "",
    hidden: false,
  };

  const wifiConnectInterface = document.getElementById("wifiConnectInterface");
  if (wifiConnectInterface) {
    wifiConnectInterface.innerHTML = wifiInterfaceOptionsMarkup();
    wifiConnectInterface.value = appState.wifiConnectDraft.interface;
  }

  const ssidField = document.getElementById("wifiConnectSSID");
  if (ssidField) {
    ssidField.value = appState.wifiConnectDraft.ssid;
  }

  const passwordField = document.getElementById("wifiConnectPassword");
  if (passwordField) {
    passwordField.value = "";
    if (String(network.security || "").toLowerCase() === "open") {
      passwordField.placeholder = "open network - leave blank";
    } else {
      passwordField.placeholder = "password";
    }
  }

  const hiddenField = document.getElementById("wifiConnectHidden");
  if (hiddenField) {
    hiddenField.checked = false;
  }

  updateText("wifiConnectState", network.ssid ? `Connecting to ${network.ssid}` : "Pick a network to connect.");
  modal.setAttribute("aria-hidden", "false");
}

function closeWifiConnectModal() {
  const modal = document.getElementById("wifiConnectModal");
  if (!modal) {
    return;
  }
  modal.setAttribute("aria-hidden", "true");
}

function renderWireGuardUSBRows(containerId) {
  const target = document.getElementById(containerId);
  if (!target) {
    return;
  }

  if (!appState.wireguardUSBConfigs.length) {
    target.innerHTML = '<div class="status-card">No WireGuard tunnel configs were found on mounted USB drives.</div>';
    return;
  }

  target.innerHTML = appState.wireguardUSBConfigs
    .map((config) => {
      const actionLabel = config.needsImport ? "Import" : "Installed";
      const disabled = config.needsImport ? "" : "disabled";
      return `
        <div class="wifi-row">
          <div>
            <div class="wifi-name">${escapeHtml(config.filename)}</div>
            <div class="wifi-meta">${escapeHtml(config.interface)} • ${escapeHtml(config.mount)}</div>
          </div>
          <span class="wifi-meta">${config.needsImport ? "Ready" : "Up to date"}</span>
          <button type="button" data-wireguard-path="${escapeHtml(config.path)}" ${disabled}>${actionLabel}</button>
        </div>
      `;
    })
    .join("");

  target.querySelectorAll("button[data-wireguard-path]").forEach((button) => {
    button.addEventListener("click", () => {
      const path = button.getAttribute("data-wireguard-path") || "";
      if (path) {
        importWireGuardUSB(path);
      }
    });
  });
}

function formatBytes(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function renderOTAUSBRows(containerId) {
  const target = document.getElementById(containerId);
  if (!target) {
    return;
  }

  if (!appState.otaUsbImages.length) {
    target.innerHTML = '<div class="status-card">No OTA images detected on USB media.</div>';
    return;
  }

  target.innerHTML = appState.otaUsbImages
    .map((image) => `
      <div class="wifi-row">
        <div>
          <div class="wifi-name">${escapeHtml(image.filename || "image")}</div>
          <div class="wifi-meta">${escapeHtml(image.mount || "")}</div>
        </div>
        <span class="wifi-meta">${escapeHtml(formatBytes(image.size))}</span>
        <button type="button" data-ota-usb-path="${escapeHtml(image.path)}">Import</button>
      </div>
    `)
    .join("");

  target.querySelectorAll("button[data-ota-usb-path]").forEach((button) => {
    button.addEventListener("click", () => {
      const path = button.getAttribute("data-ota-usb-path") || "";
      if (path) {
        importOTAUSB(path);
      }
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

function buildOTAPanel(tab) {
  const ota = appState.ota;
  const details = ota ? JSON.stringify(ota, null, 2) : "ota not loaded";
  const activeSlot = ota?.currentSlot || "n/a";
  const previousSlot = ota?.previousSlot || "n/a";
  const bootTries = ota?.bootTries || "n/a";
  const currentVersion = ota?.currentVersion || "unknown";
  const inactiveVersion = ota?.inactiveVersion || "unknown";
  const recovery = ota?.pendingRecovery ? "pending" : "clear";
  const entries = appState.otaCatalog || [];
  const selectedTag = appState.selectedOtaTag || entries[0]?.tag || "";
  const latestTag = appState.otaCheck?.latestTag || entries[0]?.tag || "";
  const checkState = appState.otaCheck?.running
    ? "Checking in background..."
    : (appState.otaCheck?.checkedAt
      ? `Last checked ${new Date(appState.otaCheck.checkedAt).toLocaleString()}`
      : "No background check yet.");

  const releaseOptions = entries.length
    ? entries
        .map((entry) => {
          const labels = (entry.labels || []).join(", ");
          const label = `${entry.tag}${labels ? ` [${labels}]` : ""}${entry.publishedAt ? ` - ${entry.publishedAt.slice(0, 10)}` : ""}`;
          return `<option value="${escapeHtml(entry.tag)}"${entry.tag === selectedTag ? " selected" : ""}>${escapeHtml(label)}</option>`;
        })
        .join("")
    : "<option value=\"\">No releases available</option>";

  return `
    <section class="tab-panel ${tab.id === appState.activeTab ? "active" : ""}" data-panel-id="${tab.id}" role="tabpanel">
      <div class="status-card">${escapeHtml(tab.title)}</div>
      <div class="tab-panel-grid">
        <div class="tab-column">
          <div class="status-card">${escapeHtml(tab.subtitle)}</div>
          ${tab.cards.map((card) => buildCard(card)).join("")}
        </div>
        <div class="tab-column">
          <section class="tab-card">
            <h3>Slot status</h3>
            <div class="status-card compact" id="otaState">${escapeHtml(details)}</div>
            <div class="form-grid">
              <label>Active slot<input type="text" value="${escapeHtml(activeSlot)}" readonly /></label>
              <label>Previous slot<input type="text" value="${escapeHtml(previousSlot)}" readonly /></label>
              <label>Boot tries<input type="text" value="${escapeHtml(bootTries)}" readonly /></label>
              <label>Recovery<input type="text" value="${escapeHtml(recovery)}" readonly /></label>
              <label>Current version<input type="text" value="${escapeHtml(currentVersion)}" readonly /></label>
              <label>Inactive version<input type="text" value="${escapeHtml(inactiveVersion)}" readonly /></label>
              <label>Latest available<input type="text" value="${escapeHtml(latestTag || "none")}" readonly /></label>
              <label>Check status<input type="text" value="${escapeHtml(checkState)}" readonly /></label>
            </div>
            <h3>Available releases</h3>
            <div class="form-grid">
              <label>Target release
                <select id="otaReleaseTag">${releaseOptions}</select>
              </label>
            </div>
            <div class="actions tight">
              <button type="button" id="runOTACheck">Check for updates</button>
              <button type="button" id="refreshOTAStatus" class="secondary">Refresh OTA</button>
              <button type="button" id="refreshOTACatalog" class="secondary">Refresh releases</button>
              <button type="button" id="runOTAUpdate" ${selectedTag ? "" : "disabled"}>Update to selected</button>
              <button type="button" id="runOTARollback" ${ota?.canRollback ? "" : "disabled"}>Rollback to previous slot</button>
            </div>
          </section>

          <section class="tab-card wifi-connect">
            <h3>USB image update</h3>
            <div class="status-card compact" id="otaUsbEventState">No USB OTA event detected.</div>
            <div class="actions tight">
              <button type="button" id="scanOTAUsb" class="secondary">Scan USB images</button>
              <button type="button" id="refreshOTAUsbEvent" class="secondary">Refresh insert event</button>
            </div>
            <div id="otaUsbList" class="wifi-list"></div>
          </section>
        </div>
      </div>
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
    if (tab.id === "updates") {
      return buildOTAPanel(tab);
    }

    if (tab.id === "terminal") {
      return buildTerminalPanel(tab);
    }

    if (tab.id === "network") {
      return `
        <section class="tab-panel ${tab.id === appState.activeTab ? "active" : ""}" data-panel-id="${tab.id}" role="tabpanel">
          <div class="status-card">${escapeHtml(tab.title)}</div>
          <div class="tab-panel-grid">
            <div class="tab-column">
              <div class="status-card">${escapeHtml(tab.subtitle)}</div>
              ${buildNetworkStatusSection()}
              ${buildNetworkForm()}
            </div>
            <div class="tab-column">
              <section class="tab-card wifi-connect">
                <h3>WiFi connection</h3>
                <div class="form-grid">
                  <label>Interface
                    <select id="wifiInterface">${wifiInterfaceOptionsMarkup()}</select>
                  </label>
                  <label>SSID<input id="wifiSSID" type="text" placeholder="WiFi network" /></label>
                  <label>Password<input id="wifiPassword" type="password" placeholder="password" /></label>
                  <label class="switch-row"><input id="wifiHidden" type="checkbox" /> <span>Hidden network</span></label>
                </div>
                <div class="actions tight">
                  <button type="button" id="scanWifiSettings" class="secondary">Scan WiFi</button>
                  <button type="button" id="refreshWifiInterfaces" class="secondary">Refresh adapters</button>
                  <button type="button" id="applyWifi">Connect WiFi</button>
                </div>
                <div id="settingsWifiList" class="wifi-list"></div>
              </section>

              <section class="tab-card wifi-connect">
                <h3>WireGuard from USB</h3>
                <div class="status-card">Insert a USB drive containing a <strong>wg*.conf</strong> tunnel file, then scan for it here.</div>
                <div class="actions tight">
                  <button type="button" id="scanWireGuardUsb" class="secondary">Scan USB</button>
                </div>
                <div id="wireguardUsbList" class="wifi-list"></div>
              </section>
            </div>
          </div>
        </section>
      `;
    }

    const cardsMarkup = tab.cards.filter((card) => !card.custom).map((card) => buildCard(card)).join("");
    const intro = `<div class="status-card">${escapeHtml(tab.subtitle)}</div>`;

    return `
      <section class="tab-panel ${tab.id === appState.activeTab ? "active" : ""}" data-panel-id="${tab.id}" role="tabpanel">
        <div class="status-card">${escapeHtml(tab.title)}</div>
        <div class="tab-panel-grid">
          <div class="tab-column">${intro}${cardsMarkup}</div>
          <div class="tab-column"></div>
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
  syncInterfaceFields();
  renderWifiRows("settingsWifiList");
  renderWireGuardUSBRows("wireguardUsbList");
  renderOTAUSBRows("otaUsbList");
}

function attachSettingsActions() {
  const scanButton = document.getElementById("scanWifiSettings");
  if (scanButton && !scanButton.dataset.bound) {
    scanButton.dataset.bound = "1";
    scanButton.addEventListener("click", () => scanWifi(true));
  }

  const applyButton = document.getElementById("applyWifi");
  if (applyButton && !applyButton.dataset.bound) {
    applyButton.dataset.bound = "1";
    applyButton.addEventListener("click", () => connectWifi());
  }

  const refreshWifiAdaptersButton = document.getElementById("refreshWifiInterfaces");
  if (refreshWifiAdaptersButton && !refreshWifiAdaptersButton.dataset.bound) {
    refreshWifiAdaptersButton.dataset.bound = "1";
    refreshWifiAdaptersButton.addEventListener("click", async () => {
      await refreshNetworkInterfaces();
      await scanWifi();
    });
  }

  const refreshNetworkAdaptersButton = document.getElementById("refreshNetworkInterfaces");
  if (refreshNetworkAdaptersButton && !refreshNetworkAdaptersButton.dataset.bound) {
    refreshNetworkAdaptersButton.dataset.bound = "1";
    refreshNetworkAdaptersButton.addEventListener("click", async () => {
      await refreshNetworkInterfaces();
    });
  }

  const scanWireGuardUsbButton = document.getElementById("scanWireGuardUsb");
  if (scanWireGuardUsbButton && !scanWireGuardUsbButton.dataset.bound) {
    scanWireGuardUsbButton.dataset.bound = "1";
    scanWireGuardUsbButton.addEventListener("click", () => refreshWireGuardUSB());
  }

  const refreshOTAButton = document.getElementById("refreshOTAStatus");
  if (refreshOTAButton && !refreshOTAButton.dataset.bound) {
    refreshOTAButton.dataset.bound = "1";
    refreshOTAButton.addEventListener("click", () => refreshOTA());
  }

  const refreshOTAReleasesButton = document.getElementById("refreshOTAReleases");
  if (refreshOTAReleasesButton && !refreshOTAReleasesButton.dataset.bound) {
    refreshOTAReleasesButton.dataset.bound = "1";
    refreshOTAReleasesButton.addEventListener("click", () => refreshOTACatalog());
  }

  const refreshOTACatalogButton = document.getElementById("refreshOTACatalog");
  if (refreshOTACatalogButton && !refreshOTACatalogButton.dataset.bound) {
    refreshOTACatalogButton.dataset.bound = "1";
    refreshOTACatalogButton.addEventListener("click", () => refreshOTACatalog());
  }

  const runOTACheckButton = document.getElementById("runOTACheck");
  if (runOTACheckButton && !runOTACheckButton.dataset.bound) {
    runOTACheckButton.dataset.bound = "1";
    runOTACheckButton.addEventListener("click", () => startOTACheck());
  }

  const otaReleaseTag = document.getElementById("otaReleaseTag");
  if (otaReleaseTag && !otaReleaseTag.dataset.bound) {
    otaReleaseTag.dataset.bound = "1";
    otaReleaseTag.addEventListener("change", () => {
      appState.selectedOtaTag = otaReleaseTag.value;
    });
  }

  const runOTAUpdateButton = document.getElementById("runOTAUpdate");
  if (runOTAUpdateButton && !runOTAUpdateButton.dataset.bound) {
    runOTAUpdateButton.dataset.bound = "1";
    runOTAUpdateButton.addEventListener("click", () => {
      const selectedTag = appState.selectedOtaTag || document.getElementById("otaReleaseTag")?.value || "";
      openOTAConfirmModal(selectedTag);
    });
  }

  const rollbackButton = document.getElementById("runOTARollback");
  if (rollbackButton && !rollbackButton.dataset.bound) {
    rollbackButton.dataset.bound = "1";
    rollbackButton.addEventListener("click", () => triggerOTARollback());
  }

  const scanOTAUsbButton = document.getElementById("scanOTAUsb");
  if (scanOTAUsbButton && !scanOTAUsbButton.dataset.bound) {
    scanOTAUsbButton.dataset.bound = "1";
    scanOTAUsbButton.addEventListener("click", () => refreshOTAUSB());
  }

  const refreshOTAUsbEventButton = document.getElementById("refreshOTAUsbEvent");
  if (refreshOTAUsbEventButton && !refreshOTAUsbEventButton.dataset.bound) {
    refreshOTAUsbEventButton.dataset.bound = "1";
    refreshOTAUsbEventButton.addEventListener("click", () => refreshOTAUSBEvent());
  }

  const otaBannerUpdateNow = document.getElementById("otaBannerUpdateNow");
  if (otaBannerUpdateNow && !otaBannerUpdateNow.dataset.bound) {
    otaBannerUpdateNow.dataset.bound = "1";
    otaBannerUpdateNow.addEventListener("click", () => {
      const selectedTag = appState.otaCheck?.latestTag || appState.selectedOtaTag || "";
      openOTAConfirmModal(selectedTag);
    });
  }

  const otaBannerDismiss = document.getElementById("otaBannerDismiss");
  if (otaBannerDismiss && !otaBannerDismiss.dataset.bound) {
    otaBannerDismiss.dataset.bound = "1";
    otaBannerDismiss.addEventListener("click", () => {
      const banner = document.getElementById("otaUpdateBanner");
      if (banner) {
        banner.setAttribute("aria-hidden", "true");
      }
    });
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

  const networkMode = document.getElementById("networkMode");
  if (networkMode && !networkMode.dataset.bound) {
    networkMode.dataset.bound = "1";
    networkMode.addEventListener("change", updateNetworkModeUI);
  }

  updateNetworkModeUI();
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

  syncInterfaceFields();
  updateNetworkModeUI();
}

function updateNetworkModeUI() {
  const modeElement = document.getElementById("networkMode");
  const staticFields = ["networkAddress", "networkPrefix", "networkGateway", "networkDNS"];
  if (!modeElement) {
    return;
  }

  const isStatic = (modeElement.value || "dhcp") === "static";
  staticFields.forEach((id) => {
    const field = document.getElementById(id);
    if (!field) {
      return;
    }
    field.disabled = !isStatic;
    field.setAttribute("aria-disabled", isStatic ? "false" : "true");
  });
}

function buildNetworkStatusSection() {
  const details = appState.networkInterfaces.details || [];
  if (details.length === 0) {
    return `<section class="tab-card"><h3>Active interfaces</h3><div class="status-card compact">No interface data — click Refresh adapters.</div></section>`;
  }
  const rows = details.map((iface) => {
    const state = iface.operstate || "unknown";
    const addrs = (iface.addresses || []).join(", ") || "—";
    const ssidPart = iface.ssid ? ` · ${escapeHtml(iface.ssid)}` : "";
    const typePart = iface.isWireless ? " (WiFi)" : "";
    return `
      <div class="metric">
        <div class="label">${escapeHtml(iface.name)}${escapeHtml(typePart)}</div>
        <div class="value">${escapeHtml(state)} · ${escapeHtml(addrs)}${ssidPart}</div>
      </div>
    `;
  }).join("");
  return `
    <section class="tab-card">
      <h3>Active interfaces</h3>
      <div class="metrics-grid">${rows}</div>
    </section>
  `;
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
        <label>Interface
          <select id="networkInterface">${networkInterfaceOptionsMarkup()}</select>
        </label>
        <label>Address<input id="networkAddress" type="text" placeholder="192.168.1.10" /></label>
        <label>Prefix<input id="networkPrefix" type="number" placeholder="24" /></label>
        <label>Gateway<input id="networkGateway" type="text" placeholder="192.168.1.1" /></label>
        <label>DNS<input id="networkDNS" type="text" placeholder="1.1.1.1 8.8.8.8" /></label>
      </div>
      <div class="actions tight">
        <button type="button" id="refreshNetworkInterfaces" class="secondary">Refresh adapters</button>
        <button type="button" id="refreshNetworkSettings" class="secondary">Reload</button>
        <button type="button" id="applyNetworkSettings">Apply wired settings</button>
      </div>
      <div class="status-card compact" id="networkState">network not loaded</div>
    </section>
  `;
}

function buildTerminalPanel(tab) {
  const terminal = appState.terminal || { running: false, ready: false, url: "http://127.0.0.1:7681/", message: "Console is stopped." };
  const status = terminal.running
    ? (terminal.ready ? `Console running at ${terminal.url}` : "Console starting...")
    : (terminal.message || "Console is stopped.");
  const iframe = terminal.running && terminal.ready
    ? `<iframe class="terminal-frame" src="${escapeHtml(terminal.url)}" title="Embedded terminal console"></iframe>`
    : `<div class="status-card compact terminal-empty">Console unavailable right now (for example, while RDP is active).</div>`;

  return `
    <section class="tab-panel terminal-panel ${tab.id === appState.activeTab ? "active" : ""}" data-panel-id="${tab.id}" role="tabpanel">
      <div class="status-card">${escapeHtml(tab.title)}</div>
      <section class="tab-card terminal-card">
        <div class="terminal-toolbar">
          <div class="terminal-status">${escapeHtml(status)}</div>
        </div>
        ${iframe}
      </section>
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
  document.getElementById("scanWifi")?.addEventListener("click", () => scanWifi(true));
  document.getElementById("loadNetwork")?.addEventListener("click", refreshNetwork);
  document.getElementById("saveNetwork")?.addEventListener("click", saveNetwork);
}

function bindModalActions() {
  document.getElementById("closeSettings")?.addEventListener("click", closeSettings);
  document.getElementById("reloadSettings")?.addEventListener("click", async () => {
    await refreshConfig();
    await refreshNetwork();
    await refreshNetworkInterfaces();
    await refreshWireGuardUSB();
    renderSettingsPanels();
  });
  document.getElementById("saveSettings")?.addEventListener("click", saveSettings);

  document.getElementById("certTrustClose")?.addEventListener("click", closeCertTrustModal);
  document.getElementById("certTrustCancel")?.addEventListener("click", closeCertTrustModal);
  document.getElementById("certTrustTofu")?.addEventListener("click", async () => {
    try {
      await persistCertPolicy("tofu");
      closeCertTrustModal();
      await connectSession("tofu", true);
    } catch (err) {
      updateText("settingsNote", `certificate trust error: ${err.message}`);
    }
  });
  document.getElementById("certTrustIgnore")?.addEventListener("click", async () => {
    try {
      await persistCertPolicy("ignore");
      closeCertTrustModal();
      await connectSession("ignore");
    } catch (err) {
      updateText("settingsNote", `certificate ignore error: ${err.message}`);
    }
  });

  document.getElementById("wifiScanClose")?.addEventListener("click", closeWifiScanModal);
  document.getElementById("wifiScanRefresh")?.addEventListener("click", () => scanWifi(true));
  document.getElementById("wifiScanInterface")?.addEventListener("change", () => scanWifi(false));

  document.getElementById("wifiConnectClose")?.addEventListener("click", closeWifiConnectModal);
  document.getElementById("wifiConnectCancel")?.addEventListener("click", closeWifiConnectModal);
  document.getElementById("wifiConnectSubmit")?.addEventListener("click", async () => {
    const payload = {
      interface: document.getElementById("wifiConnectInterface")?.value.trim() || appState.networkInterfaces.defaultWireless || "",
      ssid: document.getElementById("wifiConnectSSID")?.value.trim() || "",
      password: document.getElementById("wifiConnectPassword")?.value || "",
      hidden: Boolean(document.getElementById("wifiConnectHidden")?.checked),
    };
    await connectWifi(payload);
  });

  document.getElementById("otaConfirmClose")?.addEventListener("click", closeOTAConfirmModal);
  document.getElementById("otaConfirmCancel")?.addEventListener("click", closeOTAConfirmModal);
  document.getElementById("otaConfirmInstall")?.addEventListener("click", async () => {
    const tag = appState.otaConfirm.targetTag || appState.selectedOtaTag || "";
    closeOTAConfirmModal();
    await triggerOTAUpdate(tag);
  });
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
  // Refresh live data whenever the modal opens
  Promise.all([refreshNetwork(), refreshNetworkInterfaces(), refreshTTYDStatus()]).catch(() => {});
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
    const snapshot = appState.session;
    if (snapshot.state !== "error") {
      appState.certPrompt.dismissedSignature = "";
    }
    let display = `session: ${snapshot.state || "unknown"}`;
    if (snapshot.exitCode !== undefined && snapshot.exitCode !== null && snapshot.exitCode !== 0) {
      display += ` (exit ${snapshot.exitCode})`;
    }
    if (snapshot.lastOutput) {
      display += `\n\n${snapshot.lastOutput}`;
    }
    updateText("sessionState", display);
    maybePromptCertificateTrust(snapshot);
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

async function refreshWireGuardUSB() {
  try {
    const payload = await api("/api/v1/wireguard/usb");
    appState.wireguardUSBConfigs = payload.configs || [];
    renderWireGuardUSBRows("wireguardUsbList");
  } catch (err) {
    appState.wireguardUSBConfigs = [];
    updateText("wireguardUsbList", `wireguard scan error: ${err.message}`);
  }
}

async function refreshNetworkInterfaces() {
  try {
    const payload = await api("/api/v1/network/interfaces");
    appState.networkInterfaces = {
      interfaces: payload.interfaces || [],
      wireless: payload.wireless || [],
      hasWireless: Boolean(payload.hasWireless),
      defaultInterface: payload.defaultInterface || "",
      defaultWireless: payload.defaultWireless || "",
      details: payload.details || [],
    };
    syncInterfaceFields();
    renderSettingsPanels();
    // renderSettingsPanels resets #networkState to the placeholder; restore it if data is already loaded
    if (appState.network) {
      syncNetworkFields();
      updateText("networkState", JSON.stringify(appState.network, null, 2));
    }
    if (isModalOpen("wifiScanModal")) {
      renderWifiRows("modalWifiList");
    }
  } catch (err) {
    updateText("settingsNote", `interface refresh error: ${err.message}`);
  }
}

async function refreshStatus() {
  try {
    appState.status = await api("/api/v1/status");
    renderPills();
    renderMetrics();
    renderCornerClock();
  } catch (err) {
    updateText("configState", `status error: ${err.message}`);
    renderCornerClock();
  }
}

async function refreshOTA() {
  try {
    appState.ota = await api("/api/v1/ota");
    renderSettingsPanels();
  } catch (err) {
    appState.ota = null;
    updateText("otaState", `ota error: ${err.message}`);
  }
}

function renderOTABanner() {
  const banner = document.getElementById("otaUpdateBanner");
  if (!banner) {
    return;
  }

  const check = appState.otaCheck;
  if (!check || !check.available || !check.latestTag) {
    banner.setAttribute("aria-hidden", "true");
    return;
  }

  updateText("otaBannerText", `Update available: ${check.latestTag}`);
  banner.setAttribute("aria-hidden", "false");
}

async function refreshOTACatalog() {
  try {
    const payload = await api("/api/v1/ota/catalog");
    appState.otaCatalog = payload.entries || [];
    appState.otaReleases = appState.otaCatalog.map((entry) => ({
      tag: entry.tag,
      name: entry.name,
      publishedAt: entry.publishedAt,
      prerelease: entry.prerelease,
    }));
    const knownTags = new Set(appState.otaCatalog.map((entry) => entry.tag));
    if (!appState.selectedOtaTag || !knownTags.has(appState.selectedOtaTag)) {
      appState.selectedOtaTag = appState.otaCatalog[0]?.tag || "";
    }
    renderSettingsPanels();
  } catch (err) {
    appState.otaCatalog = [];
    appState.selectedOtaTag = "";
    updateText("settingsNote", `catalog refresh error: ${err.message}`);
    renderSettingsPanels();
  }
}

async function refreshOTACheck() {
  try {
    appState.otaCheck = await api("/api/v1/ota/check");
    renderOTABanner();
    return appState.otaCheck;
  } catch (err) {
    updateText("settingsNote", `check status error: ${err.message}`);
    return null;
  }
}

async function refreshOTAUSB() {
  try {
    const payload = await api("/api/v1/ota/usb");
    appState.otaUsbImages = payload.images || [];
    renderOTAUSBRows("otaUsbList");
    updateText("settingsNote", `Detected ${appState.otaUsbImages.length} USB OTA image(s).`);
  } catch (err) {
    appState.otaUsbImages = [];
    renderOTAUSBRows("otaUsbList");
    updateText("settingsNote", `usb scan error: ${err.message}`);
  }
}

async function refreshOTAUSBEvent() {
  try {
    const payload = await api("/api/v1/ota/usb/event");
    appState.otaUsbEvent = payload;
    if (payload?.detected) {
      updateText("otaUsbEventState", `USB image detected: ${payload.filename || payload.path}`);
    } else {
      updateText("otaUsbEventState", "No USB OTA event detected.");
    }
  } catch (err) {
    updateText("otaUsbEventState", `usb event error: ${err.message}`);
  }
}

function classifyOTAError(message) {
  const m = String(message).toLowerCase();
  if (m.includes("signature verification failed") || m.includes("manifest signature")) {
    return "Manifest signature verification failed. The release may have been tampered with or the wrong signing key is installed.";
  }
  if (m.includes("sha256 mismatch")) {
    return "Image integrity check failed (SHA256 mismatch). The download may be corrupt or tampered.";
  }
  if (m.includes("ota public key not readable") || m.includes("ota-signing-public.pem")) {
    return "OTA signing key is missing or unreadable on this device. Contact your administrator.";
  }
  if (m.includes("manifest.json.sig") && m.includes("required")) {
    return "No manifest signature found on this USB drive. A signed manifest is required.";
  }
  if (m.includes("manifest.json") && m.includes("required")) {
    return "No manifest found on this USB drive. A manifest.json is required.";
  }
  return message;
}

async function importOTAUSB(path) {
  try {
    updateText("settingsNote", "Importing OTA image from USB...");
    const payload = await api("/api/v1/ota/usb/import", "POST", { path });
    updateText("settingsNote", payload?.message || "USB OTA import staged.");
    await refreshOTA();
    await refreshOTAUSB();
  } catch (err) {
    updateText("settingsNote", `USB OTA import error: ${classifyOTAError(err.message)}`);
  }
}

async function startOTACheck() {
  try {
    updateText("settingsNote", "Checking for updates in background...");
    await api("/api/v1/ota/check", "POST", {});

    for (let attempt = 0; attempt < 25; attempt += 1) {
      const state = await refreshOTACheck();
      if (!state || !state.running) {
        break;
      }
      await new Promise((resolve) => setTimeout(resolve, 400));
    }

    await Promise.all([refreshOTACatalog(), refreshOTA()]);
    if (appState.otaCheck?.error) {
      updateText("settingsNote", `update check failed: ${appState.otaCheck.error}`);
    } else if (appState.otaCheck?.available) {
      const latest = appState.otaCheck?.latestTag || "new release";
      updateText("settingsNote", `${latest} is available.`);
      if (appState.otaCheck.latestTag) {
        appState.selectedOtaTag = appState.otaCheck.latestTag;
        renderSettingsPanels();
      }
    } else {
      updateText("settingsNote", "No new updates available.");
    }
  } catch (err) {
    updateText("settingsNote", `check error: ${err.message}`);
  }
}

async function refreshOTAReleases() {
  try {
    await refreshOTACatalog();
  } catch (err) {
    updateText("settingsNote", `release refresh error: ${err.message}`);
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

async function refreshTTYDStatus() {
  try {
    const payload = await api("/api/v1/terminal/ttyd");
    appState.terminal = {
      running: Boolean(payload.running),
      ready: Boolean(payload.ready),
      url: payload.url || "http://127.0.0.1:7681/",
      message: payload.message || (payload.running ? "Console running." : "Console is stopped."),
    };
    renderSettingsPanels();
  } catch (err) {
    appState.terminal = {
      running: false,
      ready: false,
      url: "http://127.0.0.1:7681/",
      message: `console error: ${err.message}`,
    };
    renderSettingsPanels();
  }
}

async function refreshAll() {
  await Promise.all([refreshHealth(), refreshSession(), refreshConfig(), refreshNetwork(), refreshStatus(), refreshNetworkInterfaces(), refreshWireGuardUSB(), refreshOTA(), refreshOTACatalog(), refreshOTACheck(), refreshOTAUSB(), refreshOTAUSBEvent(), refreshTTYDStatus()]);
  await refreshWifi();
}

async function triggerOTAUpdate(targetTag = "") {
  const selectedTag = String(targetTag || appState.selectedOtaTag || document.getElementById("otaReleaseTag")?.value || "").trim();
  if (!selectedTag) {
    updateText("settingsNote", "Select a release first.");
    return;
  }

  try {
    updateText("settingsNote", `Starting OTA update to ${selectedTag}...`);
    const payload = await api("/api/v1/ota/update", "POST", { tag: selectedTag });
    if (payload?.status) {
      appState.ota = payload.status;
      renderSettingsPanels();
    }
    updateText("settingsNote", payload?.message || `Update to ${selectedTag} started.`);
  } catch (err) {
    updateText("settingsNote", `Update error: ${classifyOTAError(err.message)}`);
  }
}

async function triggerOTARollback() {
  try {
    updateText("settingsNote", "Triggering OTA rollback...");
    const payload = await api("/api/v1/ota/rollback", "POST", {});
    if (payload?.status) {
      appState.ota = payload.status;
      renderSettingsPanels();
    }
    updateText("settingsNote", payload?.message || "Rollback triggered.");
  } catch (err) {
    updateText("settingsNote", `rollback error: ${err.message}`);
  }
}

async function connectSession(certPolicyOverride = "", resetCert = false) {
  const certPolicy = normalizeCertPolicy(certPolicyOverride || valueForKey("cert_policy") || "tofu");
  const payload = {
    server: document.getElementById("server")?.value.trim(),
    username: document.getElementById("username")?.value.trim(),
    password: document.getElementById("password")?.value,
    domain: document.getElementById("domain")?.value.trim(),
    certPolicy,
    resetCert: Boolean(resetCert),
  };

  try {
    await api("/api/v1/session/connect", "POST", payload);
    closeCertTrustModal();
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
    if (!key || !settingsFieldKeys.includes(key) || NETWORK_CONFIG_KEYS.has(key)) {
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
    const changedOTAPolicy = Object.keys(values).some((key) => OTA_POLICY_KEYS.has(key));
    if (changedOTAPolicy) {
      await api("/api/v1/ota/apply-policy", "POST", {});
    }
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

  if (payload.mode === "static" && !payload.interface) {
    updateText("networkState", "network save error: static mode requires interface");
    updateText("settingsNote", "Select a specific interface for static IPv4.");
    return;
  }

  try {
    const saved = await api("/api/v1/network", "POST", payload);
    appState.network = saved;
    syncNetworkFields();
    const applyMessage = saved?.applyMessage || "Wired network saved.";
    updateText("networkState", `${applyMessage}\n${JSON.stringify(saved, null, 2)}`);
    updateText("settingsNote", applyMessage);
  } catch (err) {
    updateText("networkState", `network save error: ${err.message}`);
  }
}

async function scanWifi(openModal = false) {
  try {
    const interfaceValue = document.getElementById("wifiScanInterface")?.value.trim()
      || document.getElementById("wifiInterface")?.value.trim()
      || appState.networkInterfaces.defaultWireless
      || "";
    if (!interfaceValue && appState.networkInterfaces.hasWireless) {
      updateText("settingsNote", "No wireless adapter selected.");
      updateText("wifiScanState", "No wireless adapter selected.");
      return;
    }
    const query = interfaceValue ? `?interface=${encodeURIComponent(interfaceValue)}` : "";
    const payload = await api(`/api/v1/wifi/scan${query}`);
    appState.wifiNetworks = payload.networks || [];
    buildMainNetworkRows();
    renderWifiRows("settingsWifiList");
    renderWifiRows("modalWifiList");
    updateText("wifiScanState", `Found ${appState.wifiNetworks.length} WiFi network(s) on ${interfaceValue || "auto"}.`);
    updateText("settingsNote", `Found ${appState.wifiNetworks.length} WiFi network(s).`);
    if (openModal) {
      openWifiScanModal();
    }
  } catch (err) {
    appState.wifiNetworks = [];
    const message = `wifi scan error: ${err.message}`;
    updateText("wifiList", message);
    updateText("settingsWifiList", message);
    updateText("wifiScanState", message);
  }
}

async function importWireGuardUSB(path) {
  try {
    updateText("settingsNote", "Importing WireGuard config from USB...");
    const payload = await api("/api/v1/wireguard/import", "POST", { path });
    updateText("settingsNote", `Imported ${payload.interface || payload.path}.`);
    await refreshWireGuardUSB();
    await refreshStatus();
  } catch (err) {
    updateText("settingsNote", `wireguard import error: ${err.message}`);
  }
}

async function connectWifi(overridePayload = null) {
  const selectedInterface = document.getElementById("wifiInterface")?.value.trim() || appState.networkInterfaces.defaultWireless || "";
  const payload = overridePayload || {
    interface: selectedInterface,
    ssid: document.getElementById("wifiSSID")?.value.trim() || "",
    password: document.getElementById("wifiPassword")?.value || "",
    hidden: Boolean(document.getElementById("wifiHidden")?.checked),
  };

  if (!payload.interface) {
    updateText("settingsNote", "Select a wireless adapter before connecting.");
    return;
  }

  try {
    await api("/api/v1/wifi/connect", "POST", payload);
    closeWifiConnectModal();
    updateText("settingsNote", `Connecting to ${payload.ssid || "WiFi"}.`);
    updateText("wifiConnectState", `Connecting to ${payload.ssid || "WiFi"}.`);
    await refreshNetwork();
    await refreshNetworkInterfaces();
    await refreshStatus();
  } catch (err) {
    updateText("settingsNote", `wifi connect error: ${err.message}`);
    updateText("wifiConnectState", `wifi connect error: ${err.message}`);
  }
}

function isEditableTarget(target) {
  if (!(target instanceof Element)) {
    return false;
  }
  return Boolean(target.closest("input, textarea, select, [contenteditable=''], [contenteditable='true']"));
}

function shouldBlockGlobalShortcut(event) {
  if (isEditableTarget(event.target)) {
    return false;
  }

  const key = String(event.key || "").toLowerCase();
  const hasPrimaryModifier = event.ctrlKey || event.metaKey;

  if (event.key === "F12") {
    return true;
  }

  if (hasPrimaryModifier && event.shiftKey && (key === "i" || key === "j" || key === "c")) {
    return true;
  }

  if (hasPrimaryModifier && (key === "u" || key === "a" || key === "c" || key === "x" || key === "v")) {
    return true;
  }

  return false;
}

function wireGlobalShortcuts() {
  window.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeOTAConfirmModal();
      closeWifiConnectModal();
      closeWifiScanModal();
      closeCertTrustModal();
      closeSettings();
    }

    if (shouldBlockGlobalShortcut(event)) {
      event.preventDefault();
      event.stopPropagation();
    }
  });

  document.addEventListener("contextmenu", (event) => {
    if (!isEditableTarget(event.target)) {
      event.preventDefault();
    }
  }, true);

  document.addEventListener("selectstart", (event) => {
    if (!isEditableTarget(event.target)) {
      event.preventDefault();
    }
  }, true);

  document.addEventListener("dragstart", (event) => {
    if (!isEditableTarget(event.target)) {
      event.preventDefault();
    }
  }, true);
}

function initStaticPanels() {
  const connectionTab = SETTINGS_TABS.find((tab) => tab.id === "connection") || { cards: [] };
  const deviceTab = SETTINGS_TABS.find((tab) => tab.id === "device") || { cards: [] };
  const statusTab = SETTINGS_TABS.find((tab) => tab.id === "status") || { cards: [] };
  const supportTab = SETTINGS_TABS.find((tab) => tab.id === "support") || { cards: [] };

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
              ${connectionTab.cards.map((card) => buildCard(card)).join("")}
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
              ${buildNetworkStatusSection()}
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
            ${deviceTab.cards.map((card) => buildCard(card)).join("")}
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="status" role="tabpanel">
          <div class="status-card">Overlay toggles and indicators used by the desktop shell.</div>
          <div class="tab-panel-grid">
            ${statusTab.cards.map((card) => buildCard(card)).join("")}
          </div>
        </section>
      `,
      `
        <section class="tab-panel" data-panel-id="support" role="tabpanel">
          <div class="status-card">Recovery, helpdesk, and deployment details.</div>
          <div class="tab-panel-grid">
            ${supportTab.cards.map((card) => buildCard(card)).join("")}
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
  renderCornerClock();

  await refreshAll();
  await scanWifi(false);
  renderSettingsPanels();
  setInterval(renderCornerClock, 1000);
  setInterval(refreshSession, 5000);
  setInterval(refreshStatus, 7000);
  setInterval(() => {
    if (isModalOpen("settingsModal") || isModalOpen("wifiScanModal") || isModalOpen("wifiConnectModal")) {
      refreshNetworkInterfaces().catch(() => {});
      refreshNetwork().catch(() => {});
      refreshOTAUSBEvent().catch(() => {});
    }
  }, 8000);
  setInterval(refreshHealth, 15000);
});
