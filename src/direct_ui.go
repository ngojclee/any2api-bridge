package main

import (
	"fmt"
	"html"
	"net/url"
)

type directConsoleData struct {
	Version     string                `json:"version"`
	Locked      bool                  `json:"locked"`
	Diagnostics providerDiagnostics   `json:"diagnostics"`
	Direct      directManagementState `json:"direct"`
	Usage       usagePageData         `json:"usage"`
}

func directAccountConsolePage(diagnostics providerDiagnostics, query url.Values) string {
	settings := currentPluginSettings()
	data := directConsoleData{
		Version:     pluginVersion,
		Locked:      false,
		Diagnostics: diagnostics,
		Direct:      directAccountsManagementState(settings),
		Usage:       usageDashboardData(diagnostics, normalizeUsageFilter(query)),
	}
	return directConsoleHTML(data)
}

func directConsoleHTML(data directConsoleData) string {
	seed := jsonForScript(data)
	status := data.Diagnostics.ReplacementMode
	if status == "" {
		status = "legacy-mirror"
	}
	usageMain := renderUsageMainHTML(data.Usage, "/v0/resource/plugins/"+pluginID+"/direct")
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Any2Api Bridge Direct Accounts</title>
<style>
:root{--bg:#f7f5ef;--panel:#fffdfa;--surface:#f0ede5;--inset:#f8f6f1;--ink:#282521;--ink-2:#69635b;--ink-3:#9b948b;--line:#dfdacf;--line-2:#cfc8bb;--accent:#2563eb;--success:#0f766e;--success-bg:#ccfbf1;--warn:#9a5a00;--warn-bg:#fff0bf;--error:#b44232;--error-bg:#fbe3df;--radius:8px;--shadow:0 1px 2px #00000014}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",sans-serif;font-size:14px;line-height:1.45}button,input,select{font:inherit}.shell{display:grid;grid-template-columns:250px minmax(0,1fr);min-height:100vh}.sidebar{background:#fffdf8;border-right:1px solid var(--line);padding:18px 14px;position:sticky;top:0;height:100vh;overflow:auto}.brand{margin-bottom:16px}.brand h1{font-size:19px;line-height:1.2;margin:0 0 4px}.muted{color:var(--ink-2);font-size:12px}.nav-group{margin-top:14px}.nav-title{font-size:11px;font-weight:800;text-transform:uppercase;letter-spacing:.04em;color:var(--ink-3);margin-bottom:8px}.nav-item{display:block;width:100%%;border:0;border-radius:7px;background:transparent;color:var(--ink-2);padding:8px 10px;margin:2px 0;text-align:left;cursor:pointer}.nav-item:hover{background:var(--surface);color:var(--ink)}.nav-item.active{background:var(--accent);color:#fff}.content{padding:22px;min-width:0}.top{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:14px}.top h2{font-size:20px;margin:0 0 4px}.card{background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);box-shadow:var(--shadow);padding:14px;margin-bottom:12px}.section-title{font-size:11px;font-weight:800;text-transform:uppercase;letter-spacing:.04em;color:var(--ink-3);margin-bottom:10px}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.metric{background:var(--inset);border:1px solid var(--line);border-radius:6px;padding:10px}.metric span{display:block;color:var(--ink-3);font-size:11px;font-weight:750;text-transform:uppercase;letter-spacing:.04em}.metric strong{display:block;margin-top:3px;font-size:18px}.pill{display:inline-flex;align-items:center;border-radius:999px;border:1px solid var(--line-2);background:var(--surface);padding:4px 9px;font-size:12px;font-weight:700;color:var(--ink-2)}.pill.ok{background:var(--success-bg);border-color:#5eead4;color:var(--success)}.pill.warn{background:var(--warn-bg);border-color:#f6d365;color:var(--warn)}.pill.err{background:var(--error-bg);border-color:#f0a79d;color:var(--error)}
.actions{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.btn{border:1px solid var(--line-2);background:#fff;color:var(--ink);border-radius:6px;height:34px;padding:0 11px;font-weight:650;font-size:13px;cursor:pointer;text-decoration:none;display:inline-flex;align-items:center;justify-content:center}.btn.primary{background:var(--accent);border-color:var(--accent);color:#fff}.btn.danger{color:var(--error);border-color:#f0a79d}.btn:disabled{opacity:.55;cursor:default}.btn:hover{filter:brightness(.985)}.btn:disabled{opacity:.5;cursor:not-allowed}.table-wrap{overflow:auto;border:1px solid var(--line);border-radius:8px}.table{width:100%%;border-collapse:collapse;background:#fff;font-size:12.5px}.table th,.table td{padding:9px 10px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}.table th{background:var(--inset);color:var(--ink-2);font-size:11px;text-transform:uppercase;letter-spacing:.04em}.table tr:last-child td{border-bottom:0}.mono{font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace}.field{margin-bottom:12px}.field label{display:block;font-size:12px;color:var(--ink-2);font-weight:650;margin-bottom:5px}.field input,.field select{width:100%%;min-height:36px;border:1px solid var(--line-2);background:#fff;border-radius:8px;color:var(--ink);padding:8px 10px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.notice{position:sticky;top:0;z-index:50;background:var(--warn-bg);border:1px solid #f6d365;color:var(--warn);border-radius:6px;padding:10px;margin-bottom:10px;box-shadow:0 2px 8px #00000012}.notice.ok{background:var(--success-bg);border-color:#5eead4;color:var(--success)}.notice.err{background:var(--error-bg);border-color:#f0a79d;color:var(--error)}.result{white-space:pre-wrap;word-break:break-word;border-radius:6px;padding:10px;background:#22221f;color:#eceae5;font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace;font-size:12px;max-height:280px;overflow:auto}.chips{display:flex;gap:7px;flex-wrap:wrap}.chip{background:var(--surface);border:1px solid var(--line-2);border-radius:999px;padding:5px 9px;font-size:12px;font-weight:650}.log{max-height:220px;overflow:auto;background:#22221f;color:#eceae5;border-radius:6px;padding:10px;font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace;font-size:12px}.event{padding:3px 0}.tabs{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px}.tab{border:1px solid var(--line-2);background:#fff;border-radius:7px;padding:7px 10px;color:var(--ink-2);cursor:pointer}.tab.active{background:var(--accent);border-color:var(--accent);color:#fff}.provider-tabs{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px}.provider-tab{border:1px solid var(--line-2);background:#fff;border-radius:7px;padding:8px 12px;font-weight:700;color:var(--ink-2);cursor:pointer}.provider-tab.active{background:var(--accent);border-color:var(--accent);color:#fff}.hidden{display:none!important}.split{display:grid;grid-template-columns:minmax(0,1.3fr) minmax(280px,.7fr);gap:12px}.empty{border:1px dashed var(--line-2);border-radius:8px;padding:14px;color:var(--ink-3);background:var(--inset)}
@media(max-width:960px){.shell{grid-template-columns:1fr}.sidebar{position:relative;height:auto;border-right:0;border-bottom:1px solid var(--line)}.content{padding:16px}.grid,.form-grid,.split{grid-template-columns:1fr}}
</style>
<style>
/* Dense-but-readable console polish. The vendor usage renderer emits its own
   class names, so keep these rules local to the direct console page. */
.grid{grid-template-columns:repeat(5,minmax(0,1fr))}
.provider-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}
.provider-card{display:grid;gap:14px;padding:16px;border:1px solid var(--line);border-radius:10px;background:linear-gradient(180deg,#fff,#fcfbf8);box-shadow:var(--shadow)}
.provider-card-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}
.provider-card h3{margin:0;font-size:16px}
.provider-stats{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px}
.provider-stat{padding:10px;border:1px solid var(--line);border-radius:8px;background:var(--inset)}
.provider-stat span{display:block;color:var(--ink-3);font-size:10px;font-weight:800;text-transform:uppercase;letter-spacing:.04em}
.provider-stat strong{display:block;margin-top:3px;font-size:16px}
.usage-overview .card-head,.usage-panel .card-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:12px}
.usage-metrics{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:8px;margin-top:12px}
.usage-metric{padding:10px;border:1px solid var(--line);border-radius:8px;background:var(--inset)}
.usage-metric span{display:block;color:var(--ink-3);font-size:10px;font-weight:800;text-transform:uppercase;letter-spacing:.04em}
.usage-metric strong{display:block;margin-top:3px;font-size:15px;word-break:break-word}
.usage-analysis-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:14px}
.usage-panel{margin:0}
.metric strong.status-on{color:var(--success)}
.metric strong.status-off{color:var(--error)}
.row-actions{display:flex;align-items:center;gap:8px;flex-wrap:wrap}
.table td .btn{margin:2px 6px 2px 0}
.field select{appearance:none;background-image:linear-gradient(45deg,transparent 50%%,var(--ink-2) 50%%),linear-gradient(135deg,var(--ink-2) 50%%,transparent 50%%);background-position:calc(100%% - 16px) 15px,calc(100%% - 11px) 15px;background-size:5px 5px,5px 5px;background-repeat:no-repeat;padding-right:34px}
.custom-select{position:relative;width:100%%}
.custom-select-trigger{display:flex;align-items:center;justify-content:space-between;gap:8px;width:100%%;min-height:36px;border:1px solid var(--line-2);background:#fff;border-radius:8px;padding:8px 12px;font:inherit;font-size:13px;color:var(--ink);cursor:pointer;text-align:left}
.custom-select.open .custom-select-trigger{border-color:var(--accent);box-shadow:0 0 0 3px #2563eb1a}
.custom-select-value{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.custom-select-arrow{width:8px;height:8px;flex:0 0 auto;border-right:1.5px solid currentColor;border-bottom:1.5px solid currentColor;transform:rotate(45deg);opacity:.5;margin-top:-3px;transition:transform .18s ease}
.custom-select.open .custom-select-arrow{transform:rotate(-135deg);margin-top:3px}
.custom-select-panel{position:absolute;top:calc(100%% + 4px);left:0;right:0;z-index:60;display:none;max-height:280px;overflow:auto;padding:6px;border:1px solid var(--line-2);border-radius:10px;background:var(--panel);box-shadow:0 8px 24px #00000018}
.custom-select.open .custom-select-panel{display:block}
.custom-select-option{display:block;width:100%%;border:0;border-radius:6px;background:transparent;padding:8px 10px;font:inherit;font-size:13px;color:var(--ink);cursor:pointer;text-align:left}
.custom-select-option:hover{background:var(--surface)}
.custom-select-option.selected{background:#e8f0fe;color:var(--accent);font-weight:650}
.usage-filters{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin:12px 0}
.usage-filter-field{display:inline-flex;align-items:center;gap:6px;margin:0;font-size:12px;color:var(--ink-2);font-weight:650}
.usage-filter-field span{display:inline;margin:0;font-size:12px;color:var(--ink-2);font-weight:650;text-transform:none;letter-spacing:0}
.usage-filters select{width:auto;min-width:140px;min-height:34px;border:1px solid var(--line-2);background:#fff;border-radius:8px;padding:6px 9px;color:var(--ink);cursor:pointer}
.usage-filters .btn{height:34px;min-height:34px;padding:0 12px;font-size:13px}
.usage-filters.is-loading{opacity:.6;pointer-events:none}
.share-list,.recent-list,.bucket-list{display:grid;gap:8px}
.share-row{padding:9px 10px;border:1px solid var(--line);border-radius:8px;background:var(--inset)}
.share-row-head,.share-meta{display:flex;align-items:baseline;justify-content:space-between;gap:10px}
.share-row-head span,.share-meta{color:var(--ink-2);font-size:11px}
.share-bar{height:6px;margin:7px 0;border-radius:999px;background:#e8e4dc;overflow:hidden}
.share-bar span{display:block;height:100%%;border-radius:inherit;background:var(--accent)}
.recent-row,.bucket-row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:10px;align-items:center;padding:9px 10px;border:1px solid var(--line);border-radius:8px;background:var(--inset)}
.recent-row small,.bucket-row small{display:block;color:var(--ink-2);font-size:11px;margin-top:2px}
.recent-value{text-align:right}
.usage-tag{display:inline-flex;margin-left:5px;padding:1px 6px;border-radius:999px;background:var(--success-bg);color:var(--success);font-size:10px;font-weight:750}
.usage-empty{padding:14px;border:1px dashed var(--line-2);border-radius:8px;color:var(--ink-2);background:var(--inset)}
.management-card{padding:12px 14px}
.management-row{display:grid;grid-template-columns:auto minmax(260px,1fr) auto auto auto;align-items:center;gap:8px}
.management-row input{width:100%%;min-height:36px;border:1px solid var(--line-2);border-radius:8px;padding:8px 10px}
#provider-pane{padding:16px}
#provider-pane .tabs{margin-bottom:16px}
#provider-pane .tab{padding:8px 12px}
#provider-content > .empty{margin:0 0 14px}
#provider-content > .actions{margin:0 0 14px}
#provider-content > .table-wrap{margin-top:0}
#provider-content .table th,#provider-content .table td{padding:11px 12px}
@media(max-width:1200px){.grid,.usage-metrics{grid-template-columns:repeat(3,minmax(0,1fr))}}
@media(max-width:900px){.provider-grid,.usage-analysis-grid{grid-template-columns:1fr}.management-row{grid-template-columns:1fr}.usage-metrics{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:600px){.grid,.usage-metrics,.provider-stats{grid-template-columns:1fr}.usage-filter-field{width:100%%}.usage-filter-field select{width:100%%}}
</style>
</head>
<body style="margin:0;width:100%%;max-width:none">
<div class="shell" style="width:100%%;max-width:none">
<aside class="sidebar">
<div class="brand"><h1>Any2Api Bridge</h1><div class="muted">Direct provider console</div><div class="muted">v%s</div></div>
<div class="nav-group"><div class="nav-title">Overview</div><button class="nav-item" data-provider="global" data-view="overview">Service overview</button></div>
<div class="nav-group"><div class="nav-title">Global</div><button class="nav-item" data-provider="global" data-view="usage">Usage</button><button class="nav-item" data-provider="global" data-view="logs">Logs</button><button class="nav-item" data-provider="global" data-view="settings">Settings</button></div>
</aside>
<main class="content" style="width:100%%;max-width:none;padding:18px 22px;box-sizing:border-box">
<div class="top"><div><h2 id="page-title">Direct accounts</h2><div class="muted">Direct provider console for Antigravity and ChatGPT. Secrets stay write-only; this page only shows configured state.</div></div><div class="actions"><button class="btn primary" id="open-editor" type="button">Open editor</button><span class="pill">%s</span></div></div>
<section class="card management-card"><div class="management-row"><div class="section-title" style="margin:0">Management access</div><input id="mkey" type="password" autocomplete="current-password" placeholder="CPA management key"><button class="btn primary" id="save-mkey" type="button">Use key</button><button class="btn" id="clear-mkey" type="button">Clear</button><span class="muted">Session only</span></div></section>
<div class="provider-tabs"><button class="provider-tab active" type="button" data-provider-tab="overview">Overview</button><button class="provider-tab" type="button" data-provider-tab="agy2api">Antigravity</button><button class="provider-tab" type="button" data-provider-tab="gpt2api">ChatGPT</button></div>
<div id="notice" class="notice hidden"></div>
<section class="card">
<div class="grid">
<div class="metric"><span>Direct mode</span><strong id="direct-mode">-</strong></div><div class="metric"><span>Operator action</span><button class="btn" type="button" data-action="toggle-mode" id="toggle-mode">-</button></div>
<div class="metric"><span>Accounts</span><strong id="account-count">0</strong></div>
<div class="metric"><span>AGY2API</span><strong id="agy-count">0</strong></div>
<div class="metric"><span>GPT2API</span><strong id="gpt-count">0</strong></div>
</div>
</section>
<section id="overview-pane" class="card hidden">
<div class="section-title">Provider overview</div>
<div id="overview-providers" class="provider-grid"></div>
</section>
<section id="provider-pane" class="card">
<div class="tabs">
<button class="tab" data-view="accounts">Accounts</button>
<button class="tab" data-view="models">Models</button>
<button class="tab" data-view="headers">Headers</button>
<button class="tab" data-view="routing">Routing</button>
</div>
<div id="provider-content"></div>
</section>
<section id="usage-pane" class="card hidden"><div class="section-title">Usage</div>%s</section>
<section id="logs-pane" class="card hidden"><div class="section-title">Runtime log</div><div class="log" id="runtime-log"></div></section>
<section id="settings-pane" class="card hidden"><div class="section-title">Settings</div><div id="settings-content"></div></section>
<details class="card"><summary class="section-title" style="cursor:pointer">JSON result</summary><pre class="result" id="result">No management call yet.</pre></details>
</main>
</div>
<script>
const state = %s;
const MGT = '/v0/management/plugins/%s';
const initialUsageFilter = new URLSearchParams(window.location.search);
// Every write action refreshes by reloading the page, so the selected
// provider/view/account has to survive a reload - otherwise pressing a button
// inside the ChatGPT tab snapped straight back to Antigravity.
const UI_STATE_KEY = 'any2apiBridgeUiState';
const PROVIDER_KINDS = ['global', 'agy2api', 'gpt2api'];
const VIEW_KINDS = ['overview', 'accounts', 'models', 'headers', 'routing', 'usage', 'logs', 'settings'];
function readUiState(){ try{ const raw = sessionStorage.getItem(UI_STATE_KEY); const parsed = raw ? JSON.parse(raw) : null; return parsed && typeof parsed === 'object' ? parsed : {}; }catch(e){ return {}; } }
function writeUiState(){ try{ sessionStorage.setItem(UI_STATE_KEY, JSON.stringify({provider:provider, view:view, selectedAccount:selectedAccount, lastProvider:lastProvider})); }catch(e){} }
const savedUi = readUiState();
const usageFilterActive = initialUsageFilter.has('period') || initialUsageFilter.has('bucket') || initialUsageFilter.has('source');
let lastProvider = PROVIDER_KINDS.indexOf(savedUi.lastProvider) > 0 ? savedUi.lastProvider : 'agy2api';
let provider = usageFilterActive ? 'global' : (PROVIDER_KINDS.indexOf(savedUi.provider) >= 0 ? savedUi.provider : lastProvider);
let view = usageFilterActive ? 'usage' : (VIEW_KINDS.indexOf(savedUi.view) >= 0 ? savedUi.view : (provider === 'global' ? 'usage' : 'accounts'));
let selectedAccount = typeof savedUi.selectedAccount === 'string' ? savedUi.selectedAccount : '';
if(provider !== 'global'){ lastProvider = provider; }
const accounts = state.direct.accounts || [];
function esc(value){ return String(value === undefined || value === null ? '' : value).replace(/[&<>"']/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
function el(id){ return document.getElementById(id); }
function notice(text, kind){ const node = el('notice'); node.textContent = text || ''; node.className = 'notice' + (kind ? ' ' + kind : ''); node.classList.toggle('hidden', !text); }
function show(value){ el('result').textContent = typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
function managementKey(){ const node = el('mkey'); return (node && node.value.trim()) || sessionStorage.getItem('agyBridgeManagementKey') || ''; }
function mgmtHeaders(json){ const key = managementKey(); const headers = {}; if(json) headers['Content-Type'] = 'application/json'; if(key){ headers['X-Management-Key'] = key; headers['Authorization'] = 'Bearer ' + key; } return headers; }
async function call(path, body){ if(!managementKey()) throw new Error('missing management key: enter the CPA management key in Management access'); const res = await fetch(MGT + path, {method: body ? 'POST' : 'GET', headers: mgmtHeaders(!!body), body: body ? JSON.stringify(body) : undefined}); const text = await res.text(); let data; try{ data = JSON.parse(text); }catch{ data = text; } if(!res.ok) throw new Error(typeof data === 'string' ? data : JSON.stringify(data)); return data; }
function providerAccounts(kind){ return accounts.filter(a=>a.provider_kind === kind); }
function firstAccount(kind){ const list = providerAccounts(kind); if(!list.length) return ''; if(selectedAccount && list.some(a=>a.account_id === selectedAccount)) return selectedAccount; return list[0].account_id; }
function accountByID(id){ return accounts.find(a=>a.account_id === id) || null; }
function setProvider(kind){ provider = kind; if(kind !== 'global'){ lastProvider = kind; } if(view !== 'accounts' && view !== 'models' && view !== 'headers' && view !== 'routing') view = 'accounts'; selectedAccount = firstAccount(kind); render(); }
function setTopTab(kind){ if(kind === 'overview'){ provider = 'global'; view = 'overview'; } else { setProvider(kind); return; } render(); }
function setView(next){ view = next; render(); }
function modelCount(account){ return (account.models || []).length; }
function render(){ document.querySelectorAll('[data-provider-tab]').forEach(node=>{ const kind = node.dataset.providerTab; const active = kind === 'overview' ? (provider === 'global' && view === 'overview') : kind === provider; node.classList.toggle('active', active); }); document.querySelectorAll('.nav-item').forEach(node=>{ node.classList.toggle('active', node.dataset.provider === provider && node.dataset.view === view || node.dataset.provider === 'global' && node.dataset.view === view && provider === 'global'); }); document.querySelectorAll('.tab').forEach(node=>node.classList.toggle('active', node.dataset.view === view)); el('overview-pane').classList.toggle('hidden', !(provider === 'global' && view === 'overview')); el('provider-pane').classList.toggle('hidden', provider === 'global'); el('usage-pane').classList.toggle('hidden', !(provider === 'global' && view === 'usage')); el('logs-pane').classList.toggle('hidden', !(provider === 'global' && view === 'logs')); el('settings-pane').classList.toggle('hidden', !(provider === 'global' && view === 'settings')); const label = view.charAt(0).toUpperCase() + view.slice(1); el('page-title').textContent = provider === 'global' ? label : (provider === 'agy2api' ? 'Antigravity' : 'ChatGPT') + ' ' + label; renderSummary(); renderOverview(); renderProvider(); renderLogs(); renderSettings(); writeUiState(); }
function renderSummary(){ const agy = providerAccounts('agy2api').length; const gpt = providerAccounts('gpt2api').length; const on = !!state.direct.direct_mode_enabled; const mode = el('direct-mode'); mode.textContent = on ? 'ON' : 'OFF'; mode.className = on ? 'status-on' : 'status-off'; el('account-count').textContent = String(accounts.length); el('agy-count').textContent = String(agy); el('gpt-count').textContent = String(gpt); }
function overviewCard(kind, label, description){ const list = providerAccounts(kind); const models = list.reduce((sum,a)=>sum + modelCount(a), 0); const enabled = list.filter(a=>a.enabled).length; const tone = list.length ? 'ok' : 'warn'; return '<article class="provider-card"><div class="provider-card-head"><div><h3>' + esc(label) + '</h3><div class="muted">' + esc(description) + '</div></div><span class="pill ' + tone + '">' + list.length + ' account' + (list.length === 1 ? '' : 's') + '</span></div><div class="provider-stats"><div class="provider-stat"><span>Accounts</span><strong>' + list.length + '</strong></div><div class="provider-stat"><span>Models</span><strong>' + models + '</strong></div><div class="provider-stat"><span>Enabled</span><strong>' + enabled + '</strong></div></div><div class="actions"><button class="btn primary" type="button" data-open-provider="' + esc(kind) + '">Configure ' + esc(label) + '</button></div></article>'; }
function renderOverview(){ el('overview-providers').innerHTML = overviewCard('agy2api', 'Antigravity', 'AGY2API direct provider') + overviewCard('gpt2api', 'ChatGPT', 'GPT2API direct provider'); }
function accountSelectHTML(kind){ const list = providerAccounts(kind); const current = list.find(a=>a.account_id === selectedAccount) || list[0]; const value = current ? esc(current.label || current.channel_name || current.account_id) : 'Select account'; const options = list.map(a=>'<button type="button" class="custom-select-option' + (a.account_id === selectedAccount ? ' selected' : '') + '" data-account-option="' + esc(a.account_id) + '">' + esc(a.label || a.channel_name || a.account_id) + '</button>').join(''); return '<div class="field"><label>Account</label><div class="custom-select" data-account-select><button type="button" class="custom-select-trigger" aria-haspopup="listbox" aria-expanded="false"><span class="custom-select-value">' + value + '</span><span class="custom-select-arrow"></span></button><div class="custom-select-panel" role="listbox">' + options + '</div></div></div>'; }
function bindAccountSelect(){ const box = document.querySelector('[data-account-select]'); if(!box){ return; } const trigger = box.querySelector('.custom-select-trigger'); const panel = box.querySelector('.custom-select-panel'); if(!trigger || !panel){ return; } trigger.onclick = ev=>{ ev.stopPropagation(); const open = box.classList.toggle('open'); trigger.setAttribute('aria-expanded', open ? 'true' : 'false'); }; panel.querySelectorAll('[data-account-option]').forEach(button=>button.onclick = ev=>{ ev.stopPropagation(); selectedAccount = button.dataset.accountOption; render(); }); if(!window.__accountSelectOutsideBound){ window.__accountSelectOutsideBound = true; document.addEventListener('click', ev=>{ if(ev.target.closest && ev.target.closest('[data-account-select]')){ return; } document.querySelectorAll('[data-account-select].open').forEach(node=>{ node.classList.remove('open'); const nodeTrigger = node.querySelector('.custom-select-trigger'); if(nodeTrigger){ nodeTrigger.setAttribute('aria-expanded', 'false'); } }); }); } }
function renderProvider(){ if(provider === 'global') return; selectedAccount = firstAccount(provider); const list = providerAccounts(provider); if(!list.length){ el('provider-content').innerHTML = '<div class="empty">No accounts configured. Use Add account below to create one.</div>' + accountsView(list); bindActions(null); return; } const account = accountByID(selectedAccount) || list[0]; selectedAccount = account.account_id; const header = accountSelectHTML(provider); let body = ''; if(view === 'accounts') body = accountsView(list); else if(view === 'models') body = modelsView(account); else if(view === 'headers') body = headersView(account); else body = routingView(account); el('provider-content').innerHTML = header + body; bindAccountSelect(); bindActions(account); }
function accountsView(list){ const rows = list.map(a=>'<tr><td><strong>' + esc(a.label || a.channel_name || a.account_id) + '</strong><div class="muted mono">' + esc(a.account_id) + (a.auth_id ? ' · auth ' + esc(a.auth_id) : '') + '</div></td><td>' + esc(a.channel_name) + '</td><td><code>' + esc(a.prefix) + '</code></td><td>' + esc(a.base_url || 'redacted') + '</td><td>' + (a.api_key_configured ? '<span class="pill ok">configured</span>' : '<span class="pill warn">missing</span>') + '</td><td>' + esc(String(a.weight)) + '</td><td>' + (a.proxy_url_configured ? '<span class="pill ok">configured</span>' : '<span class="pill">none</span>') + '</td><td>' + (a.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>') + '</td><td>' + a.priority + '</td><td>' + modelCount(a) + '</td><td><button class="btn" data-action="scan" data-account="' + esc(a.account_id) + '">Scan</button><button class="btn" data-action="publish" data-account="' + esc(a.account_id) + '">Preview provider update</button><button class="btn" data-action="upsert" data-account="' + esc(a.account_id) + '">Upsert provider</button><button class="btn danger" data-action="remove-account" data-account="' + esc(a.account_id) + '">Remove</button></td></tr>').join(''); return '<div class="actions"><button class="btn primary" data-action="new-account">Add account</button><button class="btn" data-action="import-providers">Import from AI providers</button><button class="btn" data-action="refresh">Refresh</button></div><div id="import-providers-panel" class="hidden" style="margin:12px 0"><div class="section-title">Import from AI providers</div><div class="muted" style="margin-bottom:10px">Existing CPA openai-compatibility providers. Imported credentials stay server-side.</div><div id="import-provider-list" class="table-wrap"><div class="empty">Loading providers...</div></div></div><div id="new-account-form" class="hidden" style="margin:12px 0"><div class="section-title">New account draft</div><div class="form-grid"><div class="field"><label>Account ID</label><input id="draft-account-id" placeholder="agy-prod-2"></div><div class="field"><label>Label</label><input id="draft-label" placeholder="Production B"></div><div class="field"><label>CPA provider name</label><input id="draft-channel" placeholder="Antigravity"></div><div class="field"><label>CPA prefix</label><input id="draft-prefix" placeholder="agy"></div><div class="field"><label>Base URL</label><input id="draft-base-url" placeholder="https://agy2api.example/v1"></div><div class="field"><label>Priority</label><input id="draft-priority" type="number" value="0"></div><div class="field"><label>Weight</label><input id="draft-weight" type="number" value="1"></div><div class="field"><label>Proxy URL</label><input id="draft-proxy-url" placeholder="optional"></div></div><div class="field"><label>API key</label><input id="draft-api-key" type="password" autocomplete="new-password" placeholder="write-only"></div><div class="actions"><button class="btn" data-action="draft-account">Create draft</button><button class="btn primary" data-action="save-account">Save to CPA config</button><span class="muted">Saving writes plugins.configs.</div></div><div class="table-wrap"><table class="table"><thead><tr><th>Account</th><th>CPA provider</th><th>Prefix</th><th>Base URL</th><th>API key</th><th>Weight</th><th>Proxy URL</th><th>Enabled</th><th>Priority</th><th>Models</th><th>Actions</th></tr></thead><tbody>' + rows + '</tbody></table></div>'; }
function modelsView(account){ const rows = (account.models || []).map(m=>'<tr><td><code>' + esc(m.upstream_id) + '</code>' + (m.unavailable ? ' <span class="pill warn">unavailable</span>' : '') + '</td><td>' + esc(m.alias || '') + '</td><td>' + (m.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>') + '</td><td>' + (m.image ? '<span class="pill">image</span>' : '') + '</td><td>' + (m.thinking ? '<span class="pill">thinking</span>' : '') + '</td></tr>').join(''); return '<div class="actions"><button class="btn" data-action="scan" data-account="' + esc(account.account_id) + '">Fetch models</button><button class="btn" data-action="sync-provider-models" data-account="' + esc(account.account_id) + '">Sync from provider</button><button class="btn" data-action="scan-upsert" data-account="' + esc(account.account_id) + '">Scan &amp; write models</button><button class="btn" data-action="publish" data-account="' + esc(account.account_id) + '">Preview provider update</button></div><div class="table-wrap"><table class="table"><thead><tr><th>Upstream model</th><th>Client alias</th><th>Enabled</th><th>Image</th><th>Thinking</th></tr></thead><tbody>' + rows + '</tbody></table></div>'; }
function headersView(account){ const rows = (account.headers || []).map(h=>'<tr><td><code>' + esc(h.key) + '</code></td><td>' + (h.configured ? '<span class="pill ok">configured</span>' : '<span class="pill warn">missing</span>') + '</td></tr>').join('') || '<tr><td colspan="2"><div class="empty">No static headers configured.</div></td></tr>'; return '<div class="split"><div><div class="section-title">Static request headers</div><div class="table-wrap"><table class="table"><thead><tr><th>Name</th><th>State</th></tr></thead><tbody>' + rows + '</tbody></table></div></div><div><div class="section-title">Write-only update</div><div class="field"><label>Header name</label><input id="new-header-key" placeholder="X-Static-Header"></div><div class="field"><label>Header value</label><input id="new-header-value" type="password" placeholder="write-only value"></div><button class="btn" data-action="header-preview" data-account="' + esc(account.account_id) + '">Keep as draft</button><div class="muted" style="margin-top:8px">Values are never echoed after save.</div></div></div>'; }
function routingView(account){ const signing = account.identity_signing_enabled ? '<span class="pill ok">signing on</span>' : '<span class="pill warn">signing off</span>'; const enabled = account.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>'; return '<div class="grid"><div class="metric"><span>CPA prefix</span><strong>' + esc(account.prefix) + '</strong></div><div class="metric"><span>CPA provider</span><strong>' + esc(account.channel_name) + '</strong></div><div class="metric"><span>Priority</span><strong>' + esc(String(account.priority)) + '</strong></div><div class="metric"><span>Weight</span><strong>' + esc(String(account.weight)) + '</strong></div></div><div class="chips" style="margin-top:12px"><span class="chip">' + esc(account.provider_kind) + '</span>' + signing + enabled + '</div><div style="margin-top:12px"><button class="btn" data-action="publish" data-account="' + esc(account.account_id) + '">Preview CPA provider update</button></div>'; }
function renderLogs(){ const events = state.diagnostics.recent_events || []; el('runtime-log').innerHTML = events.length ? events.map(ev=>'<div class="event">' + esc(ev.at || '') + '  ' + esc((ev.level || 'info').toUpperCase()) + '  ' + esc(ev.message || '') + '</div>').join('') : 'No runtime events since plugin load.'; }
function renderSettings(){ const warnings = state.direct.warnings || []; el('settings-content').innerHTML = '<div class="chips"><span class="chip">plugin ' + esc(state.version) + '</span><span class="chip">' + (state.direct.direct_mode_enabled ? 'direct provider mode' : 'legacy mirror') + '</span><span class="chip">' + accounts.length + ' account(s)</span></div>' + (warnings.length ? '<div class="notice" style="margin-top:10px">' + warnings.map(esc).join('<br>') + '</div>' : '') + '<div class="muted" style="margin-top:10px">Direct provider updates edit the matching CPA openai-compatibility row.</div>'; }
async function loadImportProviders(){ const list = el('import-provider-list'); if(!list){ return; } list.innerHTML = '<div class="empty">Loading providers...</div>'; try{ const data = await call('/direct/providers/import?kind=' + encodeURIComponent(provider)); const items = (data.providers || []).filter(item=>item.kind === provider); if(!items.length){ const label = provider === 'agy2api' ? 'Antigravity' : 'ChatGPT'; list.innerHTML = '<div class="empty">No ' + esc(label) + ' providers were found in CPA config.</div>'; return; } const rows = items.map(item=>'<tr><td><span class="pill ' + (item.kind === 'agy2api' ? 'ok' : '') + '">' + esc(item.kind) + '</span></td><td><strong>' + esc(item.name) + '</strong><div class="muted mono">' + esc(item.base_url || '') + '</div></td><td>' + esc(item.prefix || '') + '</td><td>' + esc(String(item.model_count || 0)) + '</td><td>' + (item.key_configured ? '<span class="pill ok">key saved</span>' : '<span class="pill warn">missing</span>') + '</td><td>' + (item.imported ? '<span class="pill ok">imported</span>' : '<span class="pill">new</span>') + '</td><td><button class="btn primary" data-import-index="' + esc(String(item.index)) + '" data-import-name="' + esc(item.name) + '">Import</button></td></tr>').join(''); list.innerHTML = '<table class="table"><thead><tr><th>Kind</th><th>Provider</th><th>Prefix</th><th>Models</th><th>Key</th><th>State</th><th>Actions</th></tr></thead><tbody>' + rows + '</tbody></table>'; bindImportActions(); }catch(e){ list.innerHTML = '<div class="empty">' + esc(e.message) + '</div>'; } }
function bindImportActions(){ document.querySelectorAll('[data-import-index]').forEach(btn=>btn.onclick = async ev=>{ const node = ev.currentTarget; node.disabled = true; try{ const data = await call('/direct/providers/import', {index:Number(node.dataset.importIndex), name:node.dataset.importName}); show(data); notice('Imported provider into direct accounts. Reloading.', 'ok'); setTimeout(()=>location.reload(), 500); }catch(e){ node.disabled = false; notice('Import failed: ' + e.message, 'err'); show('Import failed: ' + e.message); } }); }
function bindActions(account){ document.querySelectorAll('[data-action="refresh"]').forEach(btn=>btn.onclick = ()=>location.reload()); document.querySelectorAll('[data-action="new-account"]').forEach(btn=>btn.onclick = ()=>el('new-account-form').classList.toggle('hidden')); document.querySelectorAll('[data-action="import-providers"]').forEach(btn=>btn.onclick = async ()=>{ const panel = el('import-providers-panel'); panel.classList.toggle('hidden'); if(!panel.classList.contains('hidden')){ await loadImportProviders(); } }); const draftBtn = document.querySelector('[data-action="draft-account"]'); if(draftBtn){ draftBtn.onclick = ()=>{ const apiKey = el('draft-api-key').value.trim(); const draft = {account_id:el('draft-account-id').value.trim(), provider_kind:provider, label:el('draft-label').value.trim(), channel_name:el('draft-channel').value.trim(), prefix:el('draft-prefix').value.trim(), base_url:el('draft-base-url').value.trim(), priority:Number(el('draft-priority').value || 0), weight:Number(el('draft-weight').value || 1), proxy_url:el('draft-proxy-url').value.trim(), enabled:true, identity_signing_enabled:true, api_key_configured:apiKey !== ''}; if(!draft.account_id || !draft.channel_name || !draft.prefix){ notice('Draft needs account id, CPA provider name, and prefix.', 'warn'); return; } lastDraft = draft; el('draft-api-key').value = ''; show(draft); notice('Draft account rendered below. No CPA config was written and the key was not echoed.', 'ok'); }; } document.querySelectorAll('[data-action="scan"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/scan', {account_id:id}); show(data); notice('Scan returned ' + (data.model_count || 0) + ' models.', 'ok'); }catch(e){ notice('Scan failed: ' + e.message, 'err'); show('Scan failed: ' + e.message); } }); document.querySelectorAll('[data-action="scan-upsert"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/scan/upsert', {account_id:id}); show(data); notice('Original provider models updated from upstream scan. Reloading.', 'ok'); setTimeout(()=>location.reload(), 500); }catch(e){ notice('Scan/write failed: ' + e.message, 'err'); show('Scan/write failed: ' + e.message); } }); document.querySelectorAll('[data-action="publish"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/publish', {account_id:id}); show(data); notice('CPA provider preview generated. No CPA config was written.', 'ok'); }catch(e){ notice('Provider preview failed: ' + e.message, 'err'); show('Provider preview failed: ' + e.message); } }); document.querySelectorAll('[data-action="upsert"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/upsert', {account_id:id}); show(data); notice('CPA provider updated.', 'ok'); }catch(e){ notice('Provider upsert failed: ' + e.message, 'err'); show('Provider upsert failed: ' + e.message); } }); const headerBtn = document.querySelector('[data-action="header-preview"]'); if(headerBtn){ headerBtn.onclick = ()=>{ const key = el('new-header-key').value.trim(); const value = el('new-header-value').value.trim(); if(!key || !value){ notice('Enter a header name and value first.', 'warn'); return; } el('new-header-value').value = ''; notice(key + ' staged as a write-only draft header for ' + account.account_id + '. It is not saved by this console.', 'ok'); }; } }
document.querySelectorAll('.nav-item').forEach(node=>{ node.onclick = ()=>{ const nextProvider = node.dataset.provider; const nextView = node.dataset.view; if(nextProvider === 'global'){ provider = 'global'; view = nextView; } else { provider = nextProvider; view = nextView; } render(); }; });
document.querySelectorAll('[data-provider-tab]').forEach(node=>{ node.onclick = ()=>setTopTab(node.dataset.providerTab); });
document.addEventListener('click', (ev)=>{ const node = ev.target && ev.target.closest ? ev.target.closest('[data-open-provider]') : null; if(!node){ return; } ev.preventDefault(); setProvider(node.dataset.openProvider); });
el('open-editor').onclick = ()=>{ if(provider === 'global'){ provider = lastProvider; } view = 'accounts'; render(); const account = firstAccount(provider); if(account){ selectedAccount = account; render(); notice('Editing CPA provider row for ' + account + '. Upsert writes the matching CPA provider row.', 'ok'); } else { notice('No account is configured for ' + provider + '. Add an account draft first.', 'warn'); } };
document.querySelectorAll('.tab').forEach(node=>{ node.onclick = ()=>setView(node.dataset.view); });
let usageRequestID = 0;
function bindUsageFilters(){ const pane = el('usage-pane'); const form = pane ? pane.querySelector('.usage-filters') : null; if(!form || form.dataset.bound === 'true'){ return; } form.dataset.bound = 'true'; form.addEventListener('submit', ev=>{ ev.preventDefault(); refreshUsageFilters(form); }); form.querySelectorAll('select').forEach(select=>select.addEventListener('change', ()=>refreshUsageFilters(form))); }
async function refreshUsageFilters(form){ const requestID = ++usageRequestID; const url = new URL(window.location.href); url.search = new URLSearchParams(new FormData(form)).toString(); form.classList.add('is-loading'); try{ const res = await fetch(url.toString(), {headers:{'X-Requested-With':'fetch'}}); if(!res.ok){ throw new Error('HTTP ' + res.status); } const html = await res.text(); if(requestID !== usageRequestID){ return; } const doc = new DOMParser().parseFromString(html, 'text/html'); const nextPane = doc.getElementById('usage-pane'); const pane = el('usage-pane'); if(!nextPane || !pane){ throw new Error('usage pane missing'); } pane.innerHTML = nextPane.innerHTML; history.replaceState(null, '', url.toString()); bindUsageFilters(); notice('Usage filter applied.', 'ok'); }catch(err){ notice('Usage filter failed: ' + err.message, 'err'); }finally{ form.classList.remove('is-loading'); } }
render();
bindUsageFilters();
// Persistent writes are delegated from document so they bind correctly even
// though the account table re-renders on every tab switch.
document.addEventListener('click', async (ev)=>{
 const node = ev.target && ev.target.closest ? ev.target.closest('[data-action]') : null;
 if(!node){ return; }
 const action = node.dataset.action;
 if(action !== 'save-account' && action !== 'remove-account' && action !== 'toggle-mode' && action !== 'sync-provider-models'){ return; }
 ev.preventDefault();
 const original = node.textContent;
 node.disabled = true;
 try{
  if(action === 'save-account'){
   const draft = {
    account_id: (el('draft-account-id') ? el('draft-account-id').value : '').trim(),
    provider_kind: provider,
    label: (el('draft-label') ? el('draft-label').value : '').trim(),
    channel_name: (el('draft-channel') ? el('draft-channel').value : '').trim(),
    prefix: (el('draft-prefix') ? el('draft-prefix').value : '').trim(),
    base_url: (el('draft-base-url') ? el('draft-base-url').value : '').trim(),
    priority: Number((el('draft-priority') ? el('draft-priority').value : 0) || 0),
    weight: Number((el('draft-weight') ? el('draft-weight').value : 1) || 1),
    proxy_url: (el('draft-proxy-url') ? el('draft-proxy-url').value : '').trim(),
    enabled: true,
    identity_signing_enabled: true
   };
   if(!draft.account_id || !draft.channel_name || !draft.prefix){ notice('Saving needs account id, CPA provider name, and prefix.', 'warn'); return; }
   const data = await call('/direct/accounts/add', draft);
   show(data);
   notice('Account ' + draft.account_id + ' saved to CPA plugin config. Reloading.', 'ok');
   setTimeout(()=>location.reload(), 500);
   return;
  }
  if(action === 'remove-account'){
   const id = node.dataset.account;
   if(!window.confirm('Remove direct account ' + id + ' from CPA plugin config? The CPA provider row is left untouched.')){ return; }
   const data = await call('/direct/accounts/remove', {account_id: id});
   show(data);
   notice('Removed ' + id + ' from plugin config. The provider row and its keys were not changed.', 'ok');
   setTimeout(()=>location.reload(), 500);
   return;
  }
  if(action === 'sync-provider-models'){
   const id = node.dataset.account;
   const data = await call('/direct/accounts/sync-provider-models', {account_id: id});
   show(data);
   notice('Synced ' + (data.model_count || 0) + ' models from ' + (data.provider_name || 'provider') + '. Reloading.', 'ok');
   setTimeout(()=>location.reload(), 500);
   return;
  }
  const next = !(state.direct && state.direct.direct_mode_enabled);
  const data = await call('/direct/mode', {enabled: next});
  show(data);
  notice('Direct mode is now ' + (data.direct_mode_enabled ? 'on' : 'off') + ' in CPA config. Reloading.', 'ok');
  setTimeout(()=>location.reload(), 500);
 }catch(err){
  notice('Write rejected, CPA config unchanged: ' + err.message, 'err');
  show('Write rejected: ' + err.message);
 }finally{
  node.disabled = false;
  node.textContent = original;
 }
});
function syncModeButton(){ const node = el('toggle-mode'); if(!node){ return; } const on = !!(state.direct && state.direct.direct_mode_enabled); const noAccounts = accounts.length === 0; node.textContent = on ? 'Turn direct mode off' : (noAccounts ? 'Add account first' : 'Turn direct mode on'); node.className = 'btn ' + (on ? 'danger' : 'primary'); node.disabled = !on && noAccounts; }
const originalRender = render;
render = function(){ originalRender(); syncModeButton(); };
syncModeButton();
const managementInput = el('mkey');
if(managementInput){ managementInput.value = sessionStorage.getItem('agyBridgeManagementKey') || ''; }
el('save-mkey').onclick = ()=>{ const value = managementInput.value.trim(); if(!value){ notice('Enter the CPA management key first.', 'warn'); return; } sessionStorage.setItem('agyBridgeManagementKey', value); notice('Management key saved for this browser session.', 'ok'); };
el('clear-mkey').onclick = ()=>{ sessionStorage.removeItem('agyBridgeManagementKey'); managementInput.value = ''; notice('Management key cleared from this browser session.', 'warn'); };
</script>
</body>
</html>`,
		html.EscapeString(data.Version),
		html.EscapeString(status),
		usageMain,
		seed,
		pluginID,
	)
}
