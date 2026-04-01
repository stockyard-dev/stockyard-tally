package server
import("encoding/json";"fmt";"net/http";"strconv";"strings";"github.com/stockyard-dev/stockyard-tally/internal/store")
func(s *Server)handleListForms(w http.ResponseWriter,r *http.Request){list,_:=s.db.ListForms();if list==nil{list=[]store.Form{}};writeJSON(w,200,list)}
func(s *Server)handleGetForm(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);f,_:=s.db.GetForm(id);if f==nil{writeError(w,404,"not found");return};writeJSON(w,200,f)}
func(s *Server)handleCreateForm(w http.ResponseWriter,r *http.Request){
    if !s.limits.IsPro(){n,_:=s.db.CountForms();if n>=3{writeError(w,403,"free tier: 3 forms max");return}}
    var f store.Form;json.NewDecoder(r.Body).Decode(&f)
    if f.Name==""{writeError(w,400,"name required");return}
    if err:=s.db.CreateForm(&f);err!=nil{writeError(w,500,err.Error());return}
    writeJSON(w,201,f)}
func(s *Server)handleUpdateFields(w http.ResponseWriter,r *http.Request){
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    var fields []interface{};if err:=json.NewDecoder(r.Body).Decode(&fields);err!=nil{writeError(w,400,"invalid JSON array");return}
    b,_:=json.Marshal(fields);s.db.UpdateFormFields(id,string(b))
    writeJSON(w,200,map[string]string{"status":"updated"})}
func(s *Server)handleDeleteForm(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.DeleteForm(id);writeJSON(w,200,map[string]string{"status":"deleted"})}
func(s *Server)handleSubmit(w http.ResponseWriter,r *http.Request){
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    f,_:=s.db.GetForm(id);if f==nil{writeError(w,404,"form not found");return}
    if !f.Active{writeError(w,403,"form is not active");return}
    var data map[string]interface{};if err:=json.NewDecoder(r.Body).Decode(&data);err!=nil{writeError(w,400,"invalid JSON");return}
    b,_:=json.Marshal(data);ip:=r.RemoteAddr
    if fwd:=r.Header.Get("X-Forwarded-For");fwd!=""{ip=strings.Split(fwd,",")[0]}
    resp:=&store.Response{FormID:id,Data:string(b),IP:ip};s.db.CreateResponse(resp)
    writeJSON(w,201,map[string]string{"status":"submitted"})}
func(s *Server)handleListResponses(w http.ResponseWriter,r *http.Request){
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    limit:=100;if l:=r.URL.Query().Get("limit");l!=""{if n,err:=strconv.Atoi(l);err==nil{limit=n}}
    list,_:=s.db.ListResponses(id,limit);if list==nil{list=[]store.Response{}};writeJSON(w,200,list)}
func(s *Server)handleFormPage(w http.ResponseWriter,r *http.Request){
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    f,_:=s.db.GetForm(id);if f==nil{http.NotFound(w,r);return}
    fields,_:=store.ParseFields(f.Fields)
    w.Header().Set("Content-Type","text/html")
    fmt.Fprintf(w,`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>%s</title><style>body{background:#1a1410;color:#e8d5b0;font-family:monospace;max-width:600px;margin:2rem auto;padding:1rem}h1{color:#c4622d}label{display:block;margin-top:1rem;color:#7a6550;font-size:0.85rem}input,textarea,select{width:100%%;background:#241c15;border:1px solid #3d2e1e;color:#e8d5b0;padding:0.5rem;margin-top:0.25rem;border-radius:4px;font-family:inherit}button{background:#c4622d;color:#f5e6c8;border:none;padding:0.75rem 1.5rem;border-radius:4px;cursor:pointer;font-size:1rem;margin-top:1rem}#msg{margin-top:1rem;color:#5cb85c;display:none}</style></head><body><h1>%s</h1><p style="color:#7a6550">%s</p><form id="f">`,f.Name,f.Name,f.Description)
    for _,field:=range fields{
        fmt.Fprintf(w,`<label>%s%s</label>`,field.Label,map[bool]string{true:" *",false:""}[field.Required])
        switch field.Type{
        case"textarea":fmt.Fprintf(w,`<textarea name="%s"></textarea>`,field.Label)
        case"select":
            fmt.Fprintf(w,`<select name="%s">`,field.Label)
            for _,opt:=range field.Options{fmt.Fprintf(w,`<option>%s</option>`,opt)}
            fmt.Fprintf(w,`</select>`)
        default:fmt.Fprintf(w,`<input type="%s" name="%s">`,field.Type,field.Label)}}
    fmt.Fprintf(w,`<button type="submit">Submit</button></form><div id="msg">Submitted! Thank you.</div>`)
    fmt.Fprintf(w,`<script>document.getElementById('f').onsubmit=async function(e){e.preventDefault();var d={};new FormData(e.target).forEach(function(v,k){d[k]=v;});var r=await fetch('/api/forms/%d/submit',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(d)});if(r.ok){document.getElementById('f').style.display='none';document.getElementById('msg').style.display='block';}}</script></body></html>`,id)}
func(s *Server)handleStats(w http.ResponseWriter,r *http.Request){f,_:=s.db.CountForms();rs,_:=s.db.CountResponses();writeJSON(w,200,map[string]interface{}{"forms":f,"responses":rs})}
