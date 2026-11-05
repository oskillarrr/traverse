package report

const head = `<!doctype html><html lang="{{.Lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="{{.CSP}}"><title>{{.Title}} · Traverse</title><style>{{.CSSs}}</style></head><body>
{{if .Classification}}<div class="banner">{{.Classification}}</div>{{end}}`

const foot = `<footer>Traverse – Offline Architecture Analyzer · {{t "generated"}} {{.Generated}}</footer><script>{{.JSs}}</script></body></html>`

const repoTemplate = head + `
<header><h1>{{.F.Repo.Name}}</h1><p>{{if .F.Repo.Group}}{{.F.Repo.Group}} · {{end}}{{t "vcs"}}: {{.F.Repo.VCS}}{{if .F.Repo.Branch}} · {{.F.Repo.Branch}}{{end}}{{if .F.Repo.Revision}} @ {{.F.Repo.Revision}}{{end}}{{if .F.Repo.Layout}} · {{.F.Repo.Layout}}{{end}} · Traverse {{.F.Tool.Version}} · rules {{.F.Tool.RulesVersion}}</p></header>
<nav>{{if .Back}}<a href="{{.Back}}">← {{t "portfolio"}}</a>{{end}}<a href="#parties">{{t "relationships"}}</a>{{if .Structure}}<a href="#structure">{{t "repoStructure"}}</a>{{end}}<a href="#context">{{t "context"}}</a><a href="#arch">{{t "architecture"}}</a><a href="#contracts">{{t "contracts"}}</a><a href="#security">{{t "secAuth"}}</a><a href="#data">{{t "data"}}</a><a href="#modules">{{t "modules"}}</a><a href="#deploy">{{t "deployment"}}</a><a href="#tech">{{t "technologies"}}</a><a href="#risks">{{t "risks"}}</a>{{if .F.VulnScan}}<a href="#vulns">{{t "vulns"}}</a>{{end}}<a href="#components">{{t "components"}}</a><a href="#metrics">{{t "metrics"}}</a><a href="#languages">{{t "languages"}}</a></nav>
<main>
<div class="kpis">
<div class="kpi"><b>{{k .KCode}}</b><span>{{t "code"}}</span></div>
<div class="kpi"><b>{{len .F.Modules}}</b><span>{{t "modules"}}</span></div>
<div class="kpi"><b>{{len .F.Components}}</b><span>{{t "components"}}</span></div>
<div class="kpi"><b>{{.KExposed}}</b><span>{{t "exposes"}}</span></div>
<div class="kpi"><b>{{.KTables}}</b><span>{{t "tables"}}</span></div>
<div class="kpi"><b>{{.KHigh}}</b><span>{{t "highRisks"}}</span></div>
{{if .F.VulnScan}}<div class="kpi"><b>{{len .F.Vulnerabilities}}</b><span>{{t "vulns"}}</span></div>{{end}}
<div class="kpi"><b>{{len .F.Cycles}}</b><span>{{t "cycles"}}</span></div>
<div class="kpi"><b>{{len .F.LayerIssues}}</b><span>{{t "layerIssues"}}</span></div>
</div>
<p class="scope">{{t "scopeNote"}}</p>
{{if .F.Repo.Notes}}<p class="note">{{range .F.Repo.Notes}}{{.}}<br>{{end}}</p>{{end}}
{{if .Summary}}<section id="summary" class="summary"><h2>{{t "execSummary"}}</h2><ul>{{range .Summary}}<li>{{.}}</li>{{end}}</ul>
<p class="note">{{t "mdExport"}}: <a href="report.md">report.md</a> · <a href="arc42.md">arc42.md</a></p></section>{{end}}
<section id="parties"><h2>{{t "relationships"}}</h2>
<h3>{{t "callsTo"}} ({{len .Parties.Calls}})</h3>
{{if .Parties.Calls}}<div class="tw"><table><thead><tr><th>{{t "application"}}</th><th>{{t "extScope"}}</th><th>{{t "protocol"}}</th><th class="n">Endpoints</th><th>{{t "transport"}}</th><th>{{t "authCol"}}</th><th>{{t "environments"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Parties.Calls}}<tr><td><b>{{.Counterpart}}</b></td><td>{{.Scope}}</td><td>{{.Protocols}}</td><td class="n">{{.Endpoints}}</td><td>{{if .Transport}}<span class="chip {{if eq .Transport "plain"}}sev-medium{{end}}">{{xport .Transport}}</span>{{end}}</td><td>{{.Auth}}</td><td>{{.Envs}}</td><td>{{.Evidence}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noCalls"}}</p>{{end}}
<h3>{{t "calledBy"}} ({{len .Parties.CalledBy}})</h3>
{{if .Parties.CalledBy}}<div class="tw"><table><thead><tr><th>{{t "application"}}</th><th>{{t "protocol"}}</th><th class="n">{{t "interfacesN"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Parties.CalledBy}}<tr><td><b>{{.Counterpart}}</b></td><td>{{.Protocols}}</td><td class="n">{{.Endpoints}}</td><td>{{.Evidence}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noCallers"}}</p>{{end}}
{{if .Parties.SharesWith}}<p><b>{{t "sharesWith"}}:</b> {{range $i, $r := .Parties.SharesWith}}{{if $i}}, {{end}}{{$r.Counterpart}}{{end}}</p>{{end}}
{{if or .Parties.UnknownAPI .Parties.UnknownUI .Parties.UnknownAny}}<p><b>{{t "noConsumer"}}:</b> {{.Parties.UnknownAPI}} API · {{.Parties.UnknownUI}} {{t "screens"}} · {{.Parties.UnknownAny}} {{t "others"}}. <span class="note">{{t "consumerNote"}}</span></p>{{end}}
{{if .Parties.Registered}}<h3>{{t "registeredCerts"}} ({{len .Parties.Registered}})</h3><div class="tw"><table><thead><tr><th>{{t "certificate"}}</th><th>{{t "issuer"}}</th><th>{{t "validUntil"}}</th><th>{{t "status"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Parties.Registered}}<tr><td>{{certLabel .}}</td><td>{{.Issuer}}</td><td>{{.NotAfter}}</td><td>{{if .Status}}<span class="chip sev-medium">{{certStatus .Status}}</span>{{end}}</td><td>{{.File}}{{if .Alias}} ({{.Alias}}){{end}}</td></tr>{{end}}</tbody></table></div>{{end}}
{{if .Parties.Identities}}<h3>{{t "ownIdentity"}}</h3><div class="tw"><table><thead><tr><th>{{t "certificate"}}</th><th>{{t "issuer"}}</th><th>{{t "validUntil"}}</th><th>{{t "status"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Parties.Identities}}<tr><td>{{certLabel .}}</td><td>{{.Issuer}}</td><td>{{.NotAfter}}</td><td>{{if .Status}}<span class="chip sev-medium">{{certStatus .Status}}</span>{{end}}</td><td>{{.File}}</td></tr>{{end}}</tbody></table></div>{{end}}
{{if .Parties.TrustedCAs}}<p><b>{{t "trustedCAs"}}:</b> {{range $i, $c := .Parties.TrustedCAs}}{{if $i}}; {{end}}{{$c}}{{end}}</p>{{end}}
</section>
{{if .Structure}}<section id="structure"><h2>{{t "repoStructure"}}</h2><ul>{{range .Structure}}<li>{{.}}</li>{{end}}</ul>
{{if .Copies}}<details><summary>{{t "copiesLeftOut"}} ({{len .Copies}}{{if .CopiesMore}} + {{.CopiesMore}}{{end}})</summary><div class="tw"><table><thead><tr><th>{{t "copy"}}</th><th>{{t "version"}}</th><th>{{t "mainLine"}}</th><th class="n">{{t "overlap"}}</th><th class="n">{{t "files_n"}}</th></tr></thead><tbody>
{{range .Copies}}<tr><td>{{.Path}}</td><td>{{.Version}}</td><td>{{if .Identical}}<span class="chip">= {{.Of}}</span>{{else}}{{.Of}}{{end}}</td><td class="n">{{pct100 .Overlap}}</td><td class="n">{{.Files}}</td></tr>{{end}}</tbody></table></div></details>{{end}}</section>{{end}}

<section id="context"><h2>{{t "context"}}</h2><div class="diagram">{{.ContextSVG}}</div>
<p class="note">{{t "exports"}}: <a href="context.svg">context.svg</a> · <a href="workspace.dsl">workspace.dsl</a> (Structurizr / C4) · <a href="facts.json">facts.json</a></p>
{{if .F.Deployables}}<h3>{{t "deployables"}}</h3><div class="tw"><table><thead><tr><th>{{t "module"}}</th><th>{{t "kind"}}</th><th>{{t "contextRoot"}}</th><th>{{t "appPaths"}}</th><th>{{t "includes"}}</th></tr></thead><tbody>
{{range .F.Deployables}}<tr><td>{{.Module}}</td><td>{{.Kind}}</td><td>{{.ContextRoot}}</td><td>{{join .AppPaths ", "}}</td><td>{{join .Includes ", "}}</td></tr>{{end}}</tbody></table></div>{{end}}
</section>

<section id="arch"><h2>{{t "architecture"}}</h2>
{{if .LayerSVG}}<div class="diagram">{{.LayerSVG}}</div><p class="note">{{.DiagramNote}} · <a href="architecture.svg">architecture.svg</a> · <a href="architecture.mmd">architecture.mmd</a></p>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
{{if .LayerSummary}}<h3>{{t "layers"}}</h3><div class="tw"><table><thead><tr><th>{{t "layer"}}</th><th class="n">{{t "package"}}</th><th class="n">{{t "types"}}</th></tr></thead><tbody>
{{range .LayerSummary}}<tr><td>{{t .Layer}}</td><td class="n">{{.Packages}}</td><td class="n">{{.Types}}</td></tr>{{end}}</tbody></table></div>{{end}}
<h3>{{t "layerIssues"}} ({{len .F.LayerIssues}})</h3>
{{if .F.LayerIssues}}<input class="f" data-t="tli" placeholder="{{t "filter"}}"><div class="tw"><table id="tli"><thead><tr><th>{{t "from"}}</th><th>{{t "to"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .F.LayerIssues}}<tr><td><span class="chip">{{t .FromLayer}}</span> {{.From}}</td><td><span class="chip">{{t .ToLayer}}</span> {{.To}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
<h3>{{t "cycles"}} ({{len .F.Cycles}})</h3>
{{if .F.Cycles}}<div class="tw"><table><thead><tr><th>{{t "level"}}</th><th class="n">#</th><th>{{t "member"}}</th></tr></thead><tbody>
{{range .F.Cycles}}<tr><td>{{.Level}}</td><td class="n">{{len .Members}}</td><td>{{range .Members}}<span class="chip">{{.}}</span>{{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="contracts"><h2>{{t "contracts"}} ({{.Contracts.Total}})</h2>
{{if .Contracts.Items}}<input class="f" data-t="tc" placeholder="{{t "filter"}}"><div class="tw"><table id="tc"><thead><tr><th>{{t "kind"}}</th><th>{{t "direction"}}</th><th>{{t "use"}}</th><th>{{t "key"}}</th><th>{{t "counterpart"}}</th><th>{{t "component"}}</th><th>{{t "detail"}}</th><th>{{t "transport"}}</th><th>{{t "authCol"}}</th><th>Host</th><th>{{t "confidence"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Contracts.Items}}<tr><td>{{.Kind}}</td><td>{{dir .Direction}}</td><td>{{role .Role}}</td><td><b>{{.Key}}</b></td><td>{{cp .Counterpart}}</td><td>{{.Component}}</td><td>{{.Detail}}</td><td>{{if .Transport}}<span class="chip {{if eq .Transport "plain"}}sev-medium{{end}}">{{xport .Transport}}</span>{{end}}</td><td>{{.Auth}}</td><td>{{if .Host}}<span class="chip">{{.Host}}</span>{{end}}</td><td>{{.Confidence}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div>
<p class="note">{{if .HostsSealed}}{{t "hostsSealed"}}{{else}}{{t "hostsOff"}}{{end}}</p>
{{if .Contracts.Cut}}<p class="note">{{showing (len .Contracts.Items) .Contracts.Total}}</p>{{end}}{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>


<section id="security"><h2>{{t "secAuth"}}</h2>
{{if .SecInbound}}<p><b>{{t "inboundMTLS"}}:</b> {{.SecInbound}}.</p>{{end}}{{if .SecNoUser}}<p><b>{{t "userLogin"}}:</b> {{t "noUserAuth"}}</p>{{end}}
{{if .SecMech}}<div class="tw"><table><thead><tr><th>{{t "authKind"}}</th><th>{{t "mechanism"}}</th><th class="n">{{t "files_n"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .SecMech}}<tr><td>{{.KindLabel}}</td><td><b>{{.Mechanism}}</b>{{if .Detail}} <span class="note">({{.Detail}})</span>{{end}}</td><td class="n">{{.Files}}</td><td>{{.Evidence}}</td></tr>{{end}}</tbody></table></div>
{{if .SecMTLS}}<p><span class="chip sev-info">mTLS</span> {{t "mtlsDetected"}}</p>{{end}}
{{if .SecWeak}}<h3>{{t "weakSpots"}}</h3><ul>{{range .SecWeak}}<li>{{.}}</li>{{end}}</ul>{{end}}
{{else}}<p class="note">{{t "noAuth"}}</p>{{end}}
{{if .SecExternals}}<h3>{{t "thirdParties"}} ({{len .SecExternals}})</h3><div class="tw"><table><thead><tr><th>{{t "counterpart"}}</th><th>{{t "extScope"}}</th><th>{{t "direction"}}</th><th>{{t "protocol"}}</th><th>{{t "transport"}}</th><th>{{t "authCol"}}</th><th>{{t "endpoint"}}</th><th>{{t "environments"}}</th><th class="n">Hosts</th><th>{{t "location"}}</th><th>{{t "confidence"}}</th></tr></thead><tbody>
{{range .SecExternals}}<tr><td><b>{{.Counterpart}}</b></td><td>{{.Scope}}</td><td>{{dir .Direction}}</td><td>{{.Protocol}}</td><td>{{if .Transport}}<span class="chip {{if eq .Transport "plain"}}sev-medium{{end}}">{{xport .Transport}}</span>{{end}}</td><td>{{.Auth}}</td><td>{{.Endpoint}}</td><td>{{.Envs}}</td><td class="n">{{if .Hosts}}{{.Hosts}}{{end}}</td><td>{{.Evidence}}</td><td>{{.Confidence}}</td></tr>{{end}}</tbody></table></div>{{end}}
</section>

<section id="data"><h2>{{t "data"}} ({{len .Tables}})</h2>
{{if .Tables}}<input class="f" data-t="tt" placeholder="{{t "filter"}}"><div class="tw"><table id="tt"><thead><tr><th>{{t "table"}}</th><th>{{t "access"}}</th><th class="n">{{t "count"}}</th><th>{{t "component"}}</th></tr></thead><tbody>
{{range .Tables}}<tr><td><b>{{.Table}}</b></td><td>{{.Ops}}</td><td class="n">{{.Refs}}</td><td>{{range .Components}}<span class="chip">{{.}}</span>{{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="modules"><h2>{{t "modules"}} ({{len .F.Modules}})</h2>
{{if .F.Modules}}<div class="tw"><table><thead><tr><th>{{t "module"}}</th><th>{{t "build"}}</th><th>{{t "version"}}</th><th>{{t "packaging"}}</th><th>{{t "runtime"}}</th><th>{{t "path"}}</th><th class="n">{{t "dependencies"}}</th></tr></thead><tbody>
{{range .F.Modules}}<tr><td><b>{{.Name}}</b><br><span class="note">{{.ID}}</span></td><td>{{.Build}}</td><td>{{.Version}}</td><td>{{.Packaging}}</td><td>{{.Runtime}}</td><td>{{.Descriptor}}</td><td class="n">{{len .Deps}}</td></tr>{{end}}</tbody></table></div>
<details><summary>{{t "dependencies"}} ({{len .BuildDeps}})</summary><input class="f" data-t="tbd" placeholder="{{t "filter"}}"><div class="tw"><table id="tbd"><thead><tr><th>{{t "module"}}</th><th>{{t "dependencies"}}</th><th>{{t "version"}}</th><th>{{t "scope"}}</th><th>{{t "kind"}}</th><th>{{t "resolved"}}</th></tr></thead><tbody>
{{range .BuildDeps}}<tr><td>{{.Module}}</td><td>{{.D.ID}}{{if .D.Managed}} <span class="chip">managed</span>{{end}}</td><td>{{.D.Version}}{{if and .D.Locked (ne .D.Locked .D.Version)}} → <b>{{.D.Locked}}</b> <span class="note">{{.D.LockFile}}</span>{{end}}</td><td>{{.D.Scope}}</td><td>{{.D.Kind}}</td><td>{{.D.Resolved}}</td></tr>{{end}}</tbody></table></div></details>
{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="deploy"><h2>{{t "deployment"}} ({{len .F.Deployment.Nodes}})</h2>
{{if .DeploySVG}}<div class="diagram">{{.DeploySVG}}</div><p class="note"><a href="deployment.svg">deployment.svg</a> · workspace.dsl (C4 deployment)</p>
<input class="f" data-t="tdp" placeholder="{{t "filter"}}"><div class="tw"><table id="tdp"><thead><tr><th>{{t "platform"}}</th><th>{{t "kind"}}</th><th>{{t "name"}}</th><th>{{t "images"}}</th><th>{{t "ports"}}</th><th>{{t "replicas"}}</th><th>{{t "envKeys"}}</th><th>Hosts</th><th>{{t "module"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .F.Deployment.Nodes}}<tr><td>{{.Platform}}</td><td>{{.Kind}}</td><td><b>{{.Name}}</b>{{if .Namespace}}<br><span class="note">{{.Namespace}}</span>{{end}}</td><td>{{join .Images ", "}}{{if .Base}}<span class="note">FROM {{join .Base ", "}}</span>{{end}}</td><td>{{join .Ports ", "}}</td><td>{{.Replicas}}{{.Schedule}}</td><td>{{range .EnvKeys}}<span class="chip">{{.}}</span>{{end}}</td><td>{{range .Hosts}}<span class="chip">{{.}}</span>{{end}}{{range .Paths}} <span class="note">{{.}}</span>{{end}}</td><td>{{.Module}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div>
{{if .F.Deployment.Links}}<details><summary>{{t "links"}} ({{len .F.Deployment.Links}})</summary><div class="tw"><table><thead><tr><th>{{t "from"}}</th><th>{{t "kind"}}</th><th>{{t "to"}}</th></tr></thead><tbody>
{{range .F.Deployment.Links}}<tr><td>{{index $.DeployNames .From}}</td><td>{{.Kind}}</td><td>{{index $.DeployNames .To}}</td></tr>{{end}}</tbody></table></div></details>{{end}}
<p class="note">{{t "envKeysNote"}}</p>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="tech"><h2>{{t "technologies"}} ({{len .Techs}})</h2>
{{if .Techs}}<input class="f" data-t="tte" placeholder="{{t "filter"}}"><div class="tw"><table id="tte"><thead><tr><th>{{t "kind"}}</th><th>{{t "technologies"}}</th><th>{{t "severity"}}</th><th class="n">{{t "files"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Techs}}<tr><td>{{.Category}}</td><td><b>{{.Technology}}</b></td><td><span class="sev sev-{{.Severity}}">{{t .Severity}}</span></td><td class="n">{{.Files}}</td><td>{{join .Paths ", "}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="risks"><h2>{{t "risks"}} ({{.Risks.Total}})</h2>
{{if .Risks.Items}}<input class="f" data-t="tr" placeholder="{{t "filter"}}"><div class="tw"><table id="tr"><thead><tr><th>{{t "severity"}}</th><th>{{t "rule"}}</th><th>{{t "title"}}</th><th>{{t "location"}}</th><th>{{t "lines"}}</th><th class="n">{{t "count"}}</th><th>{{t "message"}}</th></tr></thead><tbody>
{{range .Risks.Items}}<tr><td><span class="sev sev-{{.Severity}}">{{t .Severity}}</span></td><td>{{.RuleID}}</td><td>{{.Title}}{{if ne .SourceSet "main"}} <span class="chip">{{.SourceSet}}</span>{{end}}</td><td>{{.File}}</td><td>{{lines .Lines}}</td><td class="n">{{.Count}}</td><td>{{.Message}}</td></tr>{{end}}</tbody></table></div>
{{if .Risks.Cut}}<p class="note">{{showing (len .Risks.Items) .Risks.Total}}</p>{{end}}
<p class="note">findings.csv → «validation» column for confirmed / probable / discarded.</p>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

{{with .F.VulnScan}}<section id="vulns"><h2>{{t "vulns"}} ({{len $.F.Vulnerabilities}})</h2>
<p class="note">{{t "vulnSource"}} ({{.Database}}) · {{t "vulnAsOf"}} {{.AsOf}} · {{t "vulnChecked"}}: {{.Checked}}</p>
{{if $.F.Vulnerabilities}}<input class="f" data-t="tv" placeholder="{{t "filter"}}"><div class="tw"><table id="tv"><thead><tr><th>{{t "severity"}}</th><th>ID</th><th>{{t "package"}}</th><th>{{t "version"}}</th><th>{{t "fixed"}}</th><th>{{t "summary"}}</th><th>{{t "confidence"}}</th><th>{{t "module"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range $.F.Vulnerabilities}}<tr><td><span class="sev sev-{{.Severity}}">{{t .Severity}}</span>{{if .Score}} <span class="note">{{cvss .Score}}</span>{{end}}</td><td><a href="{{osv .ID}}" rel="noopener noreferrer">{{.ID}}</a>{{range .Aliases}}<br><span class="note">{{.}}</span>{{end}}{{if .Malicious}} <span class="chip">{{t "malicious"}}</span>{{end}}</td><td><b>{{.Package}}</b>{{if ne .Kind "declared"}} <span class="chip">{{.Kind}}</span>{{end}}{{if .Scope}} <span class="chip">{{.Scope}}</span>{{end}}</td><td>{{.Version}}</td><td>{{join .Fixed ", "}}</td><td>{{.Summary}}</td><td>{{.Confidence}}</td><td>{{join .Modules ", "}}</td><td>{{loc .File .Line}}{{if and .LockFile (ne .LockFile .File)}}<br><span class="note">{{.LockFile}}</span>{{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noVulns"}}</p>{{end}}
{{if .Unevaluated}}<details><summary>{{t "unevaluated"}} ({{len .Unevaluated}})</summary><input class="f" data-t="tvu" placeholder="{{t "filter"}}"><div class="tw"><table id="tvu"><thead><tr><th>{{t "package"}}</th><th>{{t "version"}}</th><th>{{t "reason"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Unevaluated}}<tr><td>{{.Package}}</td><td>{{.Version}}</td><td>{{why .Reason}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div></details>{{end}}
<p class="note">{{t "vulnNote"}} · <a href="vulnerabilities.csv">vulnerabilities.csv</a> · <a href="sbom.cdx.json">sbom.cdx.json</a> (CycloneDX 1.6)</p>
</section>{{end}}

<section id="components"><h2>{{t "components"}} ({{.Components.Total}})</h2>
{{if .Components.Items}}<input class="f" data-t="tco" placeholder="{{t "filter"}}"><div class="tw"><table id="tco"><thead><tr><th>{{t "component"}}</th><th>{{t "kind"}}</th><th>{{t "layer"}}</th><th>{{t "reason"}}</th><th>{{t "module"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Components.Items}}<tr><td><b>{{.Name}}</b><br><span class="note">{{.Package}}</span></td><td>{{.Kind}}</td><td>{{t .Layer}}</td><td>{{.LayerReason}}</td><td>{{.Module}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div>
{{if .Components.Cut}}<p class="note">{{showing (len .Components.Items) .Components.Total}}</p>{{end}}
<details><summary>{{t "dependencies"}} ({{.Deps.Total}})</summary><input class="f" data-t="tcd" placeholder="{{t "filter"}}"><div class="tw"><table id="tcd"><thead><tr><th>{{t "from"}}</th><th>{{t "to"}}</th><th>{{t "kind"}}</th><th>{{t "confidence"}}</th><th>{{t "location"}}</th></tr></thead><tbody>
{{range .Deps.Items}}<tr><td>{{.From}}</td><td>{{.To}}</td><td>{{.Kind}}</td><td>{{.Confidence}}</td><td>{{loc .File .Line}}</td></tr>{{end}}</tbody></table></div></details>
{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="metrics"><h2>{{t "metrics"}} ({{len .F.PackageStats}})</h2>
{{if .F.PackageStats}}<input class="f" data-t="tm" placeholder="{{t "filter"}}"><div class="tw"><table id="tm"><thead><tr><th>{{t "package"}}</th><th>{{t "layer"}}</th><th class="n">{{t "types"}}</th><th class="n">Ca</th><th class="n">Ce</th><th class="n">I</th></tr></thead><tbody>
{{range .F.PackageStats}}<tr><td>{{.Package}}</td><td>{{t .Layer}}</td><td class="n">{{.Types}}</td><td class="n">{{.Afferent}}</td><td class="n">{{.Efferent}}</td><td class="n">{{pct .Instability}}</td></tr>{{end}}</tbody></table></div>
<p class="note">Ca = afferent coupling, Ce = efferent coupling, I = Ce/(Ca+Ce) (R. C. Martin).</p>{{else}}<p class="note">{{t "noData"}}</p>{{end}}
</section>

<section id="languages"><h2>{{t "languages"}}</h2><div class="tw"><table><thead><tr><th>{{t "languages"}}</th><th>{{t "kind"}}</th><th class="n">{{t "files"}}</th><th class="n">{{t "code"}}</th><th class="n">{{t "comment"}}</th><th class="n">{{t "blank"}}</th></tr></thead><tbody>
{{range .F.Languages}}<tr><td>{{.Language}}</td><td>{{.Category}}</td><td class="n">{{.Files}}</td><td class="n">{{.Code}}</td><td class="n">{{.Comment}}</td><td class="n">{{.Blank}}</td></tr>{{end}}</tbody></table></div>
{{if .F.Diagnostics}}<h3>{{t "diagnostics"}} ({{len .F.Diagnostics}})</h3><div class="tw"><table><thead><tr><th>{{t "kind"}}</th><th>{{t "location"}}</th><th>{{t "message"}}</th></tr></thead><tbody>
{{range .F.Diagnostics}}<tr><td>{{.Kind}}</td><td>{{.File}}</td><td>{{.Message}}</td></tr>{{end}}</tbody></table></div>{{end}}
</section>
</main>` + foot

