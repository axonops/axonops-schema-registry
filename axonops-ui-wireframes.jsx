import { useState } from "react";

const colors = {
  bg: "#0f1117",
  surface: "#161922",
  surfaceHover: "#1c2030",
  border: "#2a2e3d",
  borderActive: "#3b82f6",
  text: "#e2e8f0",
  textMuted: "#8892a8",
  textDim: "#5a6278",
  accent: "#3b82f6",
  accentHover: "#2563eb",
  green: "#22c55e",
  greenBg: "#0f2d1a",
  red: "#ef4444",
  redBg: "#2d0f0f",
  amber: "#f59e0b",
  amberBg: "#2d250f",
  purple: "#a855f7",
  purpleBg: "#1f0f2d",
  cyan: "#06b6d4",
};

const Badge = ({ children, color = "blue" }) => {
  const colorMap = {
    blue: { bg: "#1e3a5f", text: "#60a5fa" },
    green: { bg: colors.greenBg, text: colors.green },
    amber: { bg: colors.amberBg, text: colors.amber },
    red: { bg: colors.redBg, text: colors.red },
    purple: { bg: colors.purpleBg, text: colors.purple },
    cyan: { bg: "#0f2d2d", text: colors.cyan },
  };
  const c = colorMap[color] || colorMap.blue;
  return (
    <span style={{ background: c.bg, color: c.text, padding: "2px 8px", borderRadius: 4, fontSize: 11, fontWeight: 600, letterSpacing: 0.5 }}>
      {children}
    </span>
  );
};

const Card = ({ title, value, subtitle, color = "blue" }) => (
  <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: "16px 20px", flex: 1, minWidth: 150 }}>
    <div style={{ fontSize: 12, color: colors.textMuted, marginBottom: 4 }}>{title}</div>
    <div style={{ fontSize: 28, fontWeight: 700, color: colors.text }}>{value}</div>
    {subtitle && <div style={{ fontSize: 11, color: colors.textDim, marginTop: 2 }}>{subtitle}</div>}
  </div>
);

const NavItem = ({ icon, label, active, onClick }) => (
  <div onClick={onClick} style={{ display: "flex", alignItems: "center", gap: 8, padding: "8px 12px", borderRadius: 6, cursor: "pointer", background: active ? colors.surfaceHover : "transparent", color: active ? colors.accent : colors.textMuted, fontSize: 13, fontWeight: active ? 600 : 400, borderLeft: active ? `2px solid ${colors.accent}` : "2px solid transparent" }}>
    <span style={{ fontSize: 14, width: 18, textAlign: "center" }}>{icon}</span>
    {label}
  </div>
);

const Btn = ({ children, variant = "primary", small, onClick }) => {
  const styles = {
    primary: { background: colors.accent, color: "#fff", border: "none" },
    outline: { background: "transparent", color: colors.textMuted, border: `1px solid ${colors.border}` },
    danger: { background: colors.redBg, color: colors.red, border: `1px solid #4a1a1a` },
    ghost: { background: "transparent", color: colors.textMuted, border: "none" },
  };
  const s = styles[variant];
  return (
    <button onClick={onClick} style={{ ...s, padding: small ? "4px 10px" : "6px 14px", borderRadius: 6, fontSize: small ? 11 : 12, fontWeight: 500, cursor: "pointer", display: "inline-flex", alignItems: "center", gap: 4 }}>
      {children}
    </button>
  );
};

