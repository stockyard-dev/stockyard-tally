package main
import ("fmt";"log";"net/http";"os";"github.com/stockyard-dev/stockyard-tally/internal/server";"github.com/stockyard-dev/stockyard-tally/internal/store")
func main(){port:=os.Getenv("PORT");if port==""{port="8640"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./tally-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("tally: %v",err)};defer db.Close();srv:=server.New(db,server.DefaultLimits())
fmt.Printf("\n  Tally — Self-hosted counter and gauge API\n  ─────────────────────────────────\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Data:       %s\n  ─────────────────────────────────\n  Questions? hello@stockyard.dev\n\n",port,port,dataDir)
log.Printf("tally: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
