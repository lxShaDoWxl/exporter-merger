package cmd

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"
)

const (
	contentTypeHeader      = "Content-Type"
	contentEncodingHeader  = "Content-Encoding"
	acceptEncodingHeader   = "Accept-Encoding"
	processStartTimeHeader = "Process-Start-Time-Unix"
)

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

type Handler struct {
	Exporters            []Exporter
	ExportersHTTPTimeout int
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{
		"RequestURI": r.RequestURI,
		"UserAgent":  r.UserAgent(),
	}).Debug("handling new request")
	h.Merge(w, r)
}

func (h Handler) Merge(rsp http.ResponseWriter, req *http.Request) {
	mfs := map[string]*prom.MetricFamily{}
	mfMutex := sync.Mutex{}
	httpClientTimeout := time.Second * time.Duration(h.ExportersHTTPTimeout)

	wg := sync.WaitGroup{}
	for _, exporter := range h.Exporters {
		wg.Add(1)
		go func(exporter Exporter) {
			defer wg.Done()
			url := exporter.URL
			log.WithField("url", url).Debug("getting remote metrics")
			httpClient := http.Client{Timeout: httpClientTimeout}
			reqExport, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			if err != nil {
				log.WithField("url", url).Errorf("HTTP connection failed: %v", err)
				return
			}
			for key, val := range exporter.SetHeaders {
				reqExport.Header.Set(key, val)
			}
			resp, err := httpClient.Do(reqExport)
			if err != nil {
				log.WithField("url", url).Errorf("HTTP connection failed: %v", err)
				return
			}
			defer resp.Body.Close()

			tp := new(expfmt.TextParser)
			part, err := tp.TextToMetricFamilies(resp.Body)
			if err != nil {
				log.WithField("url", url).Errorf("Parse response body to metrics: %v", err)
				return
			}
			for n, mf := range part {
				for i, metric := range mf.Metric {
					mf.Metric[i].Label = append(metric.Label, exporter.AddLabels...)
				}
				mfMutex.Lock()
				mfo, ok := mfs[n]
				if ok {
					mfo.Metric = append(mfo.Metric, mf.Metric...)
				} else {
					mfs[n] = mf
				}
				mfMutex.Unlock()
			}
		}(exporter)
	}
	wg.Wait()
	var contentType expfmt.Format
	contentType = expfmt.Negotiate(req.Header)
	header := rsp.Header()
	header.Set(contentTypeHeader, string(contentType))
	w := io.Writer(rsp)

	if gzipAccepted(req.Header) {
		header.Set(contentEncodingHeader, "gzip")
		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)

		gz.Reset(w)
		defer gz.Close()

		w = gz
	}

	enc := expfmt.NewEncoder(w, contentType)
	for mf := range mfs {
		err := enc.Encode(mfs[mf])
		if err != nil {
			log.Error(err)
			return
		}
	}
	if closer, ok := enc.(expfmt.Closer); ok {
		// This in particular takes care of the final "# EOF\n" line for OpenMetrics.
		if err := closer.Close(); err != nil {
			log.Error(err)
			return
		}
	}

}
func gzipAccepted(header http.Header) bool {
	a := header.Get(acceptEncodingHeader)
	parts := strings.Split(a, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return true
		}
	}
	return false
}