const DashboardPage = () => (
  <div>
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20 }}>
      <div>
        <h2 style={{ color: colors.text, fontSize: 20, fontWeight: 700, margin: 0 }}>Dashboard</h2>
        <p style={{ color: colors.textMuted, fontSize: 13, margin: "4px 0 0" }}>Overview of your schema registry</p>
      </div>
      <Btn>+ New Subject</Btn>
    </div>
    <div style={{ display: "flex", gap: 12, marginBottom: 20, flexWrap: "wrap" }}>
      <Card title="Subjects" value="47" subtitle="3 contexts" />
      <Card title="Schema Versions" value="218" subtitle="12 this week" />
      <Card title="Compatibility" value="BACKWARD" subtitle="Global default" />
      <Card title="Mode" value="READWRITE" subtitle="All contexts" />
    </div>
    <div style={{ display: "flex", gap: 16 }}>
      <div style={{ flex: 2 }}>
        <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 16 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: colors.text, marginBottom: 12 }}>Recent Activity</div>
          {[
            { action: "Schema registered", subject: "orders-value", version: "v4", time: "2 min ago", user: "jmiller" },
            { action: "Compatibility changed", subject: "payments-value", version: "FULL", time: "18 min ago", user: "admin" },
            { action: "Subject created", subject: "notifications-value", version: "v1", time: "1 hr ago", user: "jmiller" },
            { action: "KEK created", subject: "prod-encrypt-key", version: "vault", time: "3 hrs ago", user: "admin" },
            { action: "Schema registered", subject: "users-value", version: "v7", time: "5 hrs ago", user: "asmith" },
          ].map((e, i) => (
            <div key={i} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "8px 0", borderTop: i > 0 ? `1px solid ${colors.border}` : "none" }}>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span style={{ color: colors.textMuted, fontSize: 12, width: 130 }}>{e.action}</span>
                <span style={{ color: colors.accent, fontSize: 12, fontWeight: 500 }}>{e.subject}</span>
                <Badge color={e.action.includes("KEK") ? "purple" : "blue"}>{e.version}</Badge>
              </div>
              <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                <span style={{ color: colors.textDim, fontSize: 11 }}>{e.user}</span>
                <span style={{ color: colors.textDim, fontSize: 11 }}>{e.time}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
      <div style={{ flex: 1 }}>
        <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 16 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: colors.text, marginBottom: 12 }}>System Info</div>
          {[
            ["Storage", "PostgreSQL"],
            ["Auth", "OIDC (Okta)"],
            ["Contexts", "3 active"],
            ["Exporters", "2 running"],
            ["CSFLE", "Vault KMS"],
            ["Rate Limit", "Enabled"],
            ["Audit Log", "Enabled"],
          ].map(([k, v], i) => (
            <div key={i} style={{ display: "flex", justifyContent: "space-between", padding: "6px 0", borderTop: i > 0 ? `1px solid ${colors.border}` : "none" }}>
              <span style={{ color: colors.textMuted, fontSize: 12 }}>{k}</span>
              <span style={{ color: colors.text, fontSize: 12, fontWeight: 500 }}>{v}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  </div>
);

const SubjectListPage = () => (
  <div>
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
      <h2 style={{ color: colors.text, fontSize: 20, fontWeight: 700, margin: 0 }}>Subjects</h2>
      <div style={{ display: "flex", gap: 8 }}>
        <Btn variant="outline">⬇ Export All</Btn>
        <Btn>+ New Subject</Btn>
      </div>
    </div>
    <div style={{ display: "flex", gap: 8, marginBottom: 14 }}>
      <input placeholder="Search subjects..." style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 6, padding: "7px 12px", color: colors.text, fontSize: 13, flex: 1, outline: "none" }} />
      <select style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 6, padding: "7px 12px", color: colors.textMuted, fontSize: 12 }}>
        <option>All Types</option><option>AVRO</option><option>PROTOBUF</option><option>JSON</option>
      </select>
      <select style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 6, padding: "7px 12px", color: colors.textMuted, fontSize: 12 }}>
        <option>All Tags</option><option>production</option><option>staging</option><option>deprecated</option>
      </select>
    </div>
    <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, overflow: "hidden" }}>
      <div style={{ display: "grid", gridTemplateColumns: "2fr 80px 60px 100px 100px 100px 60px", padding: "10px 16px", borderBottom: `1px solid ${colors.border}`, fontSize: 11, fontWeight: 600, color: colors.textDim, textTransform: "uppercase", letterSpacing: 0.5 }}>
        <span>Subject</span><span>Type</span><span>Ver</span><span>Compat</span><span>Tags</span><span>Modified</span><span></span>
      </div>
      {[
        { name: "users-value", type: "AVRO", tc: "blue", ver: 7, compat: "BACKWARD", tags: ["production"], mod: "2 min" },
        { name: "orders-value", type: "PROTO", tc: "green", ver: 4, compat: "FULL", tags: ["production", "encrypted"], mod: "18 min" },
        { name: "payments-value", type: "JSON", tc: "amber", ver: 3, compat: "NONE", tags: ["staging"], mod: "1 hr" },
        { name: "notifications-value", type: "AVRO", tc: "blue", ver: 1, compat: "BACKWARD", tags: [], mod: "1 hr" },
        { name: "inventory-key", type: "PROTO", tc: "green", ver: 2, compat: "FULL_T", tags: ["production"], mod: "3 hrs" },
        { name: "analytics-value", type: "JSON", tc: "amber", ver: 12, compat: "BACKWARD", tags: ["deprecated"], mod: "1 day" },
      ].map((s, i) => (
        <div key={i} style={{ display: "grid", gridTemplateColumns: "2fr 80px 60px 100px 100px 100px 60px", padding: "10px 16px", borderBottom: `1px solid ${colors.border}`, alignItems: "center", cursor: "pointer" }}
          onMouseEnter={e => e.currentTarget.style.background = colors.surfaceHover}
          onMouseLeave={e => e.currentTarget.style.background = "transparent"}>
          <span style={{ color: colors.accent, fontSize: 13, fontWeight: 500 }}>{s.name}</span>
          <Badge color={s.tc}>{s.type}</Badge>
          <span style={{ color: colors.text, fontSize: 13 }}>v{s.ver}</span>
          <span style={{ color: colors.textMuted, fontSize: 12 }}>{s.compat}</span>
          <div style={{ display: "flex", gap: 4 }}>{s.tags.map((t,j) => <Badge key={j} color={t === "deprecated" ? "red" : t === "encrypted" ? "purple" : "cyan"}>{t}</Badge>)}</div>
          <span style={{ color: colors.textDim, fontSize: 12 }}>{s.mod}</span>
          <Btn variant="ghost" small>⬇</Btn>
        </div>
      ))}
    </div>
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: 12 }}>
      <span style={{ color: colors.textDim, fontSize: 12 }}>6 subjects · Context: Default (.)</span>
      <div style={{ display: "flex", gap: 4 }}>
        <Btn variant="outline" small>← Prev</Btn>
        <Btn variant="outline" small>Next →</Btn>
      </div>
    </div>
  </div>
);

