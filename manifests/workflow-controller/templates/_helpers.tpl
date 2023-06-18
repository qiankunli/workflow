{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "common.names.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "common.names.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "common.labels.standard" -}}
app.kubernetes.io/name: {{ include "common.names.name" . }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/instance: {{ .Release.Name }}
    {{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
    {{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
    {{- if .Values.labels }}
{{toYaml .Values.labels}}
    {{- end }}
{{- end -}}

{{/*
matchLabels
*/}}
{{- define "common.labels.matchLabels" -}}
app.kubernetes.io/name: {{ include "common.names.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Return the proper image name
*/}}
{{- define "common.images.image" -}}
    {{- $repositoryName := .Values.platformConfig.imageRepositoryRelease -}}
    {{- $imageName := .Values.image.name -}}
    {{- $tag := (default .Chart.AppVersion .Values.image.tag) | toString -}}
    {{- if .Values.platformConfig.imageRegistry -}}
        {{- $registryName := .Values.platformConfig.imageRegistry -}}
        {{- printf "%s/%s/%s:%s" $registryName $repositoryName $imageName $tag -}}
    {{- else -}}
        {{- printf "%s/%s:%s" $repositoryName $imageName $tag -}}
    {{- end -}}
{{- end -}}
