package server
import ("encoding/json";"log";"net/http";"strconv";"github.com/stockyard-dev/stockyard-tally/internal/store")
type Server struct{db *store.DB;mux *http.ServeMux}
func New(db *store.DB)*Server{s:=&Server{db:db,mux:http.NewServeMux()}
s.mux.HandleFunc("GET /api/counters",s.list);s.mux.HandleFunc("POST /api/counters",s.create);s.mux.HandleFunc("GET /api/counters/{id}",s.get);s.mux.HandleFunc("DELETE /api/counters/{id}",s.del)
s.mux.HandleFunc("POST /api/increment",s.increment);s.mux.HandleFunc("POST /api/set",s.set);s.mux.HandleFunc("GET /api/value",s.value)
s.mux.HandleFunc("GET /api/namespaces",s.namespaces)
s.mux.HandleFunc("GET /api/stats",s.stats);s.mux.HandleFunc("GET /api/health",s.health)
s.mux.HandleFunc("GET /ui",s.dashboard);s.mux.HandleFunc("GET /ui/",s.dashboard);s.mux.HandleFunc("GET /",s.root);return s}
func(s *Server)ServeHTTP(w http.ResponseWriter,r *http.Request){s.mux.ServeHTTP(w,r)}
func wj(w http.ResponseWriter,c int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(c);json.NewEncoder(w).Encode(v)}
func we(w http.ResponseWriter,c int,m string){wj(w,c,map[string]string{"error":m})}
func(s *Server)root(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};http.Redirect(w,r,"/ui",302)}
func(s *Server)list(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"counters":oe(s.db.List(r.URL.Query().Get("namespace")))})}
func(s *Server)create(w http.ResponseWriter,r *http.Request){var c store.Counter;json.NewDecoder(r.Body).Decode(&c);if c.Name==""{we(w,400,"name required");return};s.db.Create(&c);wj(w,201,s.db.GetByID(c.ID))}
func(s *Server)get(w http.ResponseWriter,r *http.Request){c:=s.db.GetByID(r.PathValue("id"));if c==nil{we(w,404,"not found");return};wj(w,200,c)}
func(s *Server)del(w http.ResponseWriter,r *http.Request){s.db.Delete(r.PathValue("id"));wj(w,200,map[string]string{"deleted":"ok"})}
func(s *Server)increment(w http.ResponseWriter,r *http.Request){var req struct{Name string `json:"name"`;Namespace string `json:"namespace"`;By int64 `json:"by"`};json.NewDecoder(r.Body).Decode(&req)
if req.Name==""{we(w,400,"name required");return};if req.By==0{req.By=1};c:=s.db.Increment(req.Name,req.Namespace,req.By);wj(w,200,c)}
func(s *Server)set(w http.ResponseWriter,r *http.Request){var req struct{Name string `json:"name"`;Namespace string `json:"namespace"`;Value int64 `json:"value"`};json.NewDecoder(r.Body).Decode(&req)
if req.Name==""{we(w,400,"name required");return};c:=s.db.Set(req.Name,req.Namespace,req.Value);wj(w,200,c)}
func(s *Server)value(w http.ResponseWriter,r *http.Request){name:=r.URL.Query().Get("name");ns:=r.URL.Query().Get("namespace");c:=s.db.Get(name,ns)
if c==nil{wj(w,200,map[string]any{"name":name,"value":0});return};wj(w,200,map[string]any{"name":c.Name,"value":c.Value})}
func(s *Server)namespaces(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"namespaces":oe(s.db.Namespaces())})}
func(s *Server)stats(w http.ResponseWriter,r *http.Request){wj(w,200,s.db.Stats())}
func(s *Server)health(w http.ResponseWriter,r *http.Request){st:=s.db.Stats();wj(w,200,map[string]any{"status":"ok","service":"tally","counters":st.Counters})}
func oe[T any](s []T)[]T{if s==nil{return[]T{}};return s}
func init(){log.SetFlags(log.LstdFlags|log.Lshortfile)};var _=strconv.Atoi