const SchemaEditorPage = () => (
  <div>
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{ color: colors.textMuted, fontSize: 13, cursor: "pointer" }}>Subjects</span>
        <span style={{ color: colors.textDim }}>›</span>
        <span style={{ color: colors.accent, fontSize: 13, fontWeight: 500 }}>users-value</span>
        <Badge color="blue">AVRO</Badge>
        <Badge color="cyan">production</Badge>
      </div>
      <div style={{ display: "flex", gap: 8 }}>
        <Btn variant="outline">⬇ Download Schema</Btn>
        <Btn variant="outline">📋 Copy ID: 42</Btn>
        <Btn>+ Register New Version</Btn>
      </div>
    </div>
    <div style={{ display: "flex", gap: 16 }}>
      <div style={{ width: 200, flexShrink: 0 }}>
        <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 12 }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: colors.text, marginBottom: 8 }}>Versions</div>
          {[7,6,5,4,3,2,1].map(v => (
            <div key={v} style={{ padding: "6px 8px", borderRadius: 4, cursor: "pointer", background: v === 7 ? colors.surfaceHover : "transparent", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <span style={{ color: v === 7 ? colors.accent : colors.textMuted, fontSize: 12, fontWeight: v === 7 ? 600 : 400 }}>v{v}</span>
              <span style={{ color: colors.textDim, fontSize: 10 }}>{v === 7 ? "latest" : ""}</span>
            </div>
          ))}
          <div style={{ marginTop: 12, paddingTop: 12, borderTop: `1px solid ${colors.border}` }}>
            <div style={{ fontSize: 11, color: colors.textMuted, marginBottom: 6 }}>Compare versions</div>
            <div style={{ display: "flex", gap: 4 }}>
              <select style={{ background: colors.bg, border: `1px solid ${colors.border}`, borderRadius: 4, padding: "3px 6px", color: colors.text, fontSize: 11, flex: 1 }}>
                <option>v6</option>
              </select>
              <span style={{ color: colors.textDim, fontSize: 11, padding: "3px" }}>↔</span>
              <select style={{ background: colors.bg, border: `1px solid ${colors.border}`, borderRadius: 4, padding: "3px 6px", color: colors.text, fontSize: 11, flex: 1 }}>
                <option>v7</option>
              </select>
            </div>
            <Btn variant="outline" small>View Diff</Btn>
          </div>
          <div style={{ marginTop: 12, paddingTop: 12, borderTop: `1px solid ${colors.border}` }}>
            <div style={{ fontSize: 11, color: colors.textMuted, marginBottom: 6 }}>Bulk Download</div>
            <Btn variant="outline" small>⬇ All Versions (.zip)</Btn>
          </div>
        </div>
        <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 12, marginTop: 12 }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: colors.text, marginBottom: 8 }}>Details</div>
          {[["Compat", "BACKWARD"], ["Schema ID", "42"], ["References", "None"], ["Tags", "production"], ["Encrypted", "No"]].map(([k,v],i) => (
            <div key={i} style={{ display: "flex", justifyContent: "space-between", padding: "4px 0", fontSize: 11 }}>
              <span style={{ color: colors.textDim }}>{k}</span>
              <span style={{ color: colors.text }}>{v}</span>
            </div>
          ))}
        </div>
      </div>
      <div style={{ flex: 1 }}>
        <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, overflow: "hidden" }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "8px 12px", borderBottom: `1px solid ${colors.border}` }}>
            <div style={{ display: "flex", gap: 8 }}>
              <Btn variant="ghost" small>Schema</Btn>
              <Btn variant="ghost" small>Tree View</Btn>
              <Btn variant="ghost" small>Diff v6 ↔ v7</Btn>
              <Btn variant="ghost" small>Data Contract</Btn>
            </div>
            <div style={{ display: "flex", gap: 6 }}>
              <Btn variant="outline" small>Normalized</Btn>
              <Btn variant="outline" small>⬇ JSON</Btn>
            </div>
          </div>
          <div style={{ padding: 16, fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: 12, lineHeight: 1.7, color: colors.textMuted, minHeight: 300, background: "#0d0f14" }}>
            <div><span style={{color: "#c9a0dc"}}>{"{"}</span></div>
            <div style={{paddingLeft: 16}}><span style={{color: "#9cdcfe"}}>"type"</span><span style={{color: colors.textDim}}>:</span> <span style={{color: "#ce9178"}}>"record"</span><span style={{color: colors.textDim}}>,</span></div>
            <div style={{paddingLeft: 16}}><span style={{color: "#9cdcfe"}}>"name"</span><span style={{color: colors.textDim}}>:</span> <span style={{color: "#ce9178"}}>"User"</span><span style={{color: colors.textDim}}>,</span></div>
            <div style={{paddingLeft: 16}}><span style={{color: "#9cdcfe"}}>"namespace"</span><span style={{color: colors.textDim}}>:</span> <span style={{color: "#ce9178"}}>"com.axonops.events"</span><span style={{color: colors.textDim}}>,</span></div>
            <div style={{paddingLeft: 16}}><span style={{color: "#9cdcfe"}}>"fields"</span><span style={{color: colors.textDim}}>:</span> <span style={{color: "#c9a0dc"}}>{"["}</span></div>
            <div style={{paddingLeft: 32}}><span style={{color: "#c9a0dc"}}>{"{"}</span> <span style={{color: "#9cdcfe"}}>"name"</span>: <span style={{color: "#ce9178"}}>"id"</span>, <span style={{color: "#9cdcfe"}}>"type"</span>: <span style={{color: "#ce9178"}}>"long"</span> <span style={{color: "#c9a0dc"}}>{"}"}</span>,</div>
            <div style={{paddingLeft: 32}}><span style={{color: "#c9a0dc"}}>{"{"}</span> <span style={{color: "#9cdcfe"}}>"name"</span>: <span style={{color: "#ce9178"}}>"name"</span>, <span style={{color: "#9cdcfe"}}>"type"</span>: <span style={{color: "#ce9178"}}>"string"</span> <span style={{color: "#c9a0dc"}}>{"}"}</span>,</div>
            <div style={{paddingLeft: 32}}><span style={{color: "#c9a0dc"}}>{"{"}</span> <span style={{color: "#9cdcfe"}}>"name"</span>: <span style={{color: "#ce9178"}}>"email"</span>, <span style={{color: "#9cdcfe"}}>"type"</span>: <span style={{color: "#ce9178"}}>"string"</span> <span style={{color: "#c9a0dc"}}>{"}"}</span>,</div>
            <div style={{paddingLeft: 32, background: colors.greenBg, marginLeft: -16, paddingLeft: 48, marginRight: -16, paddingRight: 16}}><span style={{color: colors.green}}>+</span> <span style={{color: "#c9a0dc"}}>{"{"}</span> <span style={{color: "#9cdcfe"}}>"name"</span>: <span style={{color: "#ce9178"}}>"phone"</span>, <span style={{color: "#9cdcfe"}}>"type"</span>: [<span style={{color: "#ce9178"}}>"null"</span>, <span style={{color: "#ce9178"}}>"string"</span>], <span style={{color: "#9cdcfe"}}>"default"</span>: <span style={{color: "#569cd6"}}>null</span> <span style={{color: "#c9a0dc"}}>{"}"}</span> <span style={{color: colors.green, fontSize: 10, marginLeft: 8}}>✓ BACKWARD COMPATIBLE</span></div>
            <div style={{paddingLeft: 16}}><span style={{color: "#c9a0dc"}}>{"]"}</span></div>
            <div><span style={{color: "#c9a0dc"}}>{"}"}</span></div>
          </div>
        </div>
      </div>
    </div>
  </div>
);