const portfolioTemplate = head + `
<header><h1>{{.Title}}</h1><p>{{if .Base}}{{.AreaOwn}} {{t "areaSystems"}} · {{len .P.Systems}} {{t "systems"}}{{else}}{{len .P.Systems}} {{t "systems"}}{{end}} · Traverse {{.P.Tool.Version}} · rules {{.P.Tool.RulesVersion}}</p>
{{if .Areas}}<p class="note">{{t "areas"}}: {{range $i, $a := .Areas}}{{if $i}} · {{end}}<a href="{{$a.Href}}">{{$a.Name}}</a>{{end}}</p>{{end}}{{if .Base}}<p class="note"><a href="{{.Base}}index.html">← {{t "globalPortfolio"}}</a> · {{t "areaNote"}}</p>{{end}}</header>
<nav><a href="#landscape">{{t "landscape"}}</a><a href="#systems">{{t "systems"}}</a><a href="#links">{{t "links"}}</a><a href="#shared">{{t "sharedTables"}}</a><a href="#libs">{{t "libraryVersions"}}</a><a href="#external">{{t "external"}}</a>{{if .P.VulnDB}}<a href="#vulns">{{t "vulns"}}</a>{{end}}<a href="#tech">{{t "techMatrix"}}</a></nav>
<main>
<div class="kpis">
<div class="kpi"><b>{{len .P.Systems}}</b><span>{{t "systemsCount"}}</span></div>
<div class="kpi"><b>{{k .KCode}}</b><span>{{t "code"}}</span></div>
<div class="kpi"><b>{{len .P.Links}}</b><span>{{t "linksCount"}}</span></div>
<div class="kpi"><b>{{len .P.SharedTables}}</b><span>{{t "sharedCount"}}</span></div>
<div class="kpi"><b>{{.KHigh}}</b><span>{{t "highRisks"}}</span></div>
</div>
<p class="scope">{{t "scopeNote"}}</p>
<section id="landscape"><h2>{{t "landscape"}}</h2><div class="diagram">{{.LandscapeSVG}}</div>
<p class="note">{{t "exports"}}: <a href="landscape.svg">landscape.svg</a> · <a href="workspace.dsl">workspace.dsl</a> (Structurizr / C4) · <a href="landscape.puml">landscape.puml</a> (C4-PlantUML) · <a href="landscape.mmd">landscape.mmd</a> · <a href="portfolio.json">portfolio.json</a></p></section>

<section id="systems"><h2>{{t "systems"}}</h2><input class="f" data-t="ts" placeholder="{{t "filter"}}"><div class="tw"><table id="ts"><thead><tr><th>{{t "system"}}</th><th>{{t "group"}}</th><th>{{t "languagesTop"}}</th><th class="n">{{t "code"}}</th><th class="n">{{t "modules"}}</th><th class="n">{{t "components"}}</th><th class="n">{{t "exposes"}}</th><th class="n">{{t "calls"}}</th><th class="n">{{t "tables"}}</th><th>{{t "runtime"}}</th><th>{{t "deployablesShort"}}</th><th>{{t "riskScore"}}</th><th class="n">{{t "cycles"}}</th><th class="n">{{t "layerIssues"}}</th></tr></thead><tbody>
{{range .P.Systems}}<tr><td><a href="{{$.Base}}systems/{{index $.Slugs .Name}}/report.html"><b>{{.Name}}</b></a><br><span class="note">{{.VCS}} {{.Revision}}</span></td><td>{{.Group}}</td><td>{{join .Languages ", "}}</td><td class="n">{{.Code}}</td><td class="n">{{.Modules}}</td><td class="n">{{.Components}}</td><td class="n">{{.Exposes}}</td><td class="n">{{.Calls}}</td><td class="n">{{.Tables}}</td><td>{{join .Runtimes ", "}}</td><td>{{join .Deployables ", "}}</td><td>{{sevmap .Risks}}</td><td class="n">{{.Cycles}}</td><td class="n">{{.LayerIssues}}</td></tr>{{end}}</tbody></table></div></section>

<section id="links"><h2>{{t "links"}} ({{len .P.Links}})</h2>
{{if .P.Links}}<input class="f" data-t="tl" placeholder="{{t "filter"}}"><div class="tw"><table id="tl"><thead><tr><th>{{t "from"}}</th><th>{{t "to"}}</th><th>{{t "kind"}}</th><th class="n">{{t "count"}}</th><th>{{t "confidence"}}</th><th>{{t "key"}}</th></tr></thead><tbody>
{{range .P.Links}}<tr><td>{{.From}}</td><td>{{.To}}</td><td>{{.Kind}}</td><td class="n">{{.Count}}</td><td>{{linkConf .}}</td><td>{{range .Keys}}<span class="chip">{{.}}</span>{{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}</section>

<section id="shared"><h2>{{t "sharedTables"}} ({{len .P.SharedTables}})</h2>
{{if .P.SharedTables}}<input class="f" data-t="tst" placeholder="{{t "filter"}}"><div class="tw"><table id="tst"><thead><tr><th>{{t "table"}}</th><th class="n">{{t "systems"}}</th><th>{{t "access"}}</th></tr></thead><tbody>
{{range .P.SharedTables}}<tr><td><b>{{.Table}}</b></td><td class="n">{{len .Systems}}</td><td>{{range .Access}}<span class="chip">{{.}}</span>{{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}</section>

<section id="libs"><h2>{{t "libraryVersions"}} ({{len .P.LibraryVersions}})</h2>
{{if .P.LibraryVersions}}<input class="f" data-t="tlv" placeholder="{{t "filter"}}"><div class="tw"><table id="tlv"><thead><tr><th>Library</th><th>{{t "version"}} → {{t "systems"}}</th></tr></thead><tbody>
{{range .P.LibraryVersions}}<tr><td><b>{{.Library}}</b>{{if .Internal}} <span class="chip">internal</span>{{end}}</td><td>{{range $v, $s := .Versions}}<span class="chip">{{$v}}: {{join $s ", "}}</span> {{end}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}</section>

<section id="external"><h2>{{t "external"}} ({{len .P.External}})</h2>
{{if .P.External}}<input class="f" data-t="tex" placeholder="{{t "filter"}}"><div class="tw"><table id="tex"><thead><tr><th>{{t "kind"}}</th><th>{{t "key"}}</th><th>Host</th><th>{{t "systems"}}</th></tr></thead><tbody>
{{range .P.External}}<tr><td>{{.Kind}}</td><td><b>{{.Key}}</b></td><td>{{if .Host}}<span class="chip">{{.Host}}</span>{{end}}</td><td>{{join .Systems ", "}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noData"}}</p>{{end}}</section>

{{if .P.VulnDB}}<section id="vulns"><h2>{{t "vulns"}} ({{len .P.Vulnerabilities}})</h2>
<p class="note">{{t "vulnSource"}} ({{.P.VulnDB}}) · {{t "vulnAsOf"}} {{.P.VulnAsOf}}</p>
{{if .P.Vulnerabilities}}<input class="f" data-t="tpv" placeholder="{{t "filter"}}"><div class="tw"><table id="tpv"><thead><tr><th>{{t "severity"}}</th><th>ID</th><th>{{t "package"}}</th><th>{{t "affected"}}</th><th>{{t "fixed"}}</th><th>{{t "summary"}}</th></tr></thead><tbody>
{{range .P.Vulnerabilities}}<tr><td><span class="sev sev-{{.Severity}}">{{t .Severity}}</span>{{if .Score}} <span class="note">{{cvss .Score}}</span>{{end}}</td><td><a href="{{osv .ID}}" rel="noopener noreferrer">{{.ID}}</a>{{range .Aliases}}<br><span class="note">{{.}}</span>{{end}}{{if .Malicious}} <span class="chip">{{t "malicious"}}</span>{{end}}</td><td><b>{{.Package}}</b></td><td>{{range .Affected}}<span class="chip">{{.}}</span>{{end}}</td><td>{{join .Fixed ", "}}</td><td>{{.Summary}}</td></tr>{{end}}</tbody></table></div>{{else}}<p class="note">{{t "noVulns"}}</p>{{end}}
<p class="note">{{t "vulnNote"}} · <a href="vulnerabilities.csv">vulnerabilities.csv</a></p></section>{{end}}

<section id="tech"><h2>{{t "techMatrix"}} ({{len .P.Technologies}})</h2>
{{if .P.Technologies}}<input class="f" data-t="ttm" placeholder="{{t "filter"}}"><div class="tw"><table id="ttm"><thead><tr><th>{{t "kind"}}</th><th>{{t "technologies"}}</th><th>{{t "severity"}}</th>{{range .TechSystems}}<th class="n">{{.}}</th>{{end}}</tr></thead><tbody>
{{range .P.Technologies}}{{$t := .}}<tr><td>{{.Category}}</td><td><b>{{.Technology}}</b></td><td><span class="sev sev-{{.Severity}}">{{t .Severity}}</span></td>{{range $.TechSystems}}<td class="n">{{cell $t.Systems .}}</td>{{end}}</tr>{{end}}</tbody></table></div>
<p class="note">{{t "cellsNote"}}</p>{{else}}<p class="note">{{t "noData"}}</p>{{end}}</section>
</main>` + foot
