package bus

import (
	"crypto/tls"
	"github.com/nsqio/go-nsq"
	"log"
	"time"
)

const (
	defaultWriteTimeout = 10 * time.Second
)

// ConfigureTLS configures the publisher regarding NSQ.
func (p *PublisherConfig) ConfigureNSQ() {
	if p.NSQ == nil {
		p.NSQ = nsq.NewConfig()
	}

	if p.NSQ.WriteTimeout == 0 {
		p.NSQ.WriteTimeout = defaultWriteTimeout
	}

	ConfigureTLS(p.NSQ, p.TLS)
}

// ConfigureTLS configures the given NSQ configuration for TLS connections.
func ConfigureTLS(nsqCfg *nsq.Config, tlsCfg *TLSConfig) {
	if tlsCfg.Inactive() {
		return
	}

	nsqCfg.TlsV1 = true
	err := nsqCfg.Set("tls_root_ca_file", tlsCfg.CACertFile)
	if err != nil {
		log.Panic(err)
	}
	nsqCfg.TlsConfig.InsecureSkipVerify = false
	nsqCfg.TlsConfig.GetClientCertificate = func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		return LoadCertificate(tlsCfg.ClientCertFile)
	}
}

// CreateNSQConfig creates and configures a TLS enabled (if given TLS config != nil) NSQ configuration.
func CreateNSQConfig(tlsCfg *TLSConfig) *nsq.Config {
	cfg := nsq.NewConfig()
	ConfigureTLS(cfg, tlsCfg)
	return cfg
}
