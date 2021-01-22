{{/*
Expand the name of the chart.
*/}}
{{- define "netris-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "netris-operator.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "netris-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "netris-operator.labels" -}}
helm.sh/chart: {{ include "netris-operator.chart" . }}
{{ include "netris-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "netris-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "netris-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "netris-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "netris-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts.
*/}}
{{- define "netris-operator.namespace" -}}
{{- if .Values.namespaceOverride -}}
{{- .Values.namespaceOverride -}}
{{- else -}}
{{- .Release.Namespace -}}
{{- end -}}
{{- end -}}

{{/*
Create netris-opeator controller envs
*/}}
{{- define "netris-operator.controller.envs" -}}
- name: CONTROLLER_HOST
{{- if .Values.controller.host }}
  value: {{ .Values.controller.host | quote }}
{{- else }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.controllerCreds.host.secretName }}
      key: {{ .Values.controllerCreds.host.key }}
{{- end }}
- name: CONTROLLER_LOGIN
{{- if .Values.controller.login }}
  value: {{ .Values.controller.login | quote }}
{{- else }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.controllerCreds.login.secretName }}
      key: {{ .Values.controllerCreds.login.key }}
{{- end }}
- name: CONTROLLER_PASSWORD
{{- if .Values.controller.password }}
  value: {{ .Values.controller.password | quote }}
{{- else }}
  valueFrom:
    secretKeyRef:
      name: {{ .Values.controllerCreds.password.secretName }}
      key: {{ .Values.controllerCreds.password.key }}
{{- end }}
{{- if or (eq (lower (toString .Values.controller.insecure )) "true") (eq (lower (toString .Values.controller.insecure )) "false")  }}
- name: CONTROLLER_INSECURE
  value: {{ .Values.controller.insecure | quote }}
{{- end -}}
{{- end -}}
