package cmd

import (
	"io"
	"net/http"
	"sync"
	"time"

	prom "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	Exporters            []Exporter
	ExportersHTTPTimeout int
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{
		"RequestURI": r.RequestURI,
		"UserAgent":  r.UserAgent(),
	}).Debug("handling new request")
	h.Merge(w)
}

func (h Handler) Merge(w io.Writer) {
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
			resp, err := httpClient.Get(url)
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
	enc := expfmt.NewEncoder(w, expfmt.FmtText)
	for mf := range mfs {
		err := enc.Encode(mfs[mf])
		if err != nil {
			log.Error(err)
			return
		}
	}
}
