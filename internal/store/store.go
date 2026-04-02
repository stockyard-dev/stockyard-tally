package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Counter struct{ID string `json:"id"`;Name string `json:"name"`;Namespace string `json:"namespace,omitempty"`;Value int64 `json:"value"`;Description string `json:"description,omitempty"`;CreatedAt string `json:"created_at"`;UpdatedAt string `json:"updated_at"`}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"tally.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS counters(id TEXT PRIMARY KEY,name TEXT NOT NULL,namespace TEXT DEFAULT 'default',value INTEGER DEFAULT 0,description TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')),updated_at TEXT DEFAULT(datetime('now')),UNIQUE(name,namespace))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(c *Counter)error{c.ID=genID();c.CreatedAt=now();c.UpdatedAt=c.CreatedAt;if c.Namespace==""{c.Namespace="default"}
_,err:=d.db.Exec(`INSERT INTO counters VALUES(?,?,?,?,?,?,?)`,c.ID,c.Name,c.Namespace,c.Value,c.Description,c.CreatedAt,c.UpdatedAt);return err}
func(d *DB)Get(name,ns string)*Counter{if ns==""{ns="default"};var c Counter
if d.db.QueryRow(`SELECT id,name,namespace,value,description,created_at,updated_at FROM counters WHERE name=? AND namespace=?`,name,ns).Scan(&c.ID,&c.Name,&c.Namespace,&c.Value,&c.Description,&c.CreatedAt,&c.UpdatedAt)!=nil{return nil};return &c}
func(d *DB)GetByID(id string)*Counter{var c Counter;if d.db.QueryRow(`SELECT id,name,namespace,value,description,created_at,updated_at FROM counters WHERE id=?`,id).Scan(&c.ID,&c.Name,&c.Namespace,&c.Value,&c.Description,&c.CreatedAt,&c.UpdatedAt)!=nil{return nil};return &c}
func(d *DB)List(ns string)[]Counter{q:=`SELECT id,name,namespace,value,description,created_at,updated_at FROM counters`;args:=[]any{}
if ns!=""&&ns!="all"{q+=` WHERE namespace=?`;args=append(args,ns)};q+=` ORDER BY namespace,name`
rows,_:=d.db.Query(q,args...);if rows==nil{return nil};defer rows.Close()
var o []Counter;for rows.Next(){var c Counter;rows.Scan(&c.ID,&c.Name,&c.Namespace,&c.Value,&c.Description,&c.CreatedAt,&c.UpdatedAt);o=append(o,c)};return o}
func(d *DB)Increment(name,ns string,by int64)*Counter{if ns==""{ns="default"};t:=now()
c:=d.Get(name,ns);if c==nil{id:=genID();d.db.Exec(`INSERT INTO counters VALUES(?,?,?,?,?,?,?)`,id,name,ns,by,"",t,t);return d.GetByID(id)}
d.db.Exec(`UPDATE counters SET value=value+?,updated_at=? WHERE id=?`,by,t,c.ID);return d.Get(name,ns)}
func(d *DB)Set(name,ns string,val int64)*Counter{if ns==""{ns="default"};t:=now()
c:=d.Get(name,ns);if c==nil{id:=genID();d.db.Exec(`INSERT INTO counters VALUES(?,?,?,?,?,?,?)`,id,name,ns,val,"",t,t);return d.GetByID(id)}
d.db.Exec(`UPDATE counters SET value=?,updated_at=? WHERE id=?`,val,t,c.ID);return d.Get(name,ns)}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM counters WHERE id=?`,id);return err}
func(d *DB)Namespaces()[]string{rows,_:=d.db.Query(`SELECT DISTINCT namespace FROM counters ORDER BY namespace`);if rows==nil{return nil};defer rows.Close();var o []string;for rows.Next(){var n string;rows.Scan(&n);o=append(o,n)};return o}
type Stats struct{Counters int `json:"counters"`;Namespaces int `json:"namespaces"`}
func(d *DB)Stats()Stats{var s Stats;d.db.QueryRow(`SELECT COUNT(*) FROM counters`).Scan(&s.Counters);s.Namespaces=len(d.Namespaces());return s}
