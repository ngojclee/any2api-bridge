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
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",sans-serif;font-size:14px;line-height:1.45}button,input,select{font:inherit}.shell{display:grid;grid-template-columns:250px minmax(0,1fr);min-height:100vh}.sidebar{background:#fffdf8;border-right:1px solid var(--line);padding:18px 14px;position:sticky;top:0;height:100vh;overflow:auto}.brand{margin-bottom:16px}.brand h1{font-size:19px;line-height:1.2;margin:0 0 4px}.muted{color:var(--ink-2);font-size:12px}.nav-group{margin-top:14px}.nav-title{font-size:11px;font-weight:800;text-transform:uppercase;letter-spacing:.04em;color:var(--ink-3);margin-bottom:8px}.nav-item{display:block;width:100%%;border:0;border-radius:7px;background:transparent;color:var(--ink-2);padding:8px 10px;margin:2px 0;text-align:left;cursor:pointer}.nav-item:hover{background:var(--surface);color:var(--ink)}.nav-item.active{background:var(--accent);color:#fff}.nav-sub{margin-left:10px;border-left:2px solid var(--line);padding-left:8px}.content{padding:22px;min-width:0}.top{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;margin-bottom:14px}.top h2{font-size:20px;margin:0 0 4px}.card{background:var(--panel);border:1px solid var(--line);border-radius:var(--radius);box-shadow:var(--shadow);padding:14px;margin-bottom:12px}.section-title{font-size:11px;font-weight:800;text-transform:uppercase;letter-spacing:.04em;color:var(--ink-3);margin-bottom:10px}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.metric{background:var(--inset);border:1px solid var(--line);border-radius:6px;padding:10px}.metric span{display:block;color:var(--ink-3);font-size:11px;font-weight:750;text-transform:uppercase;letter-spacing:.04em}.metric strong{display:block;margin-top:3px;font-size:18px}.pill{display:inline-flex;align-items:center;border-radius:999px;border:1px solid var(--line-2);background:var(--surface);padding:4px 9px;font-size:12px;font-weight:700;color:var(--ink-2)}.pill.ok{background:var(--success-bg);border-color:#5eead4;color:var(--success)}.pill.warn{background:var(--warn-bg);border-color:#f6d365;color:var(--warn)}.pill.err{background:var(--error-bg);border-color:#f0a79d;color:var(--error)}
.actions{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.btn{border:1px solid var(--line-2);background:#fff;color:var(--ink);border-radius:6px;height:34px;padding:0 11px;font-weight:650;font-size:13px;cursor:pointer;text-decoration:none;display:inline-flex;align-items:center;justify-content:center}.btn.primary{background:var(--accent);border-color:var(--accent);color:#fff}.btn.danger{color:var(--error);border-color:#f0a79d}.btn:disabled{opacity:.55;cursor:default}.btn:hover{filter:brightness(.985)}.btn:disabled{opacity:.5;cursor:not-allowed}.table-wrap{overflow:auto;border:1px solid var(--line);border-radius:8px}.table{width:100%%;border-collapse:collapse;background:#fff;font-size:12.5px}.table th,.table td{padding:9px 10px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}.table th{background:var(--inset);color:var(--ink-2);font-size:11px;text-transform:uppercase;letter-spacing:.04em}.table tr:last-child td{border-bottom:0}.mono{font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace}.field{margin-bottom:12px}.field label{display:block;font-size:12px;color:var(--ink-2);font-weight:650;margin-bottom:5px}.field input,.field select{width:100%%;min-height:36px;border:1px solid var(--line-2);background:#fff;border-radius:8px;color:var(--ink);padding:8px 10px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.notice{background:var(--warn-bg);border:1px solid #f6d365;color:var(--warn);border-radius:6px;padding:10px;margin-bottom:10px}.result{white-space:pre-wrap;word-break:break-word;border-radius:6px;padding:10px;background:#22221f;color:#eceae5;font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace;font-size:12px;max-height:280px;overflow:auto}.chips{display:flex;gap:7px;flex-wrap:wrap}.chip{background:var(--surface);border:1px solid var(--line-2);border-radius:999px;padding:5px 9px;font-size:12px;font-weight:650}.log{max-height:220px;overflow:auto;background:#22221f;color:#eceae5;border-radius:6px;padding:10px;font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace;font-size:12px}.event{padding:3px 0}.tabs{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px}.tab{border:1px solid var(--line-2);background:#fff;border-radius:7px;padding:7px 10px;color:var(--ink-2);cursor:pointer}.tab.active{background:var(--accent);border-color:var(--accent);color:#fff}.hidden{display:none!important}.split{display:grid;grid-template-columns:minmax(0,1.3fr) minmax(280px,.7fr);gap:12px}.empty{border:1px dashed var(--line-2);border-radius:8px;padding:14px;color:var(--ink-3);background:var(--inset)}
@media(max-width:960px){.shell{grid-template-columns:1fr}.sidebar{position:relative;height:auto;border-right:0;border-bottom:1px solid var(--line)}.content{padding:16px}.grid,.form-grid,.split{grid-template-columns:1fr}}
</style>
</head>
<body style="margin:0;width:100%%;max-width:none">
<div class="shell" style="width:100%%;max-width:none">
<aside class="sidebar">
<div class="brand"><h1>Any2Api Bridge</h1><div class="muted">Direct provider console</div><div class="muted">v%s</div></div>
<div class="nav-group"><div class="nav-title">Overview</div><button class="nav-item" data-provider="global" data-view="overview">Service overview</button></div>
<div class="nav-group"><div class="nav-title">Antigravity / AGY2API original provider</div><button class="nav-item" data-provider="agy2api" data-view="accounts">Antigravity accounts</button><div class="nav-sub"><button class="nav-item" data-provider="agy2api" data-view="models">Models</button><button class="nav-item" data-provider="agy2api" data-view="headers">Headers</button><button class="nav-item" data-provider="agy2api" data-view="routing">Routing</button></div></div>
<div class="nav-group"><div class="nav-title">ChatGPT / GPT2API original provider</div><button class="nav-item" data-provider="gpt2api" data-view="accounts">ChatGPT accounts</button><div class="nav-sub"><button class="nav-item" data-provider="gpt2api" data-view="models">Models</button><button class="nav-item" data-provider="gpt2api" data-view="headers">Headers</button><button class="nav-item" data-provider="gpt2api" data-view="routing">Routing</button></div></div>
<div class="nav-group"><div class="nav-title">Global</div><button class="nav-item" data-provider="global" data-view="usage">Usage</button><button class="nav-item" data-provider="global" data-view="logs">Logs</button><button class="nav-item" data-provider="global" data-view="settings">Settings</button></div>
</aside>
<main class="content" style="width:100%%;max-width:none;padding:18px 22px;box-sizing:border-box">
<div class="top"><div><h2 id="page-title">Direct accounts</h2><div class="muted">Operational console for original CPA providers. Secrets stay write-only; this page only shows configured state.</div></div><div class="actions"><button class="btn primary" id="open-editor" type="button">Open editor</button><span class="pill">%s</span></div></div>
<div class="provider-tabs" style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px"><button class="provider-tab active" type="button" data-provider-tab="agy2api" style="border:1px solid var(--line-2);background:var(--accent);color:#fff;border-radius:7px;padding:8px 12px;font-weight:700;cursor:pointer">Antigravity</button><button class="provider-tab" type="button" data-provider-tab="gpt2api" style="border:1px solid var(--line-2);background:#fff;color:var(--ink-2);border-radius:7px;padding:8px 12px;font-weight:700;cursor:pointer">ChatGPT</button></div>
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
<div class="section-title">Service overview</div>
<div class="grid">
<div class="metric"><span>Direct mode</span><strong id="overview-direct-mode">-</strong></div>
<div class="metric"><span>Original providers</span><strong id="overview-providers">2</strong></div>
<div class="metric"><span>Accounts</span><strong id="overview-accounts">0</strong></div>
<div class="metric"><span>Models</span><strong id="overview-models">0</strong></div>
</div>
<div id="overview-health" class="chips" style="margin-top:12px"></div>
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
let provider = 'agy2api';
let view = 'accounts';
let selectedAccount = '';
const accounts = state.direct.accounts || [];
function esc(value){ return String(value === undefined || value === null ? '' : value).replace(/[&<>"']/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
function el(id){ return document.getElementById(id); }
function notice(text, kind){ const node = el('notice'); node.textContent = text || ''; node.className = 'notice' + (kind ? ' ' + kind : ''); node.classList.toggle('hidden', !text); }
function show(value){ el('result').textContent = typeof value === 'string' ? value : JSON.stringify(value, null, 2); }
async function call(path, body){ const res = await fetch(MGT + path, {method: body ? 'POST' : 'GET', headers: body ? {'Content-Type':'application/json'} : {}, body: body ? JSON.stringify(body) : undefined}); const text = await res.text(); let data; try{ data = JSON.parse(text); }catch{ data = text; } if(!res.ok) throw new Error(typeof data === 'string' ? data : JSON.stringify(data)); return data; }
function providerAccounts(kind){ return accounts.filter(a=>a.provider_kind === kind); }
function firstAccount(kind){ const list = providerAccounts(kind); if(!list.length) return ''; if(selectedAccount && list.some(a=>a.account_id === selectedAccount)) return selectedAccount; return list[0].account_id; }
function accountByID(id){ return accounts.find(a=>a.account_id === id) || null; }
function setProvider(kind){ provider = kind; selectedAccount = firstAccount(kind); render(); }
function setView(next){ view = next; render(); }
function modelCount(account){ return (account.models || []).length; }
function render(){ document.querySelectorAll('[data-provider-tab]').forEach(node=>{ const active = node.dataset.providerTab === provider; node.style.background = active ? 'var(--accent)' : '#fff'; node.style.color = active ? '#fff' : 'var(--ink-2)'; }); document.querySelectorAll('.nav-item').forEach(node=>{ node.classList.toggle('active', node.dataset.provider === provider && node.dataset.view === view || node.dataset.provider === 'global' && node.dataset.view === view && provider === 'global'); }); document.querySelectorAll('.tab').forEach(node=>node.classList.toggle('active', node.dataset.view === view)); el('overview-pane').classList.toggle('hidden', !(provider === 'global' && view === 'overview')); el('provider-pane').classList.toggle('hidden', provider === 'global'); el('usage-pane').classList.toggle('hidden', !(provider === 'global' && view === 'usage')); el('logs-pane').classList.toggle('hidden', !(provider === 'global' && view === 'logs')); el('settings-pane').classList.toggle('hidden', !(provider === 'global' && view === 'settings')); el('page-title').textContent = provider === 'global' ? view[0].toUpperCase() + view.slice(1) : (provider === 'agy2api' ? 'Antigravity / AGY2API' : 'ChatGPT / GPT2API') + ' - ' + view; renderSummary(); renderOverview(); renderProvider(); renderLogs(); renderSettings(); }
function renderSummary(){ const agy = providerAccounts('agy2api').length; const gpt = providerAccounts('gpt2api').length; el('direct-mode').textContent = state.direct.direct_mode_enabled ? 'on' : 'off'; el('account-count').textContent = String(accounts.length); el('agy-count').textContent = String(agy); el('gpt-count').textContent = String(gpt); }
function renderOverview(){ const models = accounts.reduce((sum,a)=>sum + modelCount(a), 0); el('overview-direct-mode').textContent = state.direct.direct_mode_enabled ? 'on' : 'off'; el('overview-accounts').textContent = String(accounts.length); el('overview-models').textContent = String(models); const agy = providerAccounts('agy2api').length; const gpt = providerAccounts('gpt2api').length; el('overview-health').innerHTML = '<span class="chip">Antigravity accounts: ' + agy + '</span><span class="chip">ChatGPT accounts: ' + gpt + '</span><span class="chip">' + (state.direct.direct_mode_enabled ? 'direct path ready' : 'legacy mirror rollback') + '</span>'; }
function accountOptions(kind){ return providerAccounts(kind).map(a=>'<option value="' + esc(a.account_id) + '"' + (a.account_id === selectedAccount ? ' selected' : '') + '>' + esc(a.label || a.channel_name || a.account_id) + '</option>').join(''); }
function renderProvider(){ if(provider === 'global') return; selectedAccount = firstAccount(provider); const list = providerAccounts(provider); if(!list.length){ el('provider-content').innerHTML = '<div class="empty">No ' + esc(provider) + ' direct accounts configured yet. Add them under plugin config direct_accounts.</div>'; return; } const account = accountByID(selectedAccount) || list[0]; selectedAccount = account.account_id; const header = '<div class="field"><label>Account</label><select id="account-select">' + accountOptions(provider) + '</select></div>'; let body = ''; if(view === 'accounts') body = accountsView(list); else if(view === 'models') body = modelsView(account); else if(view === 'headers') body = headersView(account); else body = routingView(account); el('provider-content').innerHTML = header + body; el('account-select').onchange = ev=>{ selectedAccount = ev.target.value; render(); }; bindActions(account); }
function accountsView(list){ const rows = list.map(a=>'<tr><td><strong>' + esc(a.label || a.channel_name || a.account_id) + '</strong><div class="muted mono">' + esc(a.account_id) + (a.auth_id ? ' · auth ' + esc(a.auth_id) : '') + '</div></td><td>' + esc(a.channel_name) + '</td><td><code>' + esc(a.prefix) + '</code></td><td>' + esc(a.base_url || 'redacted') + '</td><td>' + (a.api_key_configured ? '<span class="pill ok">configured</span>' : '<span class="pill warn">missing</span>') + '</td><td>' + esc(String(a.weight)) + '</td><td>' + (a.proxy_url_configured ? '<span class="pill ok">configured</span>' : '<span class="pill">none</span>') + '</td><td>' + (a.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>') + '</td><td>' + a.priority + '</td><td>' + modelCount(a) + '</td><td><button class="btn" data-action="scan" data-account="' + esc(a.account_id) + '">Scan</button><button class="btn" data-action="publish" data-account="' + esc(a.account_id) + '">Preview provider update</button><button class="btn" data-action="upsert" data-account="' + esc(a.account_id) + '">Upsert provider</button><button class="btn danger" data-action="remove-account" data-account="' + esc(a.account_id) + '">Remove</button></td></tr>').join(''); return '<div class="actions"><button class="btn primary" data-action="new-account">Add account</button><button class="btn" data-action="refresh">Refresh</button></div><div id="new-account-form" class="hidden" style="margin:12px 0"><div class="section-title">New account draft</div><div class="form-grid"><div class="field"><label>Account ID</label><input id="draft-account-id" placeholder="agy-prod-2"></div><div class="field"><label>Label</label><input id="draft-label" placeholder="Production B"></div><div class="field"><label>CPA provider name</label><input id="draft-channel" placeholder="Antigravity"></div><div class="field"><label>CPA prefix</label><input id="draft-prefix" placeholder="agy"></div><div class="field"><label>Base URL</label><input id="draft-base-url" placeholder="https://agy2api.example/v1"></div><div class="field"><label>Priority</label><input id="draft-priority" type="number" value="0"></div><div class="field"><label>Weight</label><input id="draft-weight" type="number" value="1"></div><div class="field"><label>Proxy URL</label><input id="draft-proxy-url" placeholder="optional"></div></div><div class="field"><label>API key</label><input id="draft-api-key" type="password" autocomplete="new-password" placeholder="write-only"></div><div class="actions"><button class="btn" data-action="draft-account">Create draft</button><button class="btn primary" data-action="save-account">Save to CPA config</button><span class="muted">Saving writes plugins.configs.</div></div><div class="table-wrap"><table class="table"><thead><tr><th>Account</th><th>CPA provider</th><th>Prefix</th><th>Base URL</th><th>API key</th><th>Weight</th><th>Proxy URL</th><th>Enabled</th><th>Priority</th><th>Models</th><th>Actions</th></tr></thead><tbody>' + rows + '</tbody></table></div>'; }
function modelsView(account){ const rows = (account.models || []).map(m=>'<tr><td><code>' + esc(m.upstream_id) + '</code>' + (m.unavailable ? ' <span class="pill warn">unavailable</span>' : '') + '</td><td>' + esc(m.alias || '') + '</td><td>' + (m.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>') + '</td><td>' + (m.image ? '<span class="pill">image</span>' : '') + '</td><td>' + (m.thinking ? '<span class="pill">thinking</span>' : '') + '</td></tr>').join(''); return '<div class="actions"><button class="btn" data-action="scan" data-account="' + esc(account.account_id) + '">Fetch models</button><button class="btn" data-action="scan-upsert" data-account="' + esc(account.account_id) + '">Scan &amp; write models</button><button class="btn" data-action="publish" data-account="' + esc(account.account_id) + '">Preview provider update</button></div><div class="table-wrap"><table class="table"><thead><tr><th>Upstream model</th><th>Client alias</th><th>Enabled</th><th>Image</th><th>Thinking</th></tr></thead><tbody>' + rows + '</tbody></table></div>'; }
function headersView(account){ const rows = (account.headers || []).map(h=>'<tr><td><code>' + esc(h.key) + '</code></td><td>' + (h.configured ? '<span class="pill ok">configured</span>' : '<span class="pill warn">missing</span>') + '</td></tr>').join('') || '<tr><td colspan="2"><div class="empty">No static headers configured.</div></td></tr>'; return '<div class="split"><div><div class="section-title">Static request headers</div><div class="table-wrap"><table class="table"><thead><tr><th>Name</th><th>State</th></tr></thead><tbody>' + rows + '</tbody></table></div></div><div><div class="section-title">Write-only update</div><div class="field"><label>Header name</label><input id="new-header-key" placeholder="X-Static-Header"></div><div class="field"><label>Header value</label><input id="new-header-value" type="password" placeholder="write-only value"></div><button class="btn" data-action="header-preview" data-account="' + esc(account.account_id) + '">Keep as draft</button><div class="muted" style="margin-top:8px">Values are never echoed after save.</div></div></div>'; }
function routingView(account){ const signing = account.identity_signing_enabled ? '<span class="pill ok">signing on</span>' : '<span class="pill warn">signing off</span>'; const enabled = account.enabled ? '<span class="pill ok">enabled</span>' : '<span class="pill">disabled</span>'; return '<div class="grid"><div class="metric"><span>CPA prefix</span><strong>' + esc(account.prefix) + '</strong></div><div class="metric"><span>CPA provider</span><strong>' + esc(account.channel_name) + '</strong></div><div class="metric"><span>Priority</span><strong>' + esc(String(account.priority)) + '</strong></div><div class="metric"><span>Weight</span><strong>' + esc(String(account.weight)) + '</strong></div></div><div class="chips" style="margin-top:12px"><span class="chip">' + esc(account.provider_kind) + '</span>' + signing + enabled + '</div><div style="margin-top:12px"><button class="btn" data-action="publish" data-account="' + esc(account.account_id) + '">Preview CPA provider update</button></div>'; }
function renderLogs(){ const events = state.diagnostics.recent_events || []; el('runtime-log').innerHTML = events.length ? events.map(ev=>'<div class="event">' + esc(ev.at || '') + '  ' + esc((ev.level || 'info').toUpperCase()) + '  ' + esc(ev.message || '') + '</div>').join('') : 'No runtime events since plugin load.'; }
function renderSettings(){ const warnings = state.direct.warnings || []; el('settings-content').innerHTML = '<div class="chips"><span class="chip">plugin ' + esc(state.version) + '</span><span class="chip">' + (state.direct.direct_mode_enabled ? 'direct provider mode' : 'legacy mirror') + '</span><span class="chip">' + accounts.length + ' account(s)</span></div>' + (warnings.length ? '<div class="notice" style="margin-top:10px">' + warnings.map(esc).join('<br>') + '</div>' : '') + '<div class="muted" style="margin-top:10px">agy2api and gpt2api are the original providers. Direct provider updates edit their CPA openai-compatibility row.</div>'; }
function bindActions(account){ document.querySelectorAll('[data-action="refresh"]').forEach(btn=>btn.onclick = ()=>location.reload()); document.querySelectorAll('[data-action="new-account"]').forEach(btn=>btn.onclick = ()=>el('new-account-form').classList.toggle('hidden')); const draftBtn = document.querySelector('[data-action="draft-account"]'); if(draftBtn){ draftBtn.onclick = ()=>{ const apiKey = el('draft-api-key').value.trim(); const draft = {account_id:el('draft-account-id').value.trim(), provider_kind:provider, label:el('draft-label').value.trim(), channel_name:el('draft-channel').value.trim(), prefix:el('draft-prefix').value.trim(), base_url:el('draft-base-url').value.trim(), priority:Number(el('draft-priority').value || 0), weight:Number(el('draft-weight').value || 1), proxy_url:el('draft-proxy-url').value.trim(), enabled:true, identity_signing_enabled:true, api_key_configured:apiKey !== ''}; if(!draft.account_id || !draft.channel_name || !draft.prefix){ notice('Draft needs account id, CPA provider name, and prefix.', 'warn'); return; } lastDraft = draft; el('draft-api-key').value = ''; show(draft); notice('Draft account rendered below. No CPA config was written and the key was not echoed.', 'ok'); }; } document.querySelectorAll('[data-action="scan"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/scan', {account_id:id}); show(data); notice('Scan returned ' + (data.model_count || 0) + ' models.', 'ok'); }catch(e){ notice('Scan failed: ' + e.message, 'err'); show('Scan failed: ' + e.message); } }); document.querySelectorAll('[data-action="scan-upsert"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/scan/upsert', {account_id:id}); show(data); notice('Original provider models updated from upstream scan.', 'ok'); }catch(e){ notice('Scan/write failed: ' + e.message, 'err'); show('Scan/write failed: ' + e.message); } }); document.querySelectorAll('[data-action="publish"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/publish', {account_id:id}); show(data); notice('CPA provider preview generated. No CPA config was written.', 'ok'); }catch(e){ notice('Provider preview failed: ' + e.message, 'err'); show('Provider preview failed: ' + e.message); } }); document.querySelectorAll('[data-action="upsert"]').forEach(btn=>btn.onclick = async ev=>{ const id = ev.currentTarget.dataset.account; try{ const data = await call('/direct/accounts/upsert', {account_id:id}); show(data); notice('CPA provider updated.', 'ok'); }catch(e){ notice('Provider upsert failed: ' + e.message, 'err'); show('Provider upsert failed: ' + e.message); } }); const headerBtn = document.querySelector('[data-action="header-preview"]'); if(headerBtn){ headerBtn.onclick = ()=>{ const key = el('new-header-key').value.trim(); const value = el('new-header-value').value.trim(); if(!key || !value){ notice('Enter a header name and value first.', 'warn'); return; } el('new-header-value').value = ''; notice(key + ' staged as a write-only draft header for ' + account.account_id + '. It is not saved by this console.', 'ok'); }; } }
document.querySelectorAll('.nav-item').forEach(node=>{ node.onclick = ()=>{ const nextProvider = node.dataset.provider; const nextView = node.dataset.view; if(nextProvider === 'global'){ provider = 'global'; view = nextView; } else { provider = nextProvider; view = nextView; } render(); }; });
document.querySelectorAll('[data-provider-tab]').forEach(node=>{ node.onclick = ()=>{ provider = node.dataset.providerTab; view = 'accounts'; render(); }; });
el('open-editor').onclick = ()=>{ if(provider === 'global'){ provider = 'agy2api'; } view = 'accounts'; render(); const account = firstAccount(provider); if(account){ selectedAccount = account; render(); notice('Editing CPA provider row for ' + account + '. Upsert writes the original provider row.', 'ok'); } else { notice('No account is configured for ' + provider + '. Add an account draft first.', 'warn'); } };
document.querySelectorAll('.tab').forEach(node=>{ node.onclick = ()=>setView(node.dataset.view); });
render();
// Persistent writes are delegated from document so they bind correctly even
// though the account table re-renders on every tab switch.
document.addEventListener('click', async (ev)=>{
 const node = ev.target && ev.target.closest ? ev.target.closest('[data-action]') : null;
 if(!node){ return; }
 const action = node.dataset.action;
 if(action !== 'save-account' && action !== 'remove-account' && action !== 'toggle-mode'){ return; }
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
function syncModeButton(){ const node = el('toggle-mode'); if(!node){ return; } const on = !!(state.direct && state.direct.direct_mode_enabled); node.textContent = on ? 'Turn direct mode off' : 'Turn direct mode on'; }
const originalRender = render;
render = function(){ originalRender(); syncModeButton(); };
syncModeButton();
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
