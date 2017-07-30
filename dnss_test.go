// End to end tests.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"blitiri.com.ar/go/dnss/internal/dnstohttps"
	"blitiri.com.ar/go/dnss/internal/httpstodns"
	"blitiri.com.ar/go/dnss/internal/testutil"
	"github.com/golang/glog"
	"github.com/miekg/dns"
)

// Setup:
// DNS client -> DNS-to-HTTPS -> HTTPS-to-DNS -> DNS server
//
// The DNS client will be created on each test.
// The DNS server will be created below, and the tests can adjust its
// responses as needed.

// Address of the DNS-to-HTTPS server, for the tests to use.
var ServerAddr string

// realMain is the real main function, which returns the value to pass to
// os.Exit(). We have to do this so we can use defer.
func realMain(m *testing.M) int {
	flag.Parse()
	defer glog.Flush()

	DNSToHTTPSAddr := testutil.GetFreePort()
	HTTPSToDNSAddr := testutil.GetFreePort()
	DNSServerAddr := testutil.GetFreePort()

	// We want tests talking to the DNS-to-HTTPS server, the first in the
	// chain.
	ServerAddr = DNSToHTTPSAddr

	// DNS to HTTPS server.
	r := dnstohttps.NewHTTPSResolver("http://"+HTTPSToDNSAddr+"/resolve", "")
	dtoh := dnstohttps.New(DNSToHTTPSAddr, r, "")
	go dtoh.ListenAndServe()

	// HTTPS to DNS server.
	htod := httpstodns.Server{
		Addr:     HTTPSToDNSAddr,
		Upstream: DNSServerAddr,
	}
	httpstodns.InsecureForTesting = true
	go htod.ListenAndServe()

	// Fake DNS server.
	go ServeFakeDNSServer(DNSServerAddr)

	// Wait for the servers to start up.
	err1 := testutil.WaitForDNSServer(DNSToHTTPSAddr)
	err2 := testutil.WaitForHTTPServer(HTTPSToDNSAddr)
	err3 := testutil.WaitForDNSServer(DNSServerAddr)
	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Printf("Error waiting for the test servers to start:\n")
		fmt.Printf("  DNS to HTTPS: %v\n", err1)
		fmt.Printf("  HTTPS to DNS: %v\n", err2)
		fmt.Printf("  DNS server:   %v\n", err3)
		fmt.Printf("Check the INFO logs for more details\n")
		return 1
	}
	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(realMain(m))
}

// Fake DNS server.
func ServeFakeDNSServer(addr string) {
	server := &dns.Server{
		Addr:    addr,
		Handler: dns.HandlerFunc(handleFakeDNS),
		Net:     "udp",
	}
	err := server.ListenAndServe()
	panic(err)
}

// DNS answers to give, as a map of "name type" -> []RR.
// Tests will modify this according to their needs.
var answers map[string][]dns.RR
var answersMu sync.Mutex

func resetAnswers() {
	answersMu.Lock()
	answers = map[string][]dns.RR{}
	answersMu.Unlock()
}

func addAnswers(tb testing.TB, zone string) {
	for x := range dns.ParseZone(strings.NewReader(zone), "", "") {
		if x.Error != nil {
			tb.Fatalf("error parsing zone: %v\n", x.Error)
			return
		}

		hdr := x.RR.Header()
		key := fmt.Sprintf("%s %d", hdr.Name, hdr.Rrtype)
		answersMu.Lock()
		answers[key] = append(answers[key], x.RR)
		answersMu.Unlock()
	}
}

func handleFakeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)

	if len(r.Question) != 1 {
		w.WriteMsg(m)
		return
	}

	q := r.Question[0]
	if testing.Verbose() {
		fmt.Printf("fake dns <- %v\n", q)
	}

	key := fmt.Sprintf("%s %d", q.Name, q.Qtype)
	answersMu.Lock()
	if rrs, ok := answers[key]; ok {
		m.Answer = rrs
	} else {
		m.Rcode = dns.RcodeNameError
	}
	answersMu.Unlock()

	if testing.Verbose() {
		fmt.Printf("fake dns -> %v | %v\n",
			dns.RcodeToString[m.Rcode], m.Answer)
	}
	w.WriteMsg(m)
}

//
// Tests
//

func TestSimple(t *testing.T) {
	resetAnswers()
	addAnswers(t, "test.blah. A 1.2.3.4")
	_, ans, err := testutil.DNSQuery(ServerAddr, "test.blah.", dns.TypeA)
	if err != nil {
		t.Errorf("dns query returned error: %v", err)
	}
	if ans.(*dns.A).A.String() != "1.2.3.4" {
		t.Errorf("unexpected result: %q", ans)
	}

	addAnswers(t, "test.blah. MX 10 mail.test.blah.")
	_, ans, err = testutil.DNSQuery(ServerAddr, "test.blah.", dns.TypeMX)
	if err != nil {
		t.Errorf("dns query returned error: %v", err)
	}
	if ans.(*dns.MX).Mx != "mail.test.blah." {
		t.Errorf("unexpected result: %q", ans.(*dns.MX).Mx)
	}

	in, _, err := testutil.DNSQuery(ServerAddr, "unknown.", dns.TypeA)
	if err != nil {
		t.Errorf("dns query returned error: %v", err)
	}
	if in.Rcode != dns.RcodeNameError {
		t.Errorf("unexpected result: %q", in)
	}
}

//
// Benchmarks
//

func BenchmarkSimple(b *testing.B) {
	resetAnswers()
	addAnswers(b, "test.blah. A 1.2.3.4")
	b.ResetTimer()

	var err error
	for i := 0; i < b.N; i++ {
		_, _, err = testutil.DNSQuery(ServerAddr, "test.blah.", dns.TypeA)
		if err != nil {
			b.Errorf("dns query returned error: %v", err)
		}
	}
}
