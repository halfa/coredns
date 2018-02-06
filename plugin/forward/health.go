package forward

import (
	"sync/atomic"

	"github.com/miekg/dns"
)

// For HC we send to . IN NS +norec message to the upstream. Dial timeouts and empty
// replies are considered fails, basically anything else constitutes a healthy upstream.

// Check is used as the up.Func in the up.Probe.
func (h *host) Check() error {
	err := h.send()
	if err != nil {
		HealthcheckFailureCount.WithLabelValues(h.addr).Add(1)
		atomic.AddUint32(&h.fails, 1)
		return err
	}

	atomic.StoreUint32(&h.fails, 0)
	return nil
}

func (h *host) send() error {
	hcping := new(dns.Msg)
	hcping.SetQuestion(".", dns.TypeNS)
	hcping.RecursionDesired = false

	m, _, err := h.client.Exchange(hcping, h.addr)
	// If we got a header, we're alright, basically only care about I/O errors 'n stuff
	if err != nil && m != nil {
		// Silly check, something sane came back
		if m.Response || m.Opcode == dns.OpcodeQuery {
			err = nil
		}
	}

	return err
}

// down returns true is this host has more than maxfails fails.
func (h *host) down(maxfails uint32) bool {
	if maxfails == 0 {
		return false
	}

	fails := atomic.LoadUint32(&h.fails)
	return fails > maxfails
}