const DiffPage = () => (
  <div>
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{ color: colors.accent, fontSize: 13 }}>users-value</span>
        <span style={{ color: colors.textDim }}>—</span>
        <span style={{ color: colors.text, fontSize: 14, fontWeight: 600 }}>Schema Diff</span>
        <Badge color="blue">v5</Badge>
        <span style={{ color: colors.textDim }}>↔</span>
        <Badge color="green">v7</Badge>
      </div>
      <Btn variant="outline">⬇ Download Diff</Btn>
    </div>
    <div style={{ display: "flex", gap: 2, borderRadius: 8, overflow: "hidden" }}>
      <div style={{ flex: 1, background: "#0d0f14", padding: 16, fontFamily: "monospace", fontSize: 12, lineHeight: 1.8 }}>
        <div style={{ color: colors.textDim, fontSize: 10, marginBottom: 8, fontWeight: 600 }}>VERSION 5</div>
        <div style={{ color: colors.textMuted }}>{"{"} "type": "record",</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>"name": "User",</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>"fields": [</div>
        <div style={{ color: colors.textMuted, paddingLeft: 32 }}>{"{"} "name": "id", "type": "long" {"}"},</div>
        <div style={{ color: colors.textMuted, paddingLeft: 32 }}>{"{"} "name": "name", "type": "string" {"}"},</div>
        <div style={{ background: colors.redBg, paddingLeft: 32, marginLeft: -16, paddingRight: 16, marginRight: -16, color: colors.red }}>- {"{"} "name": "age", "type": "int" {"}"}</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>]</div>
        <div style={{ color: colors.textMuted }}>{"}"}</div>
      </div>
      <div style={{ flex: 1, background: "#0d0f14", padding: 16, fontFamily: "monospace", fontSize: 12, lineHeight: 1.8 }}>
        <div style={{ color: colors.textDim, fontSize: 10, marginBottom: 8, fontWeight: 600 }}>VERSION 7 (LATEST)</div>
        <div style={{ color: colors.textMuted }}>{"{"} "type": "record",</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>"name": "User",</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>"fields": [</div>
        <div style={{ color: colors.textMuted, paddingLeft: 32 }}>{"{"} "name": "id", "type": "long" {"}"},</div>
        <div style={{ color: colors.textMuted, paddingLeft: 32 }}>{"{"} "name": "name", "type": "string" {"}"},</div>
        <div style={{ background: colors.greenBg, paddingLeft: 32, marginLeft: -16, paddingRight: 16, marginRight: -16, color: colors.green }}>+ {"{"} "name": "email", "type": "string" {"}"}</div>
        <div style={{ background: colors.greenBg, paddingLeft: 32, marginLeft: -16, paddingRight: 16, marginRight: -16, color: colors.green }}>+ {"{"} "name": "phone", "type": ["null","string"], "default": null {"}"}</div>
        <div style={{ color: colors.textMuted, paddingLeft: 16 }}>]</div>
        <div style={{ color: colors.textMuted }}>{"}"}</div>
      </div>
    </div>
    <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 12, marginTop: 12 }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: colors.text, marginBottom: 8 }}>Compatibility Analysis (BACKWARD)</div>
      <div style={{ display: "flex", gap: 16 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}><span style={{ color: colors.red }}>✗</span><span style={{ color: colors.textMuted, fontSize: 12 }}>Removed field "age" (int) — <strong style={{ color: colors.red }}>BREAKING</strong> for readers expecting this field</span></div>
      </div>
      <div style={{ display: "flex", gap: 16, marginTop: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}><span style={{ color: colors.green }}>✓</span><span style={{ color: colors.textMuted, fontSize: 12 }}>Added "email" (string) with no default — safe for new readers</span></div>
      </div>
      <div style={{ display: "flex", gap: 16, marginTop: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}><span style={{ color: colors.green }}>✓</span><span style={{ color: colors.textMuted, fontSize: 12 }}>Added "phone" (union null|string, default null) — backward compatible</span></div>
      </div>
    </div>
  </div>
);

const AdminConfigPage = () => (
  <div>
    <h2 style={{ color: colors.text, fontSize: 20, fontWeight: 700, margin: "0 0 4px" }}>Server Configuration</h2>
    <p style={{ color: colors.textMuted, fontSize: 13, margin: "0 0 16px" }}>Running configuration and system health — secrets are redacted</p>
    <div style={{ display: "flex", gap: 12, marginBottom: 16, flexWrap: "wrap" }}>
      {[["Go", "1.26.0"], ["Version", "2.1.0"], ["Uptime", "14d 7h"], ["Goroutines", "84"], ["Memory", "47 MB"]].map(([k,v],i) => (
        <div key={i} style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 6, padding: "8px 14px" }}>
          <div style={{ fontSize: 10, color: colors.textDim }}>{k}</div>
          <div style={{ fontSize: 14, fontWeight: 600, color: colors.text }}>{v}</div>
        </div>
      ))}
    </div>
    <div style={{ display: "flex", gap: 12, marginBottom: 16, flexWrap: "wrap" }}>
      {["RBAC", "Audit", "CSFLE", "Contexts", "Exporters", "Rate Limit"].map((f, i) => (
        <div key={i} style={{ display: "flex", alignItems: "center", gap: 4, background: colors.greenBg, border: `1px solid #1a3d1a`, borderRadius: 4, padding: "3px 8px" }}>
          <span style={{ color: colors.green, fontSize: 11 }}>✓</span>
          <span style={{ color: colors.green, fontSize: 11 }}>{f}</span>
        </div>
      ))}
    </div>
    {[
      { title: "Storage", items: [["Backend", "PostgreSQL"], ["Host", "db.internal:5432"], ["Database", "schema_registry"], ["Password", "[REDACTED]"], ["Pool Size", "25"], ["Migration", "Up to date"]] },
      { title: "Authentication", items: [["Method", "OIDC"], ["Issuer", "https://axonops.okta.com"], ["Client ID", "0oa4x..."], ["Client Secret", "[REDACTED]"], ["Redirect URI", "https://registry.axonops.com/api/v1/auth/oidc/callback"]] },
      { title: "TLS", items: [["Enabled", "true"], ["Cert", "/etc/ssl/registry.crt"], ["Key", "[REDACTED]"], ["Min Version", "TLS 1.3"], ["mTLS", "Disabled"]] },
    ].map((section, i) => (
      <div key={i} style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 14, marginBottom: 10 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: colors.text, marginBottom: 8 }}>{section.title}</div>
        {section.items.map(([k, v], j) => (
          <div key={j} style={{ display: "flex", justifyContent: "space-between", padding: "4px 0", borderTop: j > 0 ? `1px solid ${colors.border}` : "none" }}>
            <span style={{ color: colors.textMuted, fontSize: 12 }}>{k}</span>
            <span style={{ color: v === "[REDACTED]" ? colors.amber : colors.text, fontSize: 12, fontFamily: v === "[REDACTED]" ? "inherit" : "monospace" }}>{v}</span>
          </div>
        ))}
      </div>
    ))}
  </div>
);

