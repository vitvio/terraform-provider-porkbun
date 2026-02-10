package mockbun

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"

	porkbun "github.com/kyswtn/terraform-provider-porkbun/internal/client"
)

type Server struct {
	mux         *http.ServeMux
	server      *httptest.Server
	URL         string
	nameservers map[string][]string
	dnsRecords  map[string][]porkbun.DNSRecord
	glueRecords map[string]map[string][]string
}

func New() *Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	m := &Server{
		mux:         mux,
		server:      server,
		URL:         server.URL,
		nameservers: make(map[string][]string),
		dnsRecords:  make(map[string][]porkbun.DNSRecord),
		glueRecords: make(map[string]map[string][]string),
	}

	m.addPorkbunHandlers()
	return m
}

func (m *Server) Close() {
	m.server.Close()
}

func (m *Server) SetNameservers(domain string, nameservers []string) {
	m.nameservers[domain] = nameservers
}

func (m *Server) SetDNSRecords(domain string, records []porkbun.DNSRecord) {
	m.dnsRecords[domain] = records
}

func (m *Server) SetGlueRecords(domain string, records map[string][]string) {
	m.glueRecords[domain] = records
}

func (m *Server) addPorkbunHandlers() {
	m.mux.HandleFunc("/domain/getNs/{domain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		nameservers, found := m.nameservers[domain]

		rw.Header().Set("Content-Type", "application/json")
		if !found {
			_, _ = rw.Write([]byte(`{
				"status": "FAILURE",
				"message": "Domain not found"
			}`))
			return
		}

		ns, _ := json.Marshal(nameservers)
		_, _ = rw.Write([]byte(fmt.Sprintf(`{
			"status": "SUCCESS",
			"ns": %s
		}`, ns)))
	})

	m.mux.HandleFunc("/domain/updateNs/{domain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		_, found := m.nameservers[domain]

		rw.Header().Set("Content-Type", "application/json")
		if !found {
			_, _ = rw.Write([]byte(`{
				"status": "FAILURE",
				"message": "Domain not found"
			}`))
			return
		}

		body, _ := io.ReadAll(req.Body)
		var b struct {
			Ns []string `json:"ns"`
		}
		_ = json.Unmarshal(body, &b)

		m.nameservers[domain] = b.Ns
		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	m.mux.HandleFunc("/dns/create/{domain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")

		rw.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(req.Body)
		var b porkbun.DNSRecord
		_ = json.Unmarshal(body, &b)

		if b.ID == "" {
			b.ID = strconv.Itoa(rand.Intn(1000))
		}

		if b.Name != "" {
			b.Name = fmt.Sprintf("%s.%s", b.Name, domain)
		}

		m.dnsRecords[domain] = append(m.dnsRecords[domain], b)
		_, _ = rw.Write([]byte(fmt.Sprintf(`{
			"status": "SUCCESS",
			"id": %s
		}`, b.ID)))
	})

	m.mux.HandleFunc("/dns/retrieve/{domain}/{id}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		id := req.PathValue("id")
		records, ok := m.dnsRecords[domain]

		var found porkbun.DNSRecord
		if ok {
			for _, r := range records {
				if r.ID == id {
					found = r
				}
			}
		}

		rw.Header().Set("Content-Type", "application/json")

		if found.ID == "" { // If found has not been set.
			_, _ = rw.Write([]byte(`{
				"status": "FAILURE",
				"message": "Record not found"
			}`))
			return
		}

		rs, _ := json.Marshal([]porkbun.DNSRecord{found})
		_, _ = rw.Write([]byte(fmt.Sprintf(`{
			"status": "SUCCESS",
			"records": %s
		}`, rs)))
	})

	m.mux.HandleFunc("/dns/edit/{domain}/{id}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		id := req.PathValue("id")

		rw.Header().Set("Content-Type", "application/json")

		_, ok := m.dnsRecords[domain]
		if !ok {
			_, _ = rw.Write([]byte(`{
				"status": "FAILURE",
				"message": "Record not found"
			}`))
			return
		}

		body, _ := io.ReadAll(req.Body)
		var b porkbun.DNSRecord
		_ = json.Unmarshal(body, &b)

		for _, r := range m.dnsRecords[domain] {
			if r.ID == id {
				r.Type = b.Type
				r.Content = b.Content

				if b.Name != "" {
					r.Name = fmt.Sprintf("%s.%s", b.Name, domain)
				}
			}
		}

		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	m.mux.HandleFunc("/dns/delete/{domain}/{id}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		id := req.PathValue("id")

		rw.Header().Set("Content-Type", "application/json")

		_, ok := m.dnsRecords[domain]
		if !ok {
			_, _ = rw.Write([]byte(`{
				"status": "FAILURE",
				"message": "Record not found"
			}`))
			return
		}

		for i, r := range m.dnsRecords[domain] {
			if r.ID == id {
				m.dnsRecords[domain] = append(m.dnsRecords[domain][:i], m.dnsRecords[domain][i+1:]...)
			}
		}

		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	// Glue Records Handlers

	m.mux.HandleFunc("/domain/createGlue/{domain}/{subdomain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		subdomain := req.PathValue("subdomain")

		rw.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(req.Body)
		var b struct {
			IPs []string `json:"ips"`
		}
		_ = json.Unmarshal(body, &b)

		if _, ok := m.glueRecords[domain]; !ok {
			m.glueRecords[domain] = make(map[string][]string)
		}
		m.glueRecords[domain][subdomain] = b.IPs

		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	m.mux.HandleFunc("/domain/updateGlue/{domain}/{subdomain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		subdomain := req.PathValue("subdomain")

		rw.Header().Set("Content-Type", "application/json")

		// In real world, verify existence. Here we just upsert.
		if _, ok := m.glueRecords[domain]; !ok {
			m.glueRecords[domain] = make(map[string][]string)
		}

		body, _ := io.ReadAll(req.Body)
		var b struct {
			IPs []string `json:"ips"`
		}
		_ = json.Unmarshal(body, &b)

		m.glueRecords[domain][subdomain] = b.IPs

		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	m.mux.HandleFunc("/domain/deleteGlue/{domain}/{subdomain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")
		subdomain := req.PathValue("subdomain")

		rw.Header().Set("Content-Type", "application/json")

		if sub, ok := m.glueRecords[domain]; ok {
			delete(sub, subdomain)
		}

		_, _ = rw.Write([]byte(`{
			"status": "SUCCESS"
		}`))
	})

	m.mux.HandleFunc("/domain/getGlue/{domain}", func(rw http.ResponseWriter, req *http.Request) {
		domain := req.PathValue("domain")

		rw.Header().Set("Content-Type", "application/json")

		// Return format:
		// "hosts": [ [ "ns1.domain.com", { "v4": [...], "v6": [...] } ] ]

		var hosts [][]interface{}

		if records, ok := m.glueRecords[domain]; ok {
			for sub, ips := range records {
				v4 := []string{}
				v6 := []string{}
				for _, ip := range ips {
					addr := net.ParseIP(ip)
					if addr.To4() != nil {
						v4 = append(v4, ip)
					} else {
						v6 = append(v6, ip)
					}
				}

				hostObj := map[string][]string{
					"v4": v4,
					"v6": v6,
				}
				fullHost := fmt.Sprintf("%s.%s", sub, domain)
				hosts = append(hosts, []interface{}{fullHost, hostObj})
			}
		}

		resp := struct {
			Status string          `json:"status"`
			Hosts  [][]interface{} `json:"hosts"`
		}{
			Status: "SUCCESS",
			Hosts:  hosts,
		}

		body, _ := json.Marshal(resp)
		_, _ = rw.Write(body)
	})
}
