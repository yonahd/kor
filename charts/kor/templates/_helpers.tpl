{{/*
Expand the name of the chart.
*/}}
{{- define "kor.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kor.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "kor.namespace" -}}
{{- if .Values.namespaceOverride }}
{{- .Values.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kor.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kor.labels" -}}
helm.sh/chart: {{ include "kor.chart" . }}
{{ include "kor.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.additionalLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kor.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kor.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kor.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kor.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Check if Role should be created
*/}}
{{- define "kor.createRole" -}}
{{- $create := .Values.rbac.create -}}
{{- if or (eq (toString $create) "true") (eq $create "role") }}true{{- end }}
{{- end }}

{{/*
Check if ClusterRole should be created
*/}}
{{- define "kor.createClusterRole" -}}
{{- $create := .Values.rbac.create -}}
{{- if or (eq (toString $create) "true") (eq $create "clusterrole") }}true{{- end }}
{{- end }}

{{/*
Generate resource rules
*/}}
{{- define "kor.resourceRules" -}}
{{- range .Values.rbac.rules }}
- apiGroups:
{{- range .apiGroups }}
    - {{ . | quote }}
{{- end }}
  resources:
    {{- toYaml .resources | nindent 4 }}
  verbs:
  {{- if .verbs }}
    {{- toYaml .verbs | nindent 4 }}
  {{- else }}
    - get
    - list
    - watch
  {{- end }}
{{- end }}
{{- end }}

{{/*
Build the kor CLI args from the user-provided args and the typed config.
Args passed via `args` are preserved as-is. Config values that differ from
the kor CLI defaults are appended as `--flag=value` entries, giving the
structured config the final say while keeping existing `args` working.
*/}}
{{- define "kor.buildArgs" -}}
{{- $args := list -}}
{{- range .args | default list -}}
{{- $args = append $args . -}}
{{- end -}}
{{- $config := .config | default dict -}}
{{- if $config.kubeconfig }}{{- $args = append $args (printf "--kubeconfig=%s" $config.kubeconfig) -}}{{- end -}}
{{- if and $config.output (ne $config.output "table") }}{{- $args = append $args (printf "--output=%s" $config.output) -}}{{- end -}}
{{- if $config.clusterName }}{{- $args = append $args (printf "--cluster-name=%s" $config.clusterName) -}}{{- end -}}
{{- if $config.delete }}{{- $args = append $args "--delete" -}}{{- end -}}
{{- if $config.noInteractive }}{{- $args = append $args "--no-interactive" -}}{{- end -}}
{{- if $config.verbose }}{{- $args = append $args "--verbose" -}}{{- end -}}
{{- if $config.showReason }}{{- $args = append $args "--show-reason" -}}{{- end -}}
{{- if and $config.groupBy (ne $config.groupBy "namespace") }}{{- $args = append $args (printf "--group-by=%s" $config.groupBy) -}}{{- end -}}
{{- with $config.excludeLabels }}{{- $args = append $args (printf "--exclude-labels=%s" (join "," .)) -}}{{- end -}}
{{- if $config.includeLabels }}{{- $args = append $args (printf "--include-labels=%s" $config.includeLabels) -}}{{- end -}}
{{- with $config.excludeNamespaces }}{{- $args = append $args (printf "--exclude-namespaces=%s" (join "," .)) -}}{{- end -}}
{{- with $config.includeNamespaces }}{{- $args = append $args (printf "--include-namespaces=%s" (join "," .)) -}}{{- end -}}
{{- if $config.ignoreOwnerReferences }}{{- $args = append $args "--ignore-owner-references" -}}{{- end -}}
{{- if $config.newerThan }}{{- $args = append $args (printf "--newer-than=%s" $config.newerThan) -}}{{- end -}}
{{- if $config.olderThan }}{{- $args = append $args (printf "--older-than=%s" $config.olderThan) -}}{{- end -}}
{{- with $config.resources }}{{- $args = append $args (printf "--resources=%s" (join "," .)) -}}{{- end -}}
{{- if kindIs "bool" $config.namespaced }}{{- $args = append $args (printf "--namespaced=%t" $config.namespaced) -}}{{- end -}}
{{- toYaml $args -}}
{{- end }}
