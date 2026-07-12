import React, { useEffect, useMemo, useState } from "https://esm.sh/react@18.3.1";
import { createRoot } from "https://esm.sh/react-dom@18.3.1/client";
import htm from "https://esm.sh/htm@3.1.1";

const html = htm.bind(React.createElement);
const HISTORY_LIMIT = 30;

function clamp(value, min, max) {
	return Math.min(max, Math.max(min, value));
}

function formatCount(value) {
	return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value ?? 0);
}

function formatNumber(value, digits = 2) {
	return new Intl.NumberFormat("en-US", {
		minimumFractionDigits: digits,
		maximumFractionDigits: digits,
	}).format(Number.isFinite(value) ? value : 0);
}

function formatPercent(value) {
	return `${formatNumber(clamp(value * 100, 0, 9999), 0)}%`;
}

function formatDuration(seconds) {
	if (!Number.isFinite(seconds) || seconds <= 0) {
		return "0s";
	}
	const days = Math.floor(seconds / 86400);
	const hours = Math.floor((seconds % 86400) / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const secs = Math.floor(seconds % 60);
	const parts = [];
	if (days) parts.push(`${days}d`);
	if (hours) parts.push(`${hours}h`);
	if (minutes) parts.push(`${minutes}m`);
	if (secs || !parts.length) parts.push(`${secs}s`);
	return parts.join(" ");
}

function formatTime(value) {
	if (!value) return "just now";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return String(value);
	return new Intl.DateTimeFormat("en-US", {
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
	}).format(date);
}

function buildPath(values, width = 320, height = 80) {
	if (!values.length) {
		return `M 0 ${height} L ${width} ${height}`;
	}

	const min = Math.min(...values);
	const max = Math.max(...values);
	const span = max - min || 1;
	const step = values.length === 1 ? width : width / (values.length - 1);
	return values
		.map((value, index) => {
			const x = index * step;
			const y = height - ((value - min) / span) * (height - 8) - 4;
			return `${index === 0 ? "M" : "L"} ${x.toFixed(1)} ${y.toFixed(1)}`;
		})
		.join(" ");
}

function Sparkline({ values, stroke = "#f97316", fill = "rgba(249, 115, 22, 0.18)" }) {
	const path = useMemo(() => buildPath(values), [values]);
	const area = useMemo(() => `${path} L 320 80 L 0 80 Z`, [path]);

	return html`<svg className="sparkline" viewBox="0 0 320 80" preserveAspectRatio="none" aria-hidden="true">
		<defs>
			<linearGradient id="spark-fill" x1="0" x2="0" y1="0" y2="1">
				<stop offset="0%" stopColor=${fill.replace("rgba", "rgba").replace(", 0.18)", ", 0.22)")} />
				<stop offset="100%" stopColor="rgba(255,255,255,0)" />
			</linearGradient>
		</defs>
		<path className="fill" d=${area} fill="url(#spark-fill)" />
		<path d=${path} stroke=${stroke} />
	</svg>`;
}

function StatCard({ title, value, subtext, tone = "orange", sparkline = [], trend }) {
	const color = tone === "blue" ? "#2563eb" : tone === "teal" ? "#0f766e" : "#f97316";
	return html`<article className="card span-4">
		<div className="card-label">
			<span>${title}</span>
			${trend ? html`<span className=${`trend-row ${trend.direction === "down" ? "trend-down" : "trend-up"}`}>${trend.label}</span>` : null}
		</div>
		<div className="card-value">${value}</div>
		${subtext ? html`<div className="card-subtext">${subtext}</div>` : null}
		${sparkline.length ? html`<${Sparkline} values=${sparkline} stroke=${color} />` : html`<div />`}
	</article>`;
}

function ProgressCard({ title, value, subtext, percent, tone = "orange" }) {
	const fillClass = tone === "blue" ? "blue" : tone === "teal" ? "teal" : "";
	return html`<article className="card span-6">
		<div className="card-label"><span>${title}</span><span>${formatPercent(percent)}</span></div>
		<div className="card-value">${value}</div>
		${subtext ? html`<div className="card-subtext">${subtext}</div>` : null}
		<div className="meter">
			<div className="meter-track"><div className=${`meter-fill ${fillClass}`} style=${{ width: `${clamp(percent * 100, 0, 100)}%` }}></div></div>
			<div className="meter-meta"><span>0%</span><span>100%</span></div>
		</div>
	</article>`;
}

function ResponseMixCard({ metrics }) {
	const total = metrics.responses_2xx + metrics.responses_3xx + metrics.responses_4xx + metrics.responses_5xx || 1;
	const segments = [
		{ label: "2xx", value: metrics.responses_2xx, color: "#0f766e" },
		{ label: "3xx", value: metrics.responses_3xx, color: "#2563eb" },
		{ label: "4xx", value: metrics.responses_4xx, color: "#f97316" },
		{ label: "5xx", value: metrics.responses_5xx, color: "#dc2626" },
	];

	return html`<article className="card span-12 large">
		<div className="panel-heading">
			<h2>Response mix</h2>
			<span className="status-pill"><span className="status-dot"></span>${formatCount(total)} responses tracked</span>
		</div>
		<div className="meter">
			<div className="meter-track" style=${{ display: "flex", overflow: "hidden" }}>
				${segments.map((segment) => html`<div key=${segment.label} style=${{ width: `${(segment.value / total) * 100}%`, background: segment.color }}></div>`) }
			</div>
			<div className="meter-meta">
				${segments.map((segment) => html`<span key=${segment.label}>${segment.label}: ${formatCount(segment.value)}</span>`) }
			</div>
		</div>
		<div className="stat-grid" style=${{ marginTop: "6px" }}>
			<article className="card span-3" style=${{ boxShadow: "none", background: "rgba(255,255,255,0.42)" }}>
				<div className="card-label"><span>2xx</span></div>
				<div className="card-value" style=${{ fontSize: "2rem" }}>${formatCount(metrics.responses_2xx)}</div>
			</article>
			<article className="card span-3" style=${{ boxShadow: "none", background: "rgba(255,255,255,0.42)" }}>
				<div className="card-label"><span>4xx</span></div>
				<div className="card-value" style=${{ fontSize: "2rem" }}>${formatCount(metrics.responses_4xx)}</div>
			</article>
			<article className="card span-3" style=${{ boxShadow: "none", background: "rgba(255,255,255,0.42)" }}>
				<div className="card-label"><span>5xx</span></div>
				<div className="card-value" style=${{ fontSize: "2rem" }}>${formatCount(metrics.responses_5xx)}</div>
			</article>
			<article className="card span-3" style=${{ boxShadow: "none", background: "rgba(255,255,255,0.42)" }}>
				<div className="card-label"><span>Rate limited</span></div>
				<div className="card-value" style=${{ fontSize: "2rem" }}>${formatCount(metrics.rate_limited)}</div>
			</article>
		</div>
	</article>`;
}

function LogsPanel({ logs }) {
	return html`<section className="logs-panel section" id="logs">
		<div className="section-title">
			<div>
				<h2>Live logs</h2>
				<p>Recent server events captured from the running process.</p>
			</div>
			<div className="status-pill"><span className="status-dot"></span>${logs.length} lines</div>
		</div>
		${logs.length
			? html`<ul className="logs-list">${logs.map((entry, index) => html`<li className="log-item" key=${`${entry.time}-${index}`}>
				<div className="log-meta"><span>${formatTime(entry.time)}</span><span>${entry.time}</span></div>
				<div className="log-text">${entry.message}</div>
			</li>`)}</ul>`
			: html`<p className="empty-state">No log entries yet. Generate traffic or trigger an error to see the stream populate.</p>`}
	</section>`;
}

function App() {
	const [monitor, setMonitor] = useState(null);
	const [logs, setLogs] = useState([]);
	const [error, setError] = useState("");
	const [history, setHistory] = useState({ queue: [], active: [], latency: [] });

	useEffect(() => {
		let alive = true;

		async function load() {
			try {
				const [monitorResponse, logsResponse] = await Promise.all([
					fetch("/api/monitor", { cache: "no-store" }),
					fetch("/api/logs?limit=60", { cache: "no-store" }),
				]);

				if (!monitorResponse.ok) {
					throw new Error(`Monitor request failed with ${monitorResponse.status}`);
				}
				if (!logsResponse.ok) {
					throw new Error(`Log request failed with ${logsResponse.status}`);
				}

				const monitorJson = await monitorResponse.json();
				const logsJson = await logsResponse.json();

				if (!alive) {
					return;
				}

				setMonitor(monitorJson);
				setLogs(logsJson.logs || []);
				setHistory((current) => appendHistory(current, monitorJson));
				setError("");
			} catch (err) {
				if (!alive) return;
				setError(err instanceof Error ? err.message : "Failed to load dashboard data");
			}
		}

		load();
		const interval = setInterval(load, 2000);
		return () => {
			alive = false;
			clearInterval(interval);
		};
	}, []);

	const server = monitor?.server || {};
	const metrics = monitor?.metrics || {};
	const runtime = monitor?.runtime || {};
	const uptime = formatDuration(monitor?.uptime_seconds || 0);
	const queuePressure = server.queue_capacity ? server.queue_depth / server.queue_capacity : 0;
	const workerPressure = server.worker_pool_size ? metrics.active_connections / server.worker_pool_size : 0;
	const requestHistory = history.latency;
	const queueHistory = history.queue;
	const activeHistory = history.active;
	const latestSnapshot = monitor ? `${server.address || "unknown"} • updated ${formatTime(monitor.generated_at)}` : "Waiting for live data";

	return html`<div className="app-shell">
		<header className="topbar">
			<div className="brand">
				<div className="brand-mark">M</div>
				<div className="brand-copy">
					<strong>MTWS Monitor</strong>
					<span>Server telemetry, queue pressure, latency, and logs.</span>
				</div>
			</div>
			<nav className="nav-links" aria-label="Dashboard sections">
				<a href="#overview">Overview</a>
				<a href="#queue">Queue</a>
				<a href="#latency">Latency</a>
				<a href="#logs">Logs</a>
			</nav>
		</header>

		<section className="hero" id="overview">
			<div className="hero-copy">
				<p className="eyebrow">Server observability</p>
				<h1>A clean monitor for MTWS.</h1>
				<p>
					Track worker pressure, queue saturation, runtime footprint, request latency,
					and recent server activity from a compact React dashboard built to feel more
					like product documentation than a throwaway admin page.
				</p>
				<div className="hero-actions">
					<a className="button" href="#queue">Inspect queue</a>
					<a className="button-secondary" href="/metrics">View Prometheus metrics</a>
				</div>
				<div className="footer-note">${latestSnapshot}</div>
			</div>

			<div className="hero-panel">
				<div className="panel-heading">
					<h2>Current status</h2>
					<span className="status-pill"><span className="status-dot"></span>${error ? "degraded" : "live"}</span>
				</div>
				${error ? html`<div className="error-banner">${error}</div>` : null}
				<div className="info-list">
					<div className="info-row"><span>Server</span><strong>${server.address || "—"}</strong></div>
					<div className="info-row"><span>Static assets</span><strong>${server.static_dir || "—"}</strong></div>
					<div className="info-row"><span>TLS</span><strong>${server.tls_enabled ? "Enabled" : "Disabled"}</strong></div>
					<div className="info-row"><span>Uptime</span><strong>${uptime}</strong></div>
					<div className="info-row"><span>Runtime goroutines</span><strong>${formatCount(runtime.goroutines)}</strong></div>
					<div className="info-row"><span>Heap allocated</span><strong>${formatNumber(runtime.heap_alloc_mb)} MiB</strong></div>
				</div>
			</div>
		</section>

		<section className="section" id="queue">
			<div className="section-title">
				<div>
					<h2>Queue and workers</h2>
					<p>Capacity, active connections, and pressure against the bounded worker pool.</p>
				</div>
			</div>
			<div className="stat-grid">
				<${StatCard}
					title="Queue depth"
					value=${formatCount(server.queue_depth || 0)}
					subtext=${`of ${formatCount(server.queue_capacity || 0)} slots currently occupied`}
					tone="orange"
					sparkline=${queueHistory}
					trend=${{ label: `${formatPercent(queuePressure)} full`, direction: queuePressure > 0.75 ? "up" : "down" }}
				/>
				<${StatCard}
					title="Active connections"
					value=${formatCount(metrics.active_connections || 0)}
					subtext="Current live connections handled by the server, shown alongside worker pressure."
					tone="blue"
					sparkline=${activeHistory}
					trend=${{ label: `${formatPercent(workerPressure)} busy`, direction: workerPressure > 0.75 ? "up" : "down" }}
				/>
				<${ProgressCard}
					title="Queue saturation"
					value=${formatPercent(queuePressure)}
					subtext="Use this to spot backlog growth before rejections start climbing."
					percent=${queuePressure}
					tone="orange"
				/>
				<${ProgressCard}
					title="Worker pressure"
					value=${formatPercent(workerPressure)}
					subtext="Active connections relative to the configured worker pool size."
					percent=${workerPressure}
					tone="teal"
				/>
			</div>
		</section>

		<section className="section" id="latency">
			<div className="section-title">
				<div>
					<h2>Latency and throughput</h2>
					<p>Average and peak request latency, plus the overall response profile.</p>
				</div>
			</div>
			<div className="stat-grid">
				<${StatCard}
					title="Average latency"
					value=${`${formatNumber(metrics.average_request_duration_ms || 0)} ms`}
					subtext="Average per-request processing time captured by the server."
					tone="blue"
					sparkline=${requestHistory}
				/>
				<${StatCard}
					title="Peak latency"
					value=${`${formatNumber(metrics.max_request_duration_ms || 0)} ms`}
					subtext="Highest observed request duration since startup."
					tone="teal"
					sparkline=${requestHistory}
				/>
				<${StatCard}
					title="Accepted connections"
					value=${formatCount(metrics.accepted_connections || 0)}
					subtext="Total TCP connections observed by the listener."
					tone="orange"
					sparkline=${requestHistory}
				/>
			</div>
			<${ResponseMixCard} metrics=${metrics} />
		</section>

		<${LogsPanel} logs=${logs} />
	</div>`;
}

function appendHistory(current, monitor) {
	const nextQueue = pushValue(current.queue, monitor?.server?.queue_depth || 0);
	const nextActive = pushValue(current.active, monitor?.metrics?.active_connections || 0);
	const nextLatency = pushValue(current.latency, monitor?.metrics?.average_request_duration_ms || 0);
	return {
		queue: nextQueue,
		active: nextActive,
		latency: nextLatency,
	};
}

function pushValue(history, value) {
	const next = [...history, value];
	return next.slice(Math.max(0, next.length - HISTORY_LIMIT));
}

createRoot(document.getElementById("root")).render(html`<${App} />`);