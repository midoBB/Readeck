package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/readeck/readeck/assets"
	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/internal/server"
)

type ctxNameKey struct{}

var rndIcons = []string{}

func init() {
	f := assets.StaticFilesFS()
	fp, err := f.Open("rndicons.json")
	if err != nil {
		panic(err)
	}

	d := json.NewDecoder(fp)
	if err = d.Decode(&rndIcons); err != nil {
		panic(err)
	}
}

const svgGradient = `<?xml version="1.0" encoding="UTF-8"?>` +
	`<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 256 160" width="256" height="160">` +
	`<defs>` +
	`<linearGradient id="gradient" x1="30%%" x2="70%%" y2="100%%">` +
	`<stop stop-color="hsl(%d, 20%%, 80%%)" offset="0"/>` +
	`<stop stop-color="hsl(%d, 20%%, 83%%)" offset="1"/>` +
	`</linearGradient>` +
	`</defs>` +
	`<rect width="100%%" height="100%%" fill="url(#gradient)"/>` +
	`<svg preserveAspectRatio="xMinYMin meet" x="130" y="35" viewBox="0 0 24 24" fill="#ffffff55">` +
	`%s` +
	`</svg>` +
	`</svg>`

type hashCode int

func (c hashCode) GetSumStrings() []string {
	return []string{fmt.Sprintf("%d", c)}
}

func (c hashCode) GetLastModified() []time.Time {
	return []time.Time{configs.BuildTime()}
}

// randomSvg sends an SVG image with a gradient. The gradient's color
// is based on the name.
func randomSvg(s *server.Server) http.Handler {
	r := chi.NewRouter()
	nbIcons := len(rndIcons)

	withHashCode := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := chi.URLParam(r, "name")
			data := uint64(0)
			for _, b := range []byte(name) {
				data = (data << 8) | uint64(b)
			}
			hc := hashCode(data) % 360
			ctx := context.WithValue(r.Context(), ctxNameKey{}, hc)

			s.WriteEtag(w, hc)
			s.WriteLastModified(w, hc)
			s.WithCaching(next).ServeHTTP(w, r.WithContext(ctx))
		})
	}

	r.With(withHashCode).Get("/", func(w http.ResponseWriter, r *http.Request) {
		hcode := r.Context().Value(ctxNameKey{}).(hashCode)

		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprintf(w, svgGradient, hcode, hcode, rndIcons[int(hcode)%nbIcons])
	})
	return r
}
