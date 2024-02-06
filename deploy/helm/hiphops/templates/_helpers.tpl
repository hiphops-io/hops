{{/*
This looping pattern for getting the directories was adapted from:
https://github.com/helm/helm/issues/4157#issuecomment-490748085
*/}}

{{- define "hiphops.path" }}
{{- printf "/home/hops/%s" . | clean }}
{{- end }}

{{- define "hiphops.automationConfigMaps" }}
{{/* This dict tracks which directories have already been added */}}
{{- $processedDict := dict -}}
{{/* Loop over all the directories in the given automation path */}}
{{- range $path, $bytes := .Files.Glob (printf "%s/**" .Values.hiphops.automationsPath) -}}
{{- $name := base (dir $path) }}
{{- $automationDir := dir $path }}
{{/* If we haven't processed this directory before */}}
{{- if not (hasKey $processedDict $name) }}
{{/* then process it starting by marking it as processed */}}
{{ $_ := set $processedDict $name "true" }}
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: {{ $.Values.namespace }}
{{- /* Make each configmap's name from the automation name */}}
  name: hops-automation-{{ $name }}
data:
{{ ($.Files.Glob (printf "%s/*" $automationDir)).AsConfig | indent 2 }}
---
{{- end }}
{{- end }}
{{- end }}

{{- define "hiphops.automationName" -}}
hops-automation-{{ . }}
{{- end -}}

{{- define "hiphops.automationVolumes" }}
{{- $processedDict := dict }}
{{- range $path, $bytes := .Files.Glob (printf "%s/**" .Values.hiphops.automationsPath) }}
{{- $name := base (dir $path) }}
{{- $automationDir := dir $path }}
{{- if not (hasKey $processedDict $name) }}
{{- $_ := set $processedDict $name "true" }}
- name: {{ include "hiphops.automationName" $name }}
  configMap:
    name: {{ include "hiphops.automationName" $name }}
{{- end }}
{{- end }}
{{- end }}

{{- define "hiphops.automationVolumeMounts"}}
{{- $processedDict := dict }}
{{- range $path, $bytes := .Files.Glob (printf "%s/**" .Values.hiphops.automationsPath) }}
{{- $name := base (dir $path) }}
{{- $automationDir := dir $path }}
{{- if not (hasKey $processedDict $name) }}
{{- $_ := set $processedDict $name "true" }}
- name: {{ include "hiphops.automationName" $name }}
  mountPath: {{ include "hiphops.path" (printf "hops-conf/%s" $name) }}
{{- end }}
{{- end }}
{{- end }}
