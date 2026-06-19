{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

{{- range .MethodSets}}
const Operation{{$svrType}}{{.OriginalName}} = "/{{$svrName}}/{{.OriginalName}}"
{{- end}}

type {{.ServiceType}}HTTPServer interface {
{{- range .MethodSets}}
	{{- if ne .Comment ""}}
	{{.Comment}}
	{{- end}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
{{- end}}
}

func Register{{.ServiceType}}HTTPServer(srv binding.Router, svc {{.ServiceType}}HTTPServer) {
{{- range .Methods}}
	srv.Handle("{{.Method}}", "{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_HTTP_Handler(svc))
{{- end}}
}

{{range .Methods}}
func _{{$svrType}}_{{.Name}}{{.Num}}_HTTP_Handler(svc {{$svrType}}HTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in {{.Request}}
		{{- if .HasBody}}
			{{- if eq .BodyField "*"}}
		if err := binding.BindBody(r, &in); err != nil {
			binding.WriteError(w, err)
			return
		}
			{{- else}}
		if err := binding.BindBodyField(r, &in, "{{.BodyField}}"); err != nil {
			binding.WriteError(w, err)
			return
		}
		if err := binding.BindQuery(&in, r.URL.Query()); err != nil {
			binding.WriteError(w, err)
			return
		}
			{{- end}}
		{{- else}}
		if err := binding.BindQuery(&in, r.URL.Query()); err != nil {
			binding.WriteError(w, err)
			return
		}
		{{- end}}
		{{- if .HasVars}}
		if err := binding.BindAllPaths(&in, r, {{.PathVarsList}}); err != nil {
			binding.WriteError(w, err)
			return
		}
		{{- end}}
		out, err := svc.{{.Name}}(r.Context(), &in)
		if err != nil {
			binding.WriteError(w, err)
			return
		}
		{{- if .ResponseBody}}
		binding.WriteResponse(w, r, out{{.ResponseBody}})
		{{- else}}
		binding.WriteResponse(w, r, out)
		{{- end}}
	}
}
{{end}}