const ApiDocsPage = () => (
  <div>
    <h2 style={{ color: colors.text, fontSize: 20, fontWeight: 700, margin: "0 0 4px" }}>API Documentation</h2>
    <p style={{ color: colors.textMuted, fontSize: 13, margin: "0 0 16px" }}>Interactive API reference generated from OpenAPI specification</p>
    <div style={{ display: "flex", gap: 8, marginBottom: 16 }}>
      <Btn variant="outline" small>ReDoc View</Btn>
      <Btn variant="outline" small>Swagger UI</Btn>
      <Btn variant="outline" small>⬇ OpenAPI YAML</Btn>
    </div>
    <div style={{ background: colors.surface, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 24, textAlign: "center", minHeight: 300, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center" }}>
      <div style={{ fontSize: 40, marginBottom: 12 }}>📖</div>
      <div style={{ color: colors.text, fontSize: 15, fontWeight: 600, marginBottom: 4 }}>ReDoc Interactive Documentation</div>
      <div style={{ color: colors.textMuted, fontSize: 12, maxWidth: 400 }}>
        Full Confluent-compatible Schema Registry API reference rendered from the bundled openapi.yml — includes all endpoints for schemas, subjects, compatibility, config, mode, contexts, DEK registry, and exporters.
      </div>
      <div style={{ marginTop: 16, display: "flex", gap: 8 }}>
        {["Schemas", "Subjects", "Compatibility", "Config", "Mode", "Contexts", "DEK Registry", "Exporters"].map((g, i) => (
          <Badge key={i} color={["blue","green","amber","cyan","purple","blue","purple","green"][i]}>{g}</Badge>
        ))}
      </div>
    </div>
  </div>
);

const pages = {
  dashboard: { label: "Dashboard", icon: "◫", component: DashboardPage },
  subjects: { label: "Subjects", icon: "☰", component: SubjectListPage },
  editor: { label: "Schema Detail", icon: "{ }", component: SchemaEditorPage },
  diff: { label: "Schema Diff", icon: "⇔", component: DiffPage },
  config: { label: "Server Config", icon: "⚙", component: AdminConfigPage },
  docs: { label: "API Docs", icon: "📖", component: ApiDocsPage },
};

export default function App() {
  const [page, setPage] = useState("dashboard");
  const [ctx, setCtx] = useState(".");
  const PageComponent = pages[page].component;

  return (
    <div style={{ display: "flex", height: "100vh", background: colors.bg, color: colors.text, fontFamily: "'DM Sans', -apple-system, sans-serif" }}>
      <div style={{ width: 220, background: colors.surface, borderRight: `1px solid ${colors.border}`, display: "flex", flexDirection: "column", flexShrink: 0 }}>
        <div style={{ padding: "16px 14px", borderBottom: `1px solid ${colors.border}` }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{ width: 28, height: 28, borderRadius: 6, background: "linear-gradient(135deg, #3b82f6, #8b5cf6)", display: "flex", alignItems: "center", justifyContent: "center", fontWeight: 800, fontSize: 13, color: "#fff" }}>A</div>
            <div>
              <div style={{ fontSize: 13, fontWeight: 700, color: colors.text }}>AxonOps</div>
              <div style={{ fontSize: 10, color: colors.textDim }}>Schema Registry</div>
            </div>
          </div>
        </div>
        <div style={{ padding: "8px 8px 4px" }}>
          <div style={{ fontSize: 10, color: colors.textDim, padding: "4px 12px", textTransform: "uppercase", letterSpacing: 1 }}>Context</div>
          <select value={ctx} onChange={e => setCtx(e.target.value)} style={{ width: "100%", background: colors.bg, border: `1px solid ${colors.border}`, borderRadius: 6, padding: "6px 8px", color: colors.text, fontSize: 12, margin: "4px 0 8px" }}>
            <option value=".">Default (.)</option>
            <option value=".team-a">.team-a</option>
            <option value=".team-b">.team-b</option>
          </select>
        </div>
        <div style={{ padding: "4px 8px", flex: 1 }}>
          <div style={{ fontSize: 10, color: colors.textDim, padding: "4px 12px", textTransform: "uppercase", letterSpacing: 1 }}>Navigation</div>
          {Object.entries(pages).map(([key, p]) => (
            <NavItem key={key} icon={p.icon} label={p.label} active={page === key} onClick={() => setPage(key)} />
          ))}
          <div style={{ fontSize: 10, color: colors.textDim, padding: "12px 12px 4px", textTransform: "uppercase", letterSpacing: 1 }}>Enterprise</div>
          <NavItem icon="🔐" label="Encryption" />
          <NavItem icon="📤" label="Exporters" />
          <NavItem icon="📜" label="Audit Log" />
        </div>
        <div style={{ padding: 12, borderTop: `1px solid ${colors.border}` }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{ width: 28, height: 28, borderRadius: "50%", background: colors.accent, display: "flex", alignItems: "center", justifyContent: "center", fontSize: 11, fontWeight: 700, color: "#fff" }}>JM</div>
            <div>
              <div style={{ fontSize: 12, color: colors.text }}>jmiller</div>
              <div style={{ fontSize: 10, color: colors.textDim }}>admin · OIDC</div>
            </div>
          </div>
        </div>
      </div>
      <div style={{ flex: 1, overflow: "auto", padding: 24 }}>
        <PageComponent />
      </div>
    </div>
  );
}
