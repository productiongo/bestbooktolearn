package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// Site implements the methods we need to run a BestBookToLearn
// HTTP server. This cannot be instantiated directly from outside
// this package, and should instead be created via the NewSite function.
type Site struct {
	Handler         *http.ServeMux
	GracefulTimeout time.Duration

	templateDir string
	staticDir   string

	topics    []string
	templates map[string]*template.Template
}

// NewSite returns a new Site with a multiplexer for handling
// requests. It also pregenerates routes for the given
// topics.
func NewSite(topics []string, templateDir, staticDir string) (*Site, error) {
	s := &Site{
		GracefulTimeout: 5 * time.Second,
		templateDir:     templateDir,
		staticDir:       staticDir,
		topics:          topics,
	}

	// load in HTML templates
	err := s.initTemplates()
	if err != nil {
		return nil, err
	}

	// initialize route handlers
	s.initHandlers(topics)

	return s, nil
}

func (s *Site) initHandlers(topics []string) {
	// define the multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.HomeHandler)
	mux.HandleFunc("/about", s.AboutHandler)
	s.Handler = mux

	// initialize static file server
	s.initStatic()

	// add additional topic paths on server start
	for _, t := range topics {
		s.AddTopic(t)
	}
}

func (s *Site) initTemplates() error {
	s.templates = map[string]*template.Template{}
	templates := []string{
		"home",
		"about",
		"topic",
		"404",
	}
	baseTmpl := filepath.Join(s.templateDir, "base.html")
	for _, tmpl := range templates {
		fp := filepath.Join(s.templateDir, tmpl+".html")
		t, err := template.ParseFiles(fp, baseTmpl)
		if err != nil {
			return err
		}
		s.templates[tmpl] = t
	}
	return nil
}

func (s Site) initStatic() {
	fs := http.FileServer(http.Dir(s.staticDir))
	s.Handler.Handle("/static/", http.StripPrefix("/"+s.staticDir+"/", fs))
}

// AddTopic adds a route to a specific topic
func (s Site) AddTopic(t string) {
	s.Handler.HandleFunc("/"+t+"/", s.TopicHandler)
}

func (s Site) render(w io.Writer, name string, data interface{}) {
	err := s.templates[name].ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Println("ERROR:", err)
	}
}

// ListenAndServe starts an HTTP server for the Site
// listening on the provided address. It provides sensible
// http.Server default values and automatically handles
// graceful shutdowns.
func (s Site) ListenAndServe(addr string) {
	server := &http.Server{
		Addr:           addr,
		Handler:        s.Handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("Listening on %s\n", server.Addr)

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	s.gracefulShutdown(server)
}

// gracefulShutdown provides a graceful shutdown procedure
// which waits for pending requests to finish before stopping the
// server
func (s Site) gracefulShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), s.GracefulTimeout)
	defer cancel()

	log.Printf("\nShutting down gracefully with %s timeout\n", s.GracefulTimeout)
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error: %v\n", err)
	}
}

// HomeHandler handles a request to the home page.
func (s Site) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"topics": s.topics,
	}
	s.render(w, "home", data)
}

// TopicHandler handles a request for a topic detail page
func (s Site) TopicHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"topic": r.URL.EscapedPath(),
	}
	s.render(w, "topic", data)
}

// AboutHandler handles a request to the about page.
func (s Site) AboutHandler(w http.ResponseWriter, r *http.Request) {
	s.render(w, "about", nil)
}

// main is the entrypoint for starting a new BestBookToLearn server
func main() {
	addr := ":8080"

	topics := []string{
		"production-go",
		"linux",
		"docker",
		"discrete-mathematics",
		"competitive-programming",
	}
	site, err := NewSite(topics, "templates", "static")
	if err != nil {
		panic(err)
	}
	site.ListenAndServe(addr)
}
